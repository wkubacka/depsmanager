package depsmanager

import (
	"errors"
	"time"
)

var (
	ErrProjectNotFound         = errors.New("project not found")
	ErrDependencyNotFound      = errors.New("dependency not found")
	ErrDependencyAlreadyExists = errors.New("dependency already exists")
)

type ProjectRequest struct {
	ProjectName string `json:"project_name"`
	Version     string `json:"version"`
}

type ListDependenciesResponse struct {
	ProjectName  string       `json:"project_name"`
	Dependencies []Dependency `json:"dependencies"`
}

type ProjectDependencyRecord struct {
	Project      Project
	Dependencies []Dependency
}

type GetProjectNameByDepNameReq struct {
	DependencyName string `json:"dependency_name"`
}

type GetDependenciesByScore struct {
	Score float64 `json:"score"`
}
type Dependency struct {
	ProjectID int64   `json:"-"`
	Score     float64 `json:"score"`
	Name      string  `json:"name"`
	UpdatedAt int64   `json:"updated_at"`
}

type DependencyRequest struct {
	ProjectName    string  `json:"project_name"`
	Version        string  `json:"Version"`
	Score          float64 `json:"score"`
	DependencyName string  `json:"dependency_name"`
}

type RemoveDependencyRequest struct {
	ProjectName    string `json:"project_name"`
	Version        string `json:"Version"`
	DependencyName string `json:"dependency_name"`
}

type Project struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	UpdatedAt int64  `json:"updated_at"`
}

type ListProjectsResponse struct {
	ProjectName string `json:"project_name"`
	UpdatedAt   int64  `json:"updated_at"`
}

type DepsGetScorecardResp struct {
	Scorecard struct {
		Date   time.Time `json:"date"`
		Checks []struct {
			Name  string `json:"name"`
			Score int    `json:"score"`
		} `json:"checks"`
	} `json:"scorecard"`
}

type DepsGetVersionResp struct {
	Versions []struct {
		VersionKey struct {
			Version string `json:"version"`
		} `json:"versionKey"`
		IsDefault bool `json:"isDefault"`
	} `json:"versions"`
}

type DepsProjectMetadataResp struct {
	RelatedProjects []struct {
		ProjectKey struct {
			ID string `json:"id"`
		} `json:"projectKey"`
		RelationType string `json:"relationType"`
	} `json:"relatedProjects"`
}

type DepsProjectDependenciesResp struct {
	Nodes []struct {
		VersionKey struct {
			System  string `json:"system"`
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"versionKey"`
		Relation string `json:"relation"`
	} `json:"nodes"`
}

type DepsGetVersionsBatchResp struct {
	Responses []struct {
		Version struct {
			VersionKey struct {
				Name string `json:"name"`
			} `json:"versionKey"`
			RelatedProjects []struct {
				ProjectKey struct {
					ID string `json:"id"`
				} `json:"projectKey"`
				RelationType string `json:"relationType"`
			} `json:"relatedProjects"`
		} `json:"version"`
	} `json:"responses"`
}

type DepsGetVersionsBatchReq struct {
	Requests []interface{} `json:"requests"`
}

type DepsGetVersionKey struct {
	VersionKey ProjectDependencies `json:"versionKey"`
}

type ProjectDependencies struct {
	System  string `json:"system"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

type DependencyInfo struct {
	DependencyNames []string
}

type DepsProjectBatch struct {
	ProjectKey struct {
		ID string `json:"id"`
	} `json:"projectKey"`
}

type DepsGetProjectBatchResp struct {
	Responses []struct {
		Project struct {
			ProjectKey struct {
				ID string `json:"id"`
			} `json:"projectKey"`
			Scorecard struct {
				Date         time.Time `json:"date"`
				OverallScore float64   `json:"overallScore"`
			} `json:"scorecard"`
		} `json:"project"`
	} `json:"responses"`
}
