//go:generate ../generate_swagger.sh depsmanager api.go -p
package service

import (
	"context"
	"depsmanager"
	customErr "depsmanager/pkg/errors"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"net/http"
)

type Service interface {
	FetchAndStoreProjectDependencies(ctx context.Context, projectName, version string) error
	ListDependencies(ctx context.Context, projectName, version string) (depsmanager.ListDependenciesResponse, error)
	DeleteProject(ctx context.Context, projectName, version string) error
	ListProjects(ctx context.Context) ([]depsmanager.Project, error)
	ListProjectVersions(ctx context.Context, projectName string) ([]string, error)
	GetProjectsByDependency(ctx context.Context, depName string) ([]depsmanager.Project, error)
	GetDependenciesByExactScore(ctx context.Context, score float64) ([]string, error)

	AddDependency(ctx context.Context, projectName, version string, dep depsmanager.Dependency) error
	UpdateDependency(ctx context.Context, projectName, version string, dep depsmanager.Dependency) error
	DeleteDependency(ctx context.Context, projectName, version, depName string) error
}
type API struct {
	service Service
}

func NewAPI(service Service) API {
	return API{
		service: service,
	}
}

// GetHandler attaches chi Router as a subrouter along a routing path .
// @title DepsManager
// @version 1.0.0
// @BasePath /api
func (a *API) GetHandler() chi.Router {
	r := chi.NewRouter()
	r.Use(JSONMiddleware)

	r.Route("/api", func(r chi.Router) {
		r.Route("/v1/projects", func(r chi.Router) {
			r.Post("/", customErr.HandleError(a.FetchProject))
			r.Delete("/", customErr.HandleError(a.DeleteProject))
			r.Get("/", customErr.HandleError(a.ListProjects))
			r.Get("/versions", customErr.HandleError(a.ProjectVersions))
		})
		r.Route("/v1/dependencies", func(r chi.Router) {
			r.Post("/", customErr.HandleError(a.ListDependencies))
			r.Post("/new", customErr.HandleError(a.AddDependency))
			r.Patch("/modify", customErr.HandleError(a.ModifyDependency))
			r.Delete("/delete", customErr.HandleError(a.DeleteDependency))

			r.Post("/byprojectname", customErr.HandleError(a.ProjectByDependency))
			r.Post("/byscore", customErr.HandleError(a.DependenciesByScore))
		})
	})
	return r
}

// FetchProject
// @summary FetchProject
// @description Fetch dependencies from deps client for projectName and store them in database.
// @description If projectName already exists, update dependencies.
// @tags projects
// @accept json
// @param request r.body body depsmanager.ProjectRequest true "request body"
// @failure 500 "internal error"
// @failure 404 "not found project"
// @failure 400 "cannot decode request / body.ProjectName is required / body.Version is required"
// @Success 201 "fetched successfully"
// @Router /v1/projects [post]
func (a *API) FetchProject(w http.ResponseWriter, r *http.Request) error {
	var req depsmanager.ProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return customErr.NewBadRequest(fmt.Errorf("json.NewDecoder(r.Body).Decode(&req): %w", err))
	}

	if req.ProjectName == "" {
		return customErr.NewBadRequest(fmt.Errorf("req.ProjectName is required"))
	}

	if req.Version == "" {
		return customErr.NewBadRequest(fmt.Errorf("req.Version is required"))
	}

	if err := a.service.FetchAndStoreProjectDependencies(r.Context(), req.ProjectName, req.Version); err != nil {
		if errors.Is(err, depsmanager.ErrProjectNotFound) {
			return customErr.NewNotFound(err)
		}
		return customErr.NewInternal(fmt.Errorf("service.FetchAndStoreProjectDependencies: %w", err))
	}

	w.WriteHeader(http.StatusCreated)
	return nil
}

// DeleteProject
// @summary DeleteProject
// @description Delete project and all required dependencies.
// @tags projects
// @accept json
// @param request r.body body depsmanager.ProjectRequest true "request body"
// @failure 500 "internal error"
// @failure 404 "not found project"
// @failure 400 "cannot decode request / body.ProjectName is required / body.Version is required"
// @Success 204 "deleted successfully"
// @Router /v1/projects [delete]
func (a *API) DeleteProject(w http.ResponseWriter, r *http.Request) error {
	var req depsmanager.ProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return customErr.NewBadRequest(fmt.Errorf("json.NewDecoder(r.Body).Decode(&req): %w", err))
	}

	if req.ProjectName == "" {
		return customErr.NewBadRequest(fmt.Errorf("req.ProjectName is required"))
	}

	if req.Version == "" {
		return customErr.NewBadRequest(fmt.Errorf("req.Version is required"))
	}

	if err := a.service.DeleteProject(r.Context(), req.ProjectName, req.Version); err != nil {
		if errors.Is(err, depsmanager.ErrProjectNotFound) {
			return customErr.NewNotFound(err)
		}
		return customErr.NewInternal(fmt.Errorf("service.DeleteProject: %w", err))
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}

// ListDependencies
// @summary ListDependencies
// @description List dependencies by project name and version.
// @tags dependencies
// @accept json
// @param request r.body body depsmanager.ProjectRequest true "request body"
// @failure 500 "internal error"
// @failure 404 "not found project"
// @failure 400 "cannot decode request / body.ProjectName is required / body.Version is required"
// @Success 200 {object} depsmanager.ListDependenciesResponse "project dependencies"
// @Router /v1/dependencies [post]
func (a *API) ListDependencies(w http.ResponseWriter, r *http.Request) error {
	var req depsmanager.ProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return customErr.NewBadRequest(fmt.Errorf("json.NewDecoder(r.Body).Decode(&req): %w", err))
	}

	if req.ProjectName == "" {
		return customErr.NewBadRequest(fmt.Errorf("req.ProjectName is required"))
	}

	if req.Version == "" {
		return customErr.NewBadRequest(fmt.Errorf("req.Version is required"))
	}

	resp, err := a.service.ListDependencies(r.Context(), req.ProjectName, req.Version)
	if err != nil {
		if errors.Is(err, depsmanager.ErrProjectNotFound) {
			return customErr.NewNotFound(err)
		}
		return customErr.NewInternal(fmt.Errorf("service.ListDependencies: %w", err))
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		return customErr.NewInternal(fmt.Errorf("json.NewEncoder(w).Encode(resp)"))
	}

	return nil
}

// ListProjects
// @summary ListProjects
// @description List all projects stored in the database. To add a project with dependencies, use FetchProject.
// @tags projects
// @failure 500 "internal error"
// @Success 200 {object} []depsmanager.Project "projects"
// @Router /v1/projects [get]
func (a *API) ListProjects(w http.ResponseWriter, r *http.Request) error {
	projects, err := a.service.ListProjects(r.Context())
	if err != nil {
		return customErr.NewInternal(fmt.Errorf("service.ListProjects: %w", err))
	}

	if err := json.NewEncoder(w).Encode(projects); err != nil {
		return customErr.NewInternal(fmt.Errorf("json.NewEncoder(w).Encode(resp)"))
	}

	return nil
}

// ProjectVersions
// @summary ProjectVersions
// @description List all versions of project, by using deps.dev API.
// @tags projects
// @param project_name query string true "project name"
// @failure 500 "internal error"
// @failure 404 "not found project"
// @Success 200 {object} []string "versions"
// @Router /v1/projects/versions [get]
func (a *API) ProjectVersions(w http.ResponseWriter, r *http.Request) error {
	projectName := r.URL.Query().Get("project_name")
	if projectName == "" {
		return customErr.NewBadRequest(fmt.Errorf("project_name is required"))
	}

	versions, err := a.service.ListProjectVersions(r.Context(), projectName)
	if err != nil {
		if errors.Is(err, depsmanager.ErrProjectNotFound) {
			return customErr.NewNotFound(err)
		}
		return customErr.NewInternal(fmt.Errorf("service.ListProjectVersions: %w", err))
	}

	if err := json.NewEncoder(w).Encode(versions); err != nil {
		return customErr.NewInternal(fmt.Errorf("json.NewEncoder(w).Encode(resp)"))
	}

	return nil
}

// ProjectByDependency
// @summary ProjectByDependency
// @description List projects by dependency name.
// @tags dependencies
// @accept json
// @param request r.body body depsmanager.GetProjectNameByDepNameReq true "request body"
// @failure 500 "internal error"
// @failure 404 "not found project"
// @failure 400 "cannot decode body / empty dependency name"
// @Success 200 {object} []depsmanager.Project "related projects"
// @Router /v1/dependencies/byprojectname [post]
func (a *API) ProjectByDependency(w http.ResponseWriter, r *http.Request) error {
	var req depsmanager.GetProjectNameByDepNameReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return customErr.NewBadRequest(fmt.Errorf("json.NewDecoder(r.Body).Decode(&req): %w", err))
	}
	if req.DependencyName == "" {
		return customErr.NewBadRequest(fmt.Errorf("req.DependencyName is required"))
	}

	versions, err := a.service.GetProjectsByDependency(r.Context(), req.DependencyName)
	if err != nil {
		if errors.Is(err, depsmanager.ErrProjectNotFound) {
			return customErr.NewNotFound(err)
		}
		return customErr.NewInternal(fmt.Errorf("service.GetProjectsByDependency: %w", err))
	}

	if err := json.NewEncoder(w).Encode(versions); err != nil {
		return customErr.NewInternal(fmt.Errorf("json.NewEncoder(w).Encode(resp)"))
	}

	return nil
}

// DependenciesByScore
// @summary DependenciesByScore
// @description Get dependencies by score.
// @tags dependencies
// @accept json
// @param request r.body body depsmanager.GetDependenciesByScore true "request body"
// @failure 500 "internal error"
// @failure 404 "not found project"
// @failure 400 "cannot decode body"
// @Success 200 {object} []string "versions"
// @Router /v1/dependencies/byscore [post]
func (a *API) DependenciesByScore(w http.ResponseWriter, r *http.Request) error {
	var req depsmanager.GetDependenciesByScore
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return customErr.NewBadRequest(fmt.Errorf("json.NewDecoder(r.Body).Decode(&req): %w", err))
	}

	versions, err := a.service.GetDependenciesByExactScore(r.Context(), req.Score)
	if err != nil {
		if errors.Is(err, depsmanager.ErrProjectNotFound) {
			return customErr.NewNotFound(err)
		}
		return customErr.NewInternal(fmt.Errorf("service.GetDependenciesByExactScore: %w", err))
	}

	if err := json.NewEncoder(w).Encode(versions); err != nil {
		return customErr.NewInternal(fmt.Errorf("json.NewEncoder(w).Encode(resp)"))
	}

	return nil
}

// AddDependency
// @summary AddDependency
// @description Add manually dependency to storage for project.
// @tags dependencies
// @accept json
// @param request r.body body depsmanager.DependencyRequest true "request body"
// @failure 500 "internal error"
// @failure 409 "dependency with this name already exists"
// @failure 404 "not found project"
// @failure 400 "cannot decode body"
// @Success 201 "created"
// @Router /v1/dependencies/new [post]
func (a *API) AddDependency(w http.ResponseWriter, r *http.Request) error {
	var req depsmanager.DependencyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return customErr.NewBadRequest(fmt.Errorf("json.NewDecoder(r.Body).Decode(&req): %w", err))
	}

	if req.ProjectName == "" || req.Version == "" || req.DependencyName == "" {
		return customErr.NewBadRequest(fmt.Errorf("req.ProjectName or req.Version or req.DependencyName is required"))
	}

	if err := a.service.AddDependency(r.Context(), req.ProjectName, req.Version, depsmanager.Dependency{
		Score: req.Score,
		Name:  req.DependencyName,
	}); err != nil {
		if errors.Is(err, depsmanager.ErrProjectNotFound) {
			return customErr.NewNotFound(err)
		}
		if errors.Is(err, depsmanager.ErrDependencyAlreadyExists) {
			return customErr.NewConflict(err)
		}
		return customErr.NewInternal(fmt.Errorf("service.AddDependency: %w", err))
	}

	w.WriteHeader(http.StatusCreated)
	return nil
}

// ModifyDependency
// @summary ModifyDependency
// @description Modify manually dependency for project.
// @tags dependencies
// @accept json
// @param request r.body body depsmanager.DependencyRequest true "request body"
// @failure 500 "internal error"
// @failure 404 "not found project"
// @failure 400 "cannot decode body"
// @Success 200 "modified"
// @Router /v1/dependencies/modify [patch]
func (a *API) ModifyDependency(w http.ResponseWriter, r *http.Request) error {
	var req depsmanager.DependencyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return customErr.NewBadRequest(fmt.Errorf("json.NewDecoder(r.Body).Decode(&req): %w", err))
	}

	if req.ProjectName == "" || req.Version == "" || req.DependencyName == "" {
		return customErr.NewBadRequest(fmt.Errorf("req.ProjectName or req.Version or req.DependencyName is required"))
	}

	if err := a.service.UpdateDependency(r.Context(), req.ProjectName, req.Version, depsmanager.Dependency{
		Score: req.Score,
		Name:  req.DependencyName,
	}); err != nil {
		if errors.Is(err, depsmanager.ErrProjectNotFound) {
			return customErr.NewNotFound(err)
		}
		return customErr.NewInternal(fmt.Errorf("service.UpdateDependency: %w", err))
	}

	w.WriteHeader(http.StatusOK)
	return nil
}

// DeleteDependency
// @summary DeleteDependency
// @description Delete dependency for project
// @tags dependencies
// @accept json
// @param request r.body body depsmanager.RemoveDependencyRequest true "request body"
// @failure 500 "internal error"
// @failure 404 "not found project"
// @failure 400 "cannot decode body"
// @Success 204 "deleted"
// @Router /v1/dependencies/delete [delete]
func (a *API) DeleteDependency(w http.ResponseWriter, r *http.Request) error {
	var req depsmanager.RemoveDependencyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return customErr.NewBadRequest(fmt.Errorf("json.NewDecoder(r.Body).Decode(&req): %w", err))
	}

	if req.ProjectName == "" || req.Version == "" || req.DependencyName == "" {
		return customErr.NewBadRequest(fmt.Errorf("req.ProjectName or req.Version or req.DependencyName is required"))
	}

	if err := a.service.DeleteDependency(r.Context(), req.ProjectName, req.Version, req.DependencyName); err != nil {
		if errors.Is(err, depsmanager.ErrProjectNotFound) {
			return customErr.NewNotFound(err)
		}
		return customErr.NewInternal(fmt.Errorf("service.DeleteDependency: %w", err))
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}

func JSONMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}
