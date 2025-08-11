package service

import (
	"context"
	"depsmanager"
	"depsmanager/service/mocks"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func fixedNow() time.Time { return time.Unix(1_700_000_000, 0).UTC() }

func newSvc(t *testing.T) (*service, *mocks.Storage, *mocks.DepsClient) {
	t.Helper()
	st := new(mocks.Storage)
	dc := new(mocks.DepsClient)
	s := NewService(
		WithStorage(st),
		WithDepsClient(dc),
		WithTimeNow(fixedNow),
	)
	return s, st, dc
}

// --- JSON helpers ---

func depsRespFromJSON(t *testing.T, js string) *depsmanager.DepsProjectDependenciesResp {
	t.Helper()
	var resp depsmanager.DepsProjectDependenciesResp
	require.NoError(t, json.Unmarshal([]byte(js), &resp))
	return &resp
}
func versionsBatchFromJSON(t *testing.T, js string) *depsmanager.DepsGetVersionsBatchResp {
	t.Helper()
	var resp depsmanager.DepsGetVersionsBatchResp
	require.NoError(t, json.Unmarshal([]byte(js), &resp))
	return &resp
}
func projectsBatchFromJSON(t *testing.T, js string) *depsmanager.DepsGetProjectBatchResp {
	t.Helper()
	var resp depsmanager.DepsGetProjectBatchResp
	require.NoError(t, json.Unmarshal([]byte(js), &resp))
	return &resp
}

func matchRecord(project, version string, expectedDeps []depsmanager.Dependency, expectedUpdatedAt int64) interface{} {
	return mock.MatchedBy(func(rec depsmanager.ProjectDependencyRecord) bool {
		if rec.Project.Name != project || rec.Project.Version != version || rec.Project.UpdatedAt != expectedUpdatedAt {
			return false
		}
		if len(rec.Dependencies) != len(expectedDeps) {
			return false
		}
		type k struct{ n string }
		got := map[k]depsmanager.Dependency{}
		exp := map[k]depsmanager.Dependency{}
		for _, d := range rec.Dependencies {
			got[k{d.Name}] = d
		}
		for _, d := range expectedDeps {
			exp[k{d.Name}] = d
		}
		if len(got) != len(exp) {
			return false
		}
		for kk, vv := range exp {
			g, ok := got[kk]
			if !ok {
				return false
			}
			if g.Name != vv.Name || g.Score != vv.Score || g.UpdatedAt != vv.UpdatedAt {
				return false
			}
		}
		return true
	})
}

// --- FetchAndStoreProjectDependencies ---

func TestService_FetchAndStore_GetProjectDependencies_Error(t *testing.T) {
	s, _, dc := newSvc(t)
	ctx := context.Background()

	dc.On("GetProjectDependencies", ctx, SystemNPM, "p", "1.0.0").
		Return(nil, errors.New("deps error")).Once()

	err := s.FetchAndStoreProjectDependencies(ctx, "p", "1.0.0")
	require.Error(t, err)
	require.Contains(t, err.Error(), "s.depsClient.GetProjectDependencies")
	dc.AssertExpectations(t)
}

func TestService_FetchAndStore_NoDeps_SavesEmpty_WithInjectedTime(t *testing.T) {
	s, st, dc := newSvc(t)
	ctx := context.Background()

	depsJSON := `{"nodes":[{"versionKey":{"system":"npm","name":"p","version":"1.0.0"},"relation":"SELF"}]}`
	dc.On("GetProjectDependencies", ctx, SystemNPM, "p", "1.0.0").
		Return(depsRespFromJSON(t, depsJSON), nil).Once()

	st.On("StoreDependencies", ctx, matchRecord("p", "1.0.0", nil, fixedNow().Unix())).
		Return(nil).Once()

	require.NoError(t, s.FetchAndStoreProjectDependencies(ctx, "p", "1.0.0"))
	st.AssertExpectations(t)
	dc.AssertExpectations(t)
}

func TestService_FetchAndStore_VersionsBatch_Error(t *testing.T) {
	s, _, dc := newSvc(t)
	ctx := context.Background()

	depsJSON := `{"nodes":[{"versionKey":{"system":"npm","name":"a","version":"1.0.0"},"relation":"DIRECT"}]}`
	dc.On("GetProjectDependencies", ctx, SystemNPM, "p", "1.0.0").
		Return(depsRespFromJSON(t, depsJSON), nil).Once()

	dc.On("GetVersionsBatch", ctx, []depsmanager.ProjectDependencies{
		{System: SystemNPM, Name: "a", Version: "1.0.0"},
	}).Return(nil, errors.New("batch error")).Once()

	err := s.FetchAndStoreProjectDependencies(ctx, "p", "1.0.0")
	require.Error(t, err)
	dc.AssertExpectations(t)
}

func TestService_FetchAndStore_NoSourceRepo_SavesEmpty(t *testing.T) {
	s, st, dc := newSvc(t)
	ctx := context.Background()

	depsJSON := `{"nodes":[{"versionKey":{"system":"npm","name":"a","version":"1.0.0"},"relation":"DIRECT"}]}`
	dc.On("GetProjectDependencies", ctx, SystemNPM, "p", "1.0.0").
		Return(depsRespFromJSON(t, depsJSON), nil).Once()

	versionsJSON := `{"responses":[{"version":{"versionKey":{"name":"a"},"relatedProjects":[]}}]}`
	dc.On("GetVersionsBatch", ctx, []depsmanager.ProjectDependencies{
		{System: SystemNPM, Name: "a", Version: "1.0.0"},
	}).Return(versionsBatchFromJSON(t, versionsJSON), nil).Once()

	st.On("StoreDependencies", ctx, matchRecord("p", "1.0.0", nil, fixedNow().Unix())).
		Return(nil).Once()

	require.NoError(t, s.FetchAndStoreProjectDependencies(ctx, "p", "1.0.0"))
	st.AssertExpectations(t)
	dc.AssertExpectations(t)
}

func TestService_FetchAndStore_ProjectsBatch_Error(t *testing.T) {
	s, _, dc := newSvc(t)
	ctx := context.Background()

	depsJSON := `{"nodes":[{"versionKey":{"system":"npm","name":"a","version":"1.0.0"},"relation":"DIRECT"}]}`
	dc.On("GetProjectDependencies", ctx, SystemNPM, "p", "1.0.0").
		Return(depsRespFromJSON(t, depsJSON), nil).Once()

	versionsJSON := `{"responses":[{"version":{"versionKey":{"name":"a"},"relatedProjects":[{"projectKey":{"id":"repo-1"},"relationType":"SOURCE_REPO"}]}}]}`
	dc.On("GetVersionsBatch", ctx, []depsmanager.ProjectDependencies{
		{System: SystemNPM, Name: "a", Version: "1.0.0"},
	}).Return(versionsBatchFromJSON(t, versionsJSON), nil).Once()

	dc.On("GetProjectsBatch", ctx, []string{"repo-1"}).
		Return(nil, errors.New("pb error")).Once()

	err := s.FetchAndStoreProjectDependencies(ctx, "p", "1.0.0")
	require.Error(t, err)
	dc.AssertExpectations(t)
}

func TestService_FetchAndStore_Success_SavesDeps_WithInjectedTime(t *testing.T) {
	s, st, dc := newSvc(t)
	ctx := context.Background()

	depsJSON := `{"nodes":[
	  {"versionKey":{"system":"npm","name":"a","version":"1.0.0"},"relation":"DIRECT"},
	  {"versionKey":{"system":"npm","name":"b","version":"2.0.0"},"relation":"TRANSITIVE"}
	]}`
	dc.On("GetProjectDependencies", ctx, SystemNPM, "p", "1.0.0").
		Return(depsRespFromJSON(t, depsJSON), nil).Once()

	versionsJSON := `{"responses":[
	  {"version":{"versionKey":{"name":"a"},"relatedProjects":[{"projectKey":{"id":"repo-1"},"relationType":"SOURCE_REPO"}]}},
	  {"version":{"versionKey":{"name":"b"},"relatedProjects":[{"projectKey":{"id":"repo-2"},"relationType":"SOURCE_REPO"}]}}
	]}`
	dc.On("GetVersionsBatch", ctx, []depsmanager.ProjectDependencies{
		{System: SystemNPM, Name: "a", Version: "1.0.0"},
		{System: SystemNPM, Name: "b", Version: "2.0.0"},
	}).Return(versionsBatchFromJSON(t, versionsJSON), nil).Once()

	depDate := time.Unix(1_800_000_000, 0).UTC()
	pbJSON := `{"responses":[
	  {"project":{"projectKey":{"id":"repo-1"},"scorecard":{"date":"` + depDate.Format(time.RFC3339Nano) + `","overallScore":7.5}}},
	  {"project":{"projectKey":{"id":"repo-2"},"scorecard":{"date":"` + depDate.Format(time.RFC3339Nano) + `","overallScore":6.2}}}
	]}`
	dc.On("GetProjectsBatch", ctx, mock.MatchedBy(func(ids []string) bool {
		m := map[string]struct{}{}
		for _, id := range ids {
			m[id] = struct{}{}
		}
		_, ok1 := m["repo-1"]
		_, ok2 := m["repo-2"]
		return len(ids) == 2 && ok1 && ok2
	})).Return(projectsBatchFromJSON(t, pbJSON), nil).Once()

	expected := []depsmanager.Dependency{
		{Name: "a", Score: 7.5, UpdatedAt: depDate.Unix()},
		{Name: "b", Score: 6.2, UpdatedAt: depDate.Unix()},
	}
	st.On("StoreDependencies", ctx, matchRecord("p", "1.0.0", expected, fixedNow().Unix())).
		Return(nil).Once()

	require.NoError(t, s.FetchAndStoreProjectDependencies(ctx, "p", "1.0.0"))
	st.AssertExpectations(t)
	dc.AssertExpectations(t)
}

func TestService_FetchAndStore_StoreError_ReturnsError(t *testing.T) {
	s, st, dc := newSvc(t)
	ctx := context.Background()

	depsJSON := `{"nodes":[{"versionKey":{"system":"npm","name":"p","version":"1.0.0"},"relation":"SELF"}]}`
	dc.On("GetProjectDependencies", ctx, SystemNPM, "p", "1.0.0").
		Return(depsRespFromJSON(t, depsJSON), nil).Once()

	st.On("StoreDependencies", ctx, matchRecord("p", "1.0.0", nil, fixedNow().Unix())).
		Return(errors.New("store failed")).Once()

	err := s.FetchAndStoreProjectDependencies(ctx, "p", "1.0.0")
	require.Error(t, err)
	require.Contains(t, err.Error(), "s.storage.StoreDependencies")
	st.AssertExpectations(t)
	dc.AssertExpectations(t)
}

// --- ListDependencies ---

func TestService_ListDependencies_Success(t *testing.T) {
	s, st, _ := newSvc(t)
	ctx := context.Background()

	out := []depsmanager.Dependency{{Name: "x", Score: 1.2, UpdatedAt: 123}}
	st.On("ListProjectDependencies", ctx, "p", "1.0.0").
		Return(out, nil).Once()

	resp, err := s.ListDependencies(ctx, "p", "1.0.0")
	require.NoError(t, err)
	require.Equal(t, "p", resp.ProjectName)
	require.Len(t, resp.Dependencies, 1)
	require.Equal(t, "x", resp.Dependencies[0].Name)

	st.AssertExpectations(t)
}

func TestService_ListDependencies_StorageError(t *testing.T) {
	s, st, _ := newSvc(t)
	ctx := context.Background()

	st.On("ListProjectDependencies", ctx, "p", "1.0.0").
		Return(nil, errors.New("db error")).Once()

	_, err := s.ListDependencies(ctx, "p", "1.0.0")
	require.Error(t, err)
	require.Contains(t, err.Error(), "s.storage.ListProjectDependencies")
	st.AssertExpectations(t)
}

// --- DeleteProject ---

func TestService_DeleteProject_Success(t *testing.T) {
	s, st, _ := newSvc(t)
	ctx := context.Background()

	st.On("DeleteProject", ctx, "p", "1.0.0").
		Return(nil).Once()

	require.NoError(t, s.DeleteProject(ctx, "p", "1.0.0"))
	st.AssertExpectations(t)
}

func TestService_DeleteProject_StorageError(t *testing.T) {
	s, st, _ := newSvc(t)
	ctx := context.Background()

	st.On("DeleteProject", ctx, "p", "1.0.0").
		Return(errors.New("db error")).Once()

	err := s.DeleteProject(ctx, "p", "1.0.0")
	require.Error(t, err)
	require.Contains(t, err.Error(), "s.storage.DeleteProject")
	st.AssertExpectations(t)
}

// --- ListProjects ---

func TestService_ListProjects_Success(t *testing.T) {
	s, st, _ := newSvc(t)
	ctx := context.Background()

	projects := []depsmanager.Project{{Name: "a", Version: "1.0.0", UpdatedAt: 1}}
	st.On("ListProjects", ctx).Return(projects, nil).Once()

	got, err := s.ListProjects(ctx)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "a", got[0].Name)

	st.AssertExpectations(t)
}

func TestService_ListProjects_StorageError(t *testing.T) {
	s, st, _ := newSvc(t)
	ctx := context.Background()

	st.On("ListProjects", ctx).Return(nil, errors.New("db error")).Once()

	_, err := s.ListProjects(ctx)
	require.Error(t, err)
	st.AssertExpectations(t)
}

// --- ListProjectVersions ---

func TestService_ListProjectVersions_Success(t *testing.T) {
	s, _, dc := newSvc(t)
	ctx := context.Background()

	respJSON := `{"versions":[
	  {"versionKey":{"version":"2.0.0"},"isDefault":true},
	  {"versionKey":{"version":"1.0.0"},"isDefault":false}
	]}`
	var resp depsmanager.DepsGetVersionResp
	require.NoError(t, json.Unmarshal([]byte(respJSON), &resp))

	dc.On("GetProjectVersions", ctx, "p").Return(&resp, nil).Once()

	vers, err := s.ListProjectVersions(ctx, "p")
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"2.0.0", "1.0.0"}, vers)

	dc.AssertExpectations(t)
}

func TestService_ListProjectVersions_ClientError(t *testing.T) {
	s, _, dc := newSvc(t)
	ctx := context.Background()

	dc.On("GetProjectVersions", ctx, "p").Return(nil, fmt.Errorf("deps fail")).Once()

	_, err := s.ListProjectVersions(ctx, "p")
	require.Error(t, err)
	require.Contains(t, err.Error(), "s.depsClient.GetProjectVersions")
	dc.AssertExpectations(t)
}
