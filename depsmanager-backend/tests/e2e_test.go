//go:build test_e2e
// +build test_e2e

package tests

import (
	"bytes"
	"depsmanager"
	"depsmanager/clients"
	"depsmanager/service"
	"depsmanager/storage"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
	"time"
)

func Test_e2e(t *testing.T) {
	var conf depsmanager.Config
	envconfig.MustProcess("", &conf)

	db, err := storage.NewStorage(conf.SQLLiteConfig)
	require.NoError(t, err)
	defer db.Close()

	svg := service.NewService(
		service.WithStorage(db),
		service.WithDepsClient(clients.NewDepsClient(conf.DepsAddress)),
		service.WithTimeNow(time.Now),
	)
	api := service.NewAPI(svg)

	handler := api.GetHandler()
	handler = attachFakeClient(handler)

	httpServer := http.Server{Addr: fmt.Sprintf(":%d", conf.HTTPPort), Handler: handler}

	go func() {
		err = httpServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("server error: %v", err)
		}
	}()

	client := http.Client{}

	req := depsmanager.ProjectRequest{ProjectName: "testproject", Version: "1.0.0"}
	reqBytes, err := json.Marshal(req)
	require.NoError(t, err)
	time.Sleep(50 * time.Millisecond)

	t.Run("FetchAndStoreProjectDependencies", func(t *testing.T) {
		request, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/v1/projects", conf.DepsAddress), bytes.NewReader(reqBytes))
		require.NoError(t, err)

		do, err := client.Do(request)
		require.NoError(t, err)
		defer do.Body.Close()

		require.Equal(t, http.StatusCreated, do.StatusCode)
	})

	t.Run("List dependencies", func(t *testing.T) {
		request, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/v1/dependencies", conf.DepsAddress), bytes.NewReader(reqBytes))
		require.NoError(t, err)

		do, err := client.Do(request)
		require.NoError(t, err)
		defer do.Body.Close()

		var resp depsmanager.ListDependenciesResponse
		err = json.NewDecoder(do.Body).Decode(&resp)
		require.NoError(t, err)

		require.Equal(t, "testproject", resp.ProjectName)
		require.Equal(t, http.StatusOK, do.StatusCode)
		require.Equal(t, 3, len(resp.Dependencies))

		want := map[string]float64{"pkg-a": 0, "pkg-b": 5, "pkg-c": 9}
		var correct int
		for _, d := range resp.Dependencies {
			if s, ok := want[d.Name]; ok && s == d.Score {
				correct++
			}
		}
		require.Equal(t, 3, correct)
	})

	t.Run("Update (re-fetch)", func(t *testing.T) {
		request, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/v1/projects", conf.DepsAddress), bytes.NewReader(reqBytes))
		require.NoError(t, err)

		do, err := client.Do(request)
		require.NoError(t, err)
		defer do.Body.Close()

		require.Equal(t, http.StatusCreated, do.StatusCode)
	})

	t.Run("ListProjects", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("%s/api/v1/projects", conf.DepsAddress))
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var projects []depsmanager.Project
		err = json.NewDecoder(resp.Body).Decode(&projects)
		require.NoError(t, err)

		require.Equal(t, 1, len(projects))
		require.Equal(t, "testproject", projects[0].Name)
	})
}

func attachFakeClient(router chi.Router) chi.Router {
	router.Get("/v3/systems/NPM/packages/{project}", func(w http.ResponseWriter, r *http.Request) {
		type versionItem struct {
			VersionKey struct {
				Version string `json:"version"`
			} `json:"versionKey"`
			IsDefault bool `json:"isDefault"`
		}
		resp := struct {
			Versions []versionItem `json:"versions"`
		}{}
		resp.Versions = []versionItem{
			{VersionKey: struct {
				Version string `json:"version"`
			}(struct{ Version string }{Version: "1.0.0"}), IsDefault: true},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	router.Get("/v3/systems/npm/packages/{project}/versions/{version}:dependencies", func(w http.ResponseWriter, r *http.Request) {
		type vk struct {
			System  string `json:"system"`
			Name    string `json:"name"`
			Version string `json:"version"`
		}
		type node struct {
			VersionKey vk     `json:"versionKey"`
			Relation   string `json:"relation"`
		}
		resp := struct {
			Nodes []node `json:"nodes"`
		}{
			Nodes: []node{
				{VersionKey: vk{System: "npm", Name: "pkg-a", Version: "1.0.0"}, Relation: "DIRECT"},
				{VersionKey: vk{System: "npm", Name: "pkg-b", Version: "2.0.0"}, Relation: "DIRECT"},
				{VersionKey: vk{System: "npm", Name: "pkg-c", Version: "3.0.0"}, Relation: "TRANSITIVE"},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	router.Post("/v3alpha/versionbatch", func(w http.ResponseWriter, r *http.Request) {
		type versionKey struct {
			Name string `json:"name"`
		}
		type projectKey struct {
			ID string `json:"id"`
		}
		type related struct {
			ProjectKey   projectKey `json:"projectKey"`
			RelationType string     `json:"relationType"`
		}
		type version struct {
			VersionKey      versionKey `json:"versionKey"`
			RelatedProjects []related  `json:"relatedProjects"`
		}
		type item struct {
			Version version `json:"version"`
		}
		resp := struct {
			Responses []item `json:"responses"`
		}{
			Responses: []item{
				{Version: version{VersionKey: versionKey{Name: "pkg-a"}, RelatedProjects: []related{{ProjectKey: projectKey{ID: "repo-1"}, RelationType: "SOURCE_REPO"}}}},
				{Version: version{VersionKey: versionKey{Name: "pkg-b"}, RelatedProjects: []related{{ProjectKey: projectKey{ID: "repo-2"}, RelationType: "SOURCE_REPO"}}}},
				{Version: version{VersionKey: versionKey{Name: "pkg-c"}, RelatedProjects: []related{{ProjectKey: projectKey{ID: "repo-3"}, RelationType: "SOURCE_REPO"}}}},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	router.Post("/v3alpha/projectbatch", func(w http.ResponseWriter, r *http.Request) {
		type projectKey struct {
			ID string `json:"id"`
		}
		type scorecard struct {
			Date         time.Time `json:"date"`
			OverallScore float64   `json:"overallScore"`
		}
		type project struct {
			ProjectKey projectKey `json:"projectKey"`
			Scorecard  scorecard  `json:"scorecard"`
		}
		type item struct {
			Project project `json:"project"`
		}
		resp := struct {
			Responses []item `json:"responses"`
		}{
			Responses: []item{
				{Project: project{ProjectKey: projectKey{ID: "repo-1"}, Scorecard: scorecard{Date: time.Unix(123456789, 0), OverallScore: 0}}},
				{Project: project{ProjectKey: projectKey{ID: "repo-2"}, Scorecard: scorecard{Date: time.Unix(123456789, 0), OverallScore: 5}}},
				{Project: project{ProjectKey: projectKey{ID: "repo-3"}, Scorecard: scorecard{Date: time.Unix(123456789, 0), OverallScore: 9}}},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	return router
}
