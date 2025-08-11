import { Injectable } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable } from 'rxjs';
import { environment } from '../environments/environment';
import {
  ListDependenciesResponse,
  Project,
  ProjectRequest,
  ProjectVersionsResponse,
} from './models';

@Injectable({ providedIn: 'root' })
export class DepsApiService {
  private base = environment.apiBaseUrl; // '/api'

  constructor(private http: HttpClient) {}

  /** POST /v1/projects — FetchProject */
  fetchProject(project_name: string, version: string): Observable<void> {
    const body: ProjectRequest = { project_name, version };
    return this.http.post<void>(`${this.base}/v1/projects`, body);
  }

  /** GET /v1/projects — ListProjects */
  listProjects(): Observable<Project[]> {
    return this.http.get<Project[]>(`${this.base}/v1/projects`);
  }

  /** POST /v1/dependencies — ListDependencies */
  listDependencies(project_name: string, version: string): Observable<ListDependenciesResponse> {
    const body: ProjectRequest = { project_name, version };
    return this.http.post<ListDependenciesResponse>(`${this.base}/v1/dependencies`, body);
  }

  /** DELETE /v1/projects — DeleteProject */
  removeProject(project_name: string, version: string): Observable<void> {
    const body: ProjectRequest = { project_name, version };
    return this.http.request<void>('DELETE', `${this.base}/v1/projects`, { body });
  }

  /** GET /v1/projects/versions?project_name=... — ProjectVersions) */
  getProjectVersions(project_name: string): Observable<ProjectVersionsResponse> {
    const params = new HttpParams().set('project_name', project_name);
    return this.http.get<ProjectVersionsResponse>(`${this.base}/v1/projects/versions`, { params });
  }
}
