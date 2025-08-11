package service

import (
	"bytes"
	"depsmanager"
	"depsmanager/service/mocks"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func setup(t *testing.T) (http.Handler, *mocks.Service) {
	t.Helper()
	svc := new(mocks.Service)
	a := NewAPI(svc)
	return a.GetHandler(), svc
}

func doJSON(t *testing.T, h http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func TestFetchProject_Success(t *testing.T) {
	h, svc := setup(t)
	body := depsmanager.ProjectRequest{ProjectName: "react", Version: "18.3.1"}
	svc.On("FetchAndStoreProjectDependencies", mock.Anything, "react", "18.3.1").Return(nil).Once()
	rr := doJSON(t, h, http.MethodPost, "/api/v1/projects/", body)
	require.Equal(t, http.StatusCreated, rr.Code)
	svc.AssertExpectations(t)
}

func TestFetchProject_BadJSON(t *testing.T) {
	h, _ := setup(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/", bytes.NewBufferString("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestFetchProject_EmptyFields(t *testing.T) {
	h, _ := setup(t)
	body := depsmanager.ProjectRequest{}
	rr := doJSON(t, h, http.MethodPost, "/api/v1/projects/", body)
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestFetchProject_EmptyVersionOnly(t *testing.T) {
	h, _ := setup(t)
	body := depsmanager.ProjectRequest{ProjectName: "pkg-a", Version: ""}
	rr := doJSON(t, h, http.MethodPost, "/api/v1/projects/", body)
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestFetchProject_ProjectNotFound(t *testing.T) {
	h, svc := setup(t)
	body := depsmanager.ProjectRequest{ProjectName: "pkg-a", Version: "1.0.0"}
	svc.On("FetchAndStoreProjectDependencies", mock.Anything, "pkg-a", "1.0.0").Return(depsmanager.ErrProjectNotFound).Once()
	rr := doJSON(t, h, http.MethodPost, "/api/v1/projects/", body)
	require.Equal(t, http.StatusNotFound, rr.Code)
	svc.AssertExpectations(t)
}

func TestFetchProject_InternalError(t *testing.T) {
	h, svc := setup(t)
	body := depsmanager.ProjectRequest{ProjectName: "pkg-a", Version: "1.0.0"}
	svc.On("FetchAndStoreProjectDependencies", mock.Anything, "pkg-a", "1.0.0").Return(fmt.Errorf("db failure")).Once()
	rr := doJSON(t, h, http.MethodPost, "/api/v1/projects/", body)
	require.Equal(t, http.StatusInternalServerError, rr.Code)
	svc.AssertExpectations(t)
}

func TestDeleteProject_Success(t *testing.T) {
	h, svc := setup(t)
	body := depsmanager.ProjectRequest{ProjectName: "react", Version: "18.3.1"}
	svc.On("DeleteProject", mock.Anything, "react", "18.3.1").Return(nil).Once()
	rr := doJSON(t, h, http.MethodDelete, "/api/v1/projects/", body)
	require.Equal(t, http.StatusNoContent, rr.Code)
	svc.AssertExpectations(t)
}

func TestDeleteProject_BadJSON(t *testing.T) {
	h, _ := setup(t)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/", bytes.NewBufferString("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestDeleteProject_EmptyFields(t *testing.T) {
	h, _ := setup(t)
	body := depsmanager.ProjectRequest{}
	rr := doJSON(t, h, http.MethodDelete, "/api/v1/projects/", body)
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestDeleteProject_EmptyVersionOnly(t *testing.T) {
	h, _ := setup(t)
	body := depsmanager.ProjectRequest{ProjectName: "pkg-a", Version: ""}
	rr := doJSON(t, h, http.MethodDelete, "/api/v1/projects/", body)
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestDeleteProject_ProjectNotFound(t *testing.T) {
	h, svc := setup(t)
	body := depsmanager.ProjectRequest{ProjectName: "pkg-a", Version: "1.0.0"}
	svc.On("DeleteProject", mock.Anything, "pkg-a", "1.0.0").Return(depsmanager.ErrProjectNotFound).Once()
	rr := doJSON(t, h, http.MethodDelete, "/api/v1/projects/", body)
	require.Equal(t, http.StatusNotFound, rr.Code)
	svc.AssertExpectations(t)
}

func TestDeleteProject_InternalError(t *testing.T) {
	h, svc := setup(t)
	body := depsmanager.ProjectRequest{ProjectName: "pkg-a", Version: "1.0.0"}
	svc.On("DeleteProject", mock.Anything, "pkg-a", "1.0.0").Return(errors.New("boom")).Once()
	rr := doJSON(t, h, http.MethodDelete, "/api/v1/projects/", body)
	require.Equal(t, http.StatusInternalServerError, rr.Code)
	svc.AssertExpectations(t)
}

func TestListDependencies_Success(t *testing.T) {
	h, svc := setup(t)
	body := depsmanager.ProjectRequest{ProjectName: "react", Version: "18.3.1"}
	resp := depsmanager.ListDependenciesResponse{
		ProjectName: "react",
		Dependencies: []depsmanager.Dependency{
			{Name: "x", Score: 1.2, UpdatedAt: time.Now().Unix()},
		},
	}
	svc.On("ListDependencies", mock.Anything, "react", "18.3.1").Return(resp, nil).Once()
	rr := doJSON(t, h, http.MethodPost, "/api/v1/dependencies", body)
	require.Equal(t, http.StatusOK, rr.Code)
	var got depsmanager.ListDependenciesResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	assert.Equal(t, resp.ProjectName, got.ProjectName)
	require.Len(t, got.Dependencies, 1)
	assert.Equal(t, "x", got.Dependencies[0].Name)
	svc.AssertExpectations(t)
}

func TestListDependencies_BadJSON(t *testing.T) {
	h, _ := setup(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/dependencies", bytes.NewBufferString("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestListDependencies_EmptyFields(t *testing.T) {
	h, _ := setup(t)
	body := depsmanager.ProjectRequest{}
	rr := doJSON(t, h, http.MethodPost, "/api/v1/dependencies", body)
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestListDependencies_EmptyVersionOnly(t *testing.T) {
	h, _ := setup(t)
	body := depsmanager.ProjectRequest{ProjectName: "pkg-a", Version: ""}
	rr := doJSON(t, h, http.MethodPost, "/api/v1/dependencies", body)
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestListDependencies_ProjectNotFound(t *testing.T) {
	h, svc := setup(t)
	body := depsmanager.ProjectRequest{ProjectName: "pkg-a", Version: "1.0.0"}
	svc.On("ListDependencies", mock.Anything, "pkg-a", "1.0.0").Return(depsmanager.ListDependenciesResponse{}, depsmanager.ErrProjectNotFound).Once()
	rr := doJSON(t, h, http.MethodPost, "/api/v1/dependencies", body)
	require.Equal(t, http.StatusNotFound, rr.Code)
	svc.AssertExpectations(t)
}

func TestListDependencies_InternalError(t *testing.T) {
	h, svc := setup(t)
	body := depsmanager.ProjectRequest{ProjectName: "pkg-a", Version: "1.0.0"}
	svc.On("ListDependencies", mock.Anything, "pkg-a", "1.0.0").Return(depsmanager.ListDependenciesResponse{}, fmt.Errorf("db failure")).Once()
	rr := doJSON(t, h, http.MethodPost, "/api/v1/dependencies", body)
	require.Equal(t, http.StatusInternalServerError, rr.Code)
	svc.AssertExpectations(t)
}

func TestListProjects_Success(t *testing.T) {
	h, svc := setup(t)
	out := []depsmanager.Project{{Name: "a", Version: "1.0.0", UpdatedAt: 1}}
	svc.On("ListProjects", mock.Anything).Return(out, nil).Once()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/", nil)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
	var got []depsmanager.Project
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	require.Len(t, got, 1)
	assert.Equal(t, "a", got[0].Name)
	svc.AssertExpectations(t)
}

func TestProjectVersions_Success(t *testing.T) {
	h, svc := setup(t)
	svc.On("ListProjectVersions", mock.Anything, "react").Return([]string{"18.3.1", "18.2.0"}, nil).Once()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/versions?project_name=react", nil)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
	var got []string
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	assert.Equal(t, []string{"18.3.1", "18.2.0"}, got)
	svc.AssertExpectations(t)
}

func TestProjectVersions_Validation(t *testing.T) {
	h, _ := setup(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/versions", nil)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	require.Equal(t, http.StatusBadRequest, rr.Code)
}
func TestListProjects_InternalError(t *testing.T) {
	h, svc := setup(t)
	svc.On("ListProjects", mock.Anything).Return(nil, errors.New("db failure")).Once()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/", nil)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	require.Equal(t, http.StatusInternalServerError, rr.Code)
	svc.AssertExpectations(t)
}

func TestProjectVersions_NotFound(t *testing.T) {
	h, svc := setup(t)
	svc.On("ListProjectVersions", mock.Anything, "react").Return(nil, depsmanager.ErrProjectNotFound).Once()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/versions?project_name=react", nil)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	require.Equal(t, http.StatusNotFound, rr.Code)
	svc.AssertExpectations(t)
}

func TestProjectVersions_InternalError(t *testing.T) {
	h, svc := setup(t)
	svc.On("ListProjectVersions", mock.Anything, "react").Return(nil, errors.New("deps failure")).Once()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/versions?project_name=react", nil)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	require.Equal(t, http.StatusInternalServerError, rr.Code)
	svc.AssertExpectations(t)
}
func TestProjectByDependency_Success(t *testing.T) {
	h, svc := setup(t)
	body := depsmanager.GetProjectNameByDepNameReq{DependencyName: "shared"}

	out := []depsmanager.Project{
		{Name: "react", Version: "18.3.1", UpdatedAt: time.Now().Unix()},
		{Name: "vue", Version: "3.5.0", UpdatedAt: time.Now().Unix()},
	}
	svc.
		On("GetProjectsByDependency", mock.Anything, "shared").
		Return(out, nil).Once()

	rr := doJSON(t, h, http.MethodPost, "/api/v1/dependencies/byprojectname", body)
	require.Equal(t, http.StatusOK, rr.Code)

	var got []depsmanager.Project
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	require.Len(t, got, 2)
	assert.Equal(t, "react", got[0].Name)
	assert.Equal(t, "vue", got[1].Name)

	svc.AssertExpectations(t)
}

func TestProjectByDependency_BadJSON(t *testing.T) {
	h, _ := setup(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/dependencies/byprojectname", bytes.NewBufferString("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestProjectByDependency_Validation(t *testing.T) {
	h, _ := setup(t)
	body := depsmanager.GetProjectNameByDepNameReq{DependencyName: ""}
	rr := doJSON(t, h, http.MethodPost, "/api/v1/dependencies/byprojectname", body)
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestProjectByDependency_NotFound(t *testing.T) {
	h, svc := setup(t)
	body := depsmanager.GetProjectNameByDepNameReq{DependencyName: "missing"}

	svc.
		On("GetProjectsByDependency", mock.Anything, "missing").
		Return([]depsmanager.Project(nil), depsmanager.ErrProjectNotFound).Once()

	rr := doJSON(t, h, http.MethodPost, "/api/v1/dependencies/byprojectname", body)
	require.Equal(t, http.StatusNotFound, rr.Code)

	svc.AssertExpectations(t)
}

func TestProjectByDependency_InternalError(t *testing.T) {
	h, svc := setup(t)
	body := depsmanager.GetProjectNameByDepNameReq{DependencyName: "shared"}

	svc.
		On("GetProjectsByDependency", mock.Anything, "shared").
		Return([]depsmanager.Project(nil), errors.New("boom")).Once()

	rr := doJSON(t, h, http.MethodPost, "/api/v1/dependencies/byprojectname", body)
	require.Equal(t, http.StatusInternalServerError, rr.Code)

	svc.AssertExpectations(t)
}

func TestDependenciesByScore_Success(t *testing.T) {
	h, svc := setup(t)
	body := depsmanager.GetDependenciesByScore{Score: 81.5}

	svc.
		On("GetDependenciesByExactScore", mock.Anything, 81.5).
		Return([]string{"left-pad", "shared"}, nil).Once()

	rr := doJSON(t, h, http.MethodPost, "/api/v1/dependencies/byscore", body)
	require.Equal(t, http.StatusOK, rr.Code)

	var got []string
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	assert.ElementsMatch(t, []string{"left-pad", "shared"}, got)

	svc.AssertExpectations(t)
}

func TestDependenciesByScore_BadJSON(t *testing.T) {
	h, _ := setup(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/dependencies/byscore", bytes.NewBufferString("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestDependenciesByScore_NotFound(t *testing.T) {
	h, svc := setup(t)
	body := depsmanager.GetDependenciesByScore{Score: 99.99}

	svc.
		On("GetDependenciesByExactScore", mock.Anything, 99.99).
		Return([]string(nil), depsmanager.ErrProjectNotFound).Once()

	rr := doJSON(t, h, http.MethodPost, "/api/v1/dependencies/byscore", body)
	require.Equal(t, http.StatusNotFound, rr.Code)

	svc.AssertExpectations(t)
}

func TestDependenciesByScore_InternalError(t *testing.T) {
	h, svc := setup(t)
	body := depsmanager.GetDependenciesByScore{Score: 77.7}

	svc.
		On("GetDependenciesByExactScore", mock.Anything, 77.7).
		Return([]string(nil), errors.New("deps failure")).Once()

	rr := doJSON(t, h, http.MethodPost, "/api/v1/dependencies/byscore", body)
	require.Equal(t, http.StatusInternalServerError, rr.Code)

	svc.AssertExpectations(t)
}
