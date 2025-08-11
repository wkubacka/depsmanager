import { Component, OnInit, computed, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule, ReactiveFormsModule, FormControl, Validators } from '@angular/forms';
import { BaseChartDirective } from 'ng2-charts';
import { ChartConfiguration } from 'chart.js';
import { DepsApiService } from './deps-api.service';
import { Dependency, ListDependenciesResponse, Project } from './models';
import { of } from 'rxjs';
import { debounceTime, distinctUntilChanged, filter, switchMap, catchError, tap, map } from 'rxjs/operators';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [CommonModule, FormsModule, ReactiveFormsModule, BaseChartDirective],
  templateUrl: './app.html',
  styleUrls: ['./app.css'],
})
export class AppComponent implements OnInit {
  // Fetch form
  projectInput = new FormControl('', { nonNullable: true });
  versionCtrl  = new FormControl<string>('', { nonNullable: true, validators: [Validators.required] });
  versions: string[] = [];
  isLoadingVersions = false;
  versionsError = '';
  isFetching = false;
  fetchError = '';

  // Projects (signals)
  projects = signal<Project[]>([]);
  isLoadingProjects = false;
  projectsError = '';
  search = signal('');

  updatingRow: Record<string, boolean> = {};
  updatingErr: Record<string, string> = {};

  // Row versions (per project)
  projectVersions: Record<string, string[]> = {};
  projectVersionSel: Record<string, string>  = {};
  loadingRowVersions: Record<string, boolean> = {};
  rowVersionsError: Record<string, string> = {};

  // Dependencies
  selectedProjectName: string | null = null;
  selectedVersion: string | null = null;
  dependencies: Dependency[] = [];
  filterCtrl = new FormControl('', { nonNullable: true });
  depsError = '';
  isLoadingDeps = false;

  // Chart
  barChartData: ChartConfiguration<'bar'>['data'] = {
    labels: [],
    datasets: [{ data: [], label: 'OpenSSF score' }],
  };
  barChartOptions: ChartConfiguration<'bar'>['options'] = {
    responsive: true,
    maintainAspectRatio: false,
    plugins: { legend: { display: true } },
    scales: { x: {}, y: { beginAtZero: true, suggestedMax: 10 } },
  };

  // 1) Projects by dependency name
  depSearchCtrl = new FormControl<string>('', { nonNullable: true, validators: [Validators.required] });
  depSearchLoading = false;
  depSearchError = '';
  projectsByDep: Project[] = [];

  // 2) Dependency names by exact score
  scoreSearchCtrl = new FormControl<number | null>(null, {
    validators: [Validators.required, Validators.min(0), Validators.max(1)],
  });
  scoreSearchLoading = false;
  scoreSearchError = '';
  depsByScore: string[] = [];

  constructor(private api: DepsApiService) {}

  ngOnInit(): void {
    this.loadProjects();

    this.filterCtrl.valueChanges.subscribe(() => this.applyFilterToChart());

    this.projectInput.valueChanges.pipe(
      debounceTime(300),
      map(v => (v || '').trim()),
      distinctUntilChanged(),
      tap(() => {
        this.versionCtrl.setValue('', { emitEvent: false });
        this.versions = [];
        this.versionsError = '';
      }),
      filter(v => v.length > 0),
      tap(() => (this.isLoadingVersions = true)),
      switchMap(name =>
        this.api.getProjectVersions(name).pipe(
          catchError(err => {
            this.versionsError = this.humanizeError(err);
            return of<string[]>([]);
          })
        )
      ),
      tap(() => (this.isLoadingVersions = false)),
    ).subscribe(list => {
      this.versions = list || [];
    });
  }

  // Projects
  refreshProjects() { this.loadProjects(); }

  loadProjects(): void {
    this.isLoadingProjects = true;
    this.projectsError = '';
    this.api.listProjects().subscribe({
      next: (data) => {
        this.isLoadingProjects = false;
        const sorted = (data || []).slice().sort((a, b) => {
          const nameCmp = a.name.localeCompare(b.name);
          if (nameCmp !== 0) return nameCmp;

          const [aMaj, aMin, aPatch] = parseSemver(a.version);
          const [bMaj, bMin, bPatch] = parseSemver(b.version);

          if (aMaj !== bMaj) return aMaj - bMaj;
          if (aMin !== bMin) return aMin - bMin;
          return aPatch - bPatch;
        });
        this.projects.set(sorted);
      },
      error: (err) => {
        this.isLoadingProjects = false;
        this.projectsError = this.humanizeError(err);
      },
    });
  }

  filteredProjects = computed(() => {
    const list = this.projects();
    const term = this.search().trim().toLowerCase();
    if (!term) return list;
    return list.filter(p =>
      p.name.toLowerCase().includes(term) ||
      (p.version ?? '').toLowerCase().includes(term)
    );
  });

  updateProject(name: string, version: string): void {
    const key = `${name}@${version}`;
    this.updatingRow[key] = true;
    this.updatingErr[key] = '';

    this.api.fetchProject(name, version).subscribe({
      next: () => {
        this.updatingRow[key] = false;
        this.loadProjects();
        if (this.selectedProjectName === name && this.selectedVersion === version) {
          this.onListDependenciesFor(name, version);
        }
      },
      error: (err) => {
        this.updatingRow[key] = false;
        this.updatingErr[key] = this.humanizeHttpError(err);
      },
    });
  }

  // FetchProject
  onFetchProject(): void {
    const name = (this.projectInput.value || '').trim();
    const version = (this.versionCtrl.value || '').trim();
    if (!name || !version) return;

    this.fetchError = '';
    this.isFetching = true;
    this.api.fetchProject(name, version).subscribe({
      next: () => {
        this.isFetching = false;
        this.projectInput.setValue('');
        this.versionCtrl.setValue('');
        this.versions = [];
        this.loadProjects();
      },
      error: (err) => {
        this.isFetching = false;
        this.fetchError = this.humanizeHttpError(err);
      },
    });
  }

  // Row versions (per project)
  loadRowVersions(projectName: string) {
    if (!projectName) return;
    this.loadingRowVersions[projectName] = true;
    this.rowVersionsError[projectName] = '';
    this.api.getProjectVersions(projectName).subscribe({
      next: (list) => {
        this.projectVersions[projectName] = list || [];
        this.projectVersionSel[projectName] = '';
        this.loadingRowVersions[projectName] = false;
      },
      error: (err) => {
        this.loadingRowVersions[projectName] = false;
        this.rowVersionsError[projectName] = this.humanizeError(err);
        this.projectVersions[projectName] = [];
        this.projectVersionSel[projectName] = '';
      }
    });
  }

  // Dependencies
  onListDependenciesFor(projectName: string, version: string) {
    this.selectedProjectName = projectName;
    this.selectedVersion = version;
    this.depsError = '';
       this.isLoadingDeps = true;

    this.api.listDependencies(projectName, version).subscribe({
      next: (resp: ListDependenciesResponse) => {
        this.isLoadingDeps = false;
        this.dependencies = (resp.dependencies || []).slice().sort((a, b) => a.name.localeCompare(b.name));
        this.applyFilterToChart();
      },
      error: (err) => {
        this.isLoadingDeps = false;
        this.dependencies = [];
        this.applyFilterToChart();
        this.depsError = this.humanizeError(err);
      },
    });
  }

  onRemoveProject(projectName: string, version: string) {
    if (!confirm(`Remove project ${projectName}@${version}?`)) return;

    this.api.removeProject(projectName, version).subscribe({
      next: () => {
        this.loadProjects();
      },
      error: (err) => {
        this.projectsError = this.humanizeHttpError(err);
      },
    });
    
  }

  // Filtering helpers
  filteredDependencies(): Dependency[] {
    const term = (this.filterCtrl.value || '').toLowerCase();
    if (!term) return this.dependencies;
    return this.dependencies.filter((d) => d.name.toLowerCase().includes(term));
  }

  applyFilterToChart(): void {
    const deps = this.filteredDependencies();
    this.barChartData = {
      labels: deps.map((d) => d.name),
      datasets: [{ data: deps.map((d) => d.score), label: 'OpenSSF score' }],
    };
  }

  searchProjectsByDependency(): void {
    this.depSearchError = '';
    this.projectsByDep = [];
    const dep = (this.depSearchCtrl.value || '').trim();
    if (!dep) {
      this.depSearchError = 'Please enter a dependency name.';
      return;
    }
    this.depSearchLoading = true;
    this.api.searchProjectsByDependencyName(dep).subscribe({
      next: (rows) => {
        this.depSearchLoading = false;
        this.projectsByDep = rows || [];
      },
      error: (err) => {
        this.depSearchLoading = false;
        const s = err?.status;
        if (s === 404) this.depSearchError = 'No projects found for this dependency.';
        else if (s === 400) this.depSearchError = 'Bad request.';
        else this.depSearchError = 'Server error. Please try again.';
      },
    });
  }

  private parseScore(val: unknown): number | null {
    if (val == null) return null;
    const s = String(val).replace(',', '.').trim();
    if (s === '') return null;
    const n = Number(s);
    return Number.isFinite(n) ? n : null;
  }

  searchDependenciesByScore(): void {
    this.scoreSearchError = '';
    this.depsByScore = [];

    const scoreParsed = this.parseScore(this.scoreSearchCtrl.value);
    if (scoreParsed == null || scoreParsed <- 10 || scoreParsed > 10) {
      this.scoreSearchError = 'Score must be a number in the range -10..10.';
      return;
    }

    this.scoreSearchLoading = true;
    this.api.searchDependenciesByScore(scoreParsed).subscribe({
      next: (names) => {
        this.scoreSearchLoading = false;
        this.depsByScore = names ?? [];
      },
      error: (err) => {
        this.scoreSearchLoading = false;
        const s = err?.status;
        if (s === 404) this.scoreSearchError = 'No dependencies found with that score.';
        else if (s === 400) this.scoreSearchError = 'Bad request.';
        else this.scoreSearchError = 'Server error. Please try again.';
      },
    });
  }

  // Utils
  formatDate(ts: number): string {
    if (!ts || ts === 0) return 'UNKNOWN';
    return new Date(ts * 1000).toLocaleString('pl-PL', { timeZone: 'Europe/Warsaw' });
  }

  humanizeHttpError(err: any): string {
    const status = err?.status;
    if (status === 404) return 'Not found (404): project does not exist or was not found.';
    if (status === 500) return 'Server error (500): please try again later.';
    return this.humanizeError(err);
  }

  humanizeError(err: any): string {
    if (err?.error && typeof err.error === 'string') return err.error;
    if (err?.message) return err.message;
    try { return JSON.stringify(err); } catch { return String(err); }
  }
}

function parseSemver(v: string): number[] {
  const parts = v.split('.').map(n => parseInt(n, 10));
  while (parts.length < 3) parts.push(0);
  return parts;
}
