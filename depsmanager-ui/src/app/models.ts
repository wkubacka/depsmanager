export interface Dependency {
  name: string;
  score: number;
  updated_at: number;
}

export interface ListDependenciesResponse {
  project_name: string;
  dependencies: Dependency[];
}

export interface ProjectRequest {
  project_name: string;
  version: string;
}

export interface Project {
  name: string;
  version: string;
  updated_at: number;
}

export type ProjectVersionsResponse = string[];
