package service

import (
	"context"
	"depsmanager"
	"fmt"
	"time"
)

const SystemNPM = "npm"

type Storage interface {
	StoreDependencies(ctx context.Context, deps depsmanager.ProjectDependencyRecord) error
	DeleteProject(ctx context.Context, projectName, version string) error
	ListProjectDependencies(ctx context.Context, projectName, version string) ([]depsmanager.Dependency, error)
	ListProjects(ctx context.Context) ([]depsmanager.Project, error)
	GetDependenciesByExactScore(ctx context.Context, score float64) ([]string, error)
	GetProjectsByDependency(ctx context.Context, depName string) ([]depsmanager.Project, error)
}

type DepsClient interface {
	GetProjectVersions(ctx context.Context, project string) (*depsmanager.DepsGetVersionResp, error)
	GetProjectDependencies(ctx context.Context, system, project, version string) (*depsmanager.DepsProjectDependenciesResp, error)
	GetVersionsBatch(ctx context.Context, projects []depsmanager.ProjectDependencies) (*depsmanager.DepsGetVersionsBatchResp, error)
	GetProjectsBatch(ctx context.Context, projects []string) (*depsmanager.DepsGetProjectBatchResp, error)
}
type service struct {
	storage    Storage
	depsClient DepsClient

	tNow func() time.Time
}

func NewService(opts ...func(s *service)) *service {
	s := &service{}
	for _, opt := range opts {
		opt(s)
	}

	return s
}

func WithStorage(storage Storage) func(s *service) {
	return func(s *service) {
		s.storage = storage
	}
}

func WithDepsClient(depsClient DepsClient) func(s *service) {
	return func(s *service) {
		s.depsClient = depsClient
	}
}

func WithTimeNow(tNow func() time.Time) func(s *service) {
	return func(s *service) {
		s.tNow = tNow
	}
}

func (s *service) FetchAndStoreProjectDependencies(ctx context.Context, projectName, version string) error {
	dependencies, err := s.depsClient.GetProjectDependencies(ctx, SystemNPM, projectName, version)
	if err != nil {
		return fmt.Errorf("s.depsClient.GetProjectDependencies: %w", err)
	}

	var projectDependencies []depsmanager.ProjectDependencies
	seen := make(map[string]struct{})
	for _, deps := range dependencies.Nodes {
		if deps.Relation == "SELF" {
			continue
		}

		key := deps.VersionKey.System + deps.VersionKey.Name
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		projectDependencies = append(projectDependencies, depsmanager.ProjectDependencies{
			System:  deps.VersionKey.System,
			Name:    deps.VersionKey.Name,
			Version: deps.VersionKey.Version,
		})
	}

	// not found dependencies for project
	if len(projectDependencies) == 0 {
		if err = s.storeProjectWithDependencies(ctx, projectName, version, nil); err != nil {
			return fmt.Errorf("s.storeProjectWithDependencies(): %w", err)
		}
		return nil
	}

	batch, err := s.depsClient.GetVersionsBatch(ctx, projectDependencies)
	if err != nil {
		return err
	}

	deps := make(map[string][]string)
	for _, b := range batch.Responses {
		for _, relatedProjects := range b.Version.RelatedProjects {
			if relatedProjects.RelationType == "SOURCE_REPO" {
				deps[relatedProjects.ProjectKey.ID] = append(deps[relatedProjects.ProjectKey.ID], b.Version.VersionKey.Name)
				break
			}
		}
	}
	if len(deps) == 0 {
		// No mappable repos - store empty deps for this project/version
		if err := s.storeProjectWithDependencies(ctx, projectName, version, nil); err != nil {
			return fmt.Errorf("s.storeProjectWithDependencies(): %w", err)
		}
		return nil
	}

	var uniqueDepsLinks []string
	for k := range deps {
		uniqueDepsLinks = append(uniqueDepsLinks, k)
	}

	projectsBatch, err := s.depsClient.GetProjectsBatch(ctx, uniqueDepsLinks)
	if err != nil {
		return err
	}

	var dependencyScores []depsmanager.Dependency
	for _, pBatch := range projectsBatch.Responses {
		for id, pName := range deps {
			if pBatch.Project.ProjectKey.ID == id {
				for _, name := range pName {
					updatedAt := pBatch.Project.Scorecard.Date.Unix()
					if pBatch.Project.Scorecard.Date.IsZero() {
						updatedAt = 0
					}
					dependencyScores = append(dependencyScores, depsmanager.Dependency{
						Score:     pBatch.Project.Scorecard.OverallScore,
						Name:      name,
						UpdatedAt: updatedAt,
					})
				}
				break
			}
		}
	}

	if err = s.storeProjectWithDependencies(ctx, projectName, version, dependencyScores); err != nil {
		return fmt.Errorf("s.storeProjectWithDependencies(): %w", err)
	}

	return nil
}

func (s *service) ListDependencies(ctx context.Context, projectName, version string) (depsmanager.ListDependenciesResponse, error) {
	deps, err := s.storage.ListProjectDependencies(ctx, projectName, version)
	if err != nil {
		return depsmanager.ListDependenciesResponse{}, fmt.Errorf("s.storage.ListProjectDependencies() projectName: %s, error: %w", projectName, err)
	}

	return depsmanager.ListDependenciesResponse{
		ProjectName:  projectName,
		Dependencies: deps,
	}, nil
}

func (s *service) DeleteProject(ctx context.Context, projectName, version string) error {
	if err := s.storage.DeleteProject(ctx, projectName, version); err != nil {
		return fmt.Errorf("s.storage.DeleteProject() projectName: %s, error: %w", projectName, err)
	}

	return nil
}

func (s *service) ListProjects(ctx context.Context) ([]depsmanager.Project, error) {
	return s.storage.ListProjects(ctx)
}

func (s *service) ListProjectVersions(ctx context.Context, projectName string) ([]string, error) {
	projectVersions, err := s.depsClient.GetProjectVersions(ctx, projectName)
	if err != nil {
		return nil, fmt.Errorf("s.depsClient.GetProjectVersions() projectName: %s, error: %w", projectName, err)
	}

	var versions []string
	for _, v := range projectVersions.Versions {
		versions = append(versions, v.VersionKey.Version)
	}
	return versions, nil
}

func (s *service) GetDependenciesByExactScore(ctx context.Context, score float64) ([]string, error) {
	return s.storage.GetDependenciesByExactScore(ctx, score)
}

func (s *service) GetProjectsByDependency(ctx context.Context, depName string) ([]depsmanager.Project, error) {
	return s.storage.GetProjectsByDependency(ctx, depName)
}

func (s *service) storeProjectWithDependencies(ctx context.Context, projectName, version string, dependencyScores []depsmanager.Dependency) error {
	if err := s.storage.StoreDependencies(ctx, depsmanager.ProjectDependencyRecord{
		Project: depsmanager.Project{
			Name:      projectName,
			Version:   version,
			UpdatedAt: s.tNow().Unix(),
		},
		Dependencies: dependencyScores,
	}); err != nil {
		return fmt.Errorf("s.storage.StoreDependencies() projectName: %s, error: %w", projectName, err)
	}

	return nil
}
