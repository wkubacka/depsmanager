package storage

import (
	"context"
	"errors"
	"testing"
	"time"

	"depsmanager"
)

// Creates a new in-memory SQLite storage instance.
// Important: for ":memory:" we must set max connections to 1,
// otherwise each new connection would get a separate empty DB.
func newInMemoryStorage(t *testing.T) *Storage {
	t.Helper()
	st, err := NewStorage(depsmanager.SQLLiteConfig{DBPath: ":memory:"})
	if err != nil {
		t.Fatalf("NewStorage(:memory:): %v", err)
	}
	st.db.SetMaxOpenConns(1)
	st.db.SetMaxIdleConns(1)
	t.Cleanup(func() { _ = st.Close() })
	return st
}

// Helper to create a slice of dependencies with given names.
func makeDeps(names ...string) []depsmanager.Dependency {
	now := time.Now().Unix()
	out := make([]depsmanager.Dependency, 0, len(names))
	for i, n := range names {
		out = append(out, depsmanager.Dependency{
			Name:      n,
			Score:     float64(80+i) + 0.5, // use float to verify REAL type
			UpdatedAt: now + int64(i),
		})
	}
	return out
}

func TestInsertVersion1AndList(t *testing.T) {
	st := newInMemoryStorage(t)
	ctx := context.Background()

	rec := depsmanager.ProjectDependencyRecord{
		Project: depsmanager.Project{
			Name:      "react",
			Version:   "18.3.1",
			UpdatedAt: time.Now().Unix(),
		},
		Dependencies: makeDeps("scheduler", "loose-envify"),
	}

	if err := st.StoreDependencies(ctx, rec); err != nil {
		t.Fatalf("StoreDependencies(v1): %v", err)
	}

	// List all projects
	projects, err := st.ListProjects(ctx)
	if err != nil {
		t.Fatalf("ListProjects(): %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("expected 1 project row, got %d", len(projects))
	}
	if projects[0].Name != "react" || projects[0].Version != "18.3.1" {
		t.Fatalf("unexpected project row: %+v", projects[0])
	}

	// List dependencies for the given version
	deps, err := st.ListProjectDependencies(ctx, "react", "18.3.1")
	if err != nil {
		t.Fatalf("ListProjectDependencies(react,18.3.1): %v", err)
	}
	if len(deps) != 2 {
		t.Fatalf("expected 2 deps, got %d", len(deps))
	}
}

func TestInsertSecondVersionSameName(t *testing.T) {
	st := newInMemoryStorage(t)
	ctx := context.Background()

	// v1
	if err := st.StoreDependencies(ctx, depsmanager.ProjectDependencyRecord{
		Project:      depsmanager.Project{Name: "vue", Version: "3.4.0", UpdatedAt: time.Now().Unix()},
		Dependencies: makeDeps("typescript", "postcss"),
	}); err != nil {
		t.Fatalf("StoreDependencies(v1): %v", err)
	}

	// v2 (different version, same name) — should create a second row in `projects`
	if err := st.StoreDependencies(ctx, depsmanager.ProjectDependencyRecord{
		Project:      depsmanager.Project{Name: "vue", Version: "3.5.0", UpdatedAt: time.Now().Unix()},
		Dependencies: makeDeps("typescript", "sass"),
	}); err != nil {
		t.Fatalf("StoreDependencies(v2): %v", err)
	}

	projects, err := st.ListProjects(ctx)
	if err != nil {
		t.Fatalf("ListProjects(): %v", err)
	}
	// Expect 2 rows (both versions)
	var v1, v2 int
	for _, p := range projects {
		if p.Name == "vue" && p.Version == "3.4.0" {
			v1++
		}
		if p.Name == "vue" && p.Version == "3.5.0" {
			v2++
		}
	}
	if v1 != 1 || v2 != 1 {
		t.Fatalf("expected both versions present (v1=%d, v2=%d). Rows: %+v", v1, v2, projects)
	}
}

func TestUpdateSameVersion_DiffAddDel(t *testing.T) {
	st := newInMemoryStorage(t)
	ctx := context.Background()

	version := "1.0.0"
	name := "angular"

	// First insert: deps A, B
	if err := st.StoreDependencies(ctx, depsmanager.ProjectDependencyRecord{
		Project:      depsmanager.Project{Name: name, Version: version, UpdatedAt: time.Now().Unix()},
		Dependencies: makeDeps("rxjs", "zone.js"),
	}); err != nil {
		t.Fatalf("StoreDependencies(v1): %v", err)
	}

	// Second insert for the same version, but deps: B (keep), C (new)
	// => should trigger UpdateProject + diff: remove A, add C
	if err := st.StoreDependencies(ctx, depsmanager.ProjectDependencyRecord{
		Project:      depsmanager.Project{Name: name, Version: version, UpdatedAt: time.Now().Unix()},
		Dependencies: makeDeps("rxjs", "tslib"),
	}); err != nil {
		t.Fatalf("StoreDependencies(update same version): %v", err)
	}

	got, err := st.ListProjectDependencies(ctx, name, version)
	if err != nil {
		t.Fatalf("ListProjectDependencies(%s,%s): %v", name, version, err)
	}
	set := map[string]bool{}
	for _, d := range got {
		set[d.Name] = true
	}
	if !(set["rxjs"] && set["tslib"] && !set["zone.js"]) {
		t.Fatalf("unexpected deps after update: %+v", got)
	}
}

func TestDeleteOneVersion_LeavesOther(t *testing.T) {
	st := newInMemoryStorage(t)
	ctx := context.Background()

	// v1
	if err := st.StoreDependencies(ctx, depsmanager.ProjectDependencyRecord{
		Project:      depsmanager.Project{Name: "svelte", Version: "5.0.0", UpdatedAt: time.Now().Unix()},
		Dependencies: makeDeps("magic-string"),
	}); err != nil {
		t.Fatalf("StoreDependencies(svelte v1): %v", err)
	}
	// v2
	if err := st.StoreDependencies(ctx, depsmanager.ProjectDependencyRecord{
		Project:      depsmanager.Project{Name: "svelte", Version: "5.1.0", UpdatedAt: time.Now().Unix()},
		Dependencies: makeDeps("magic-string", "estree-walker"),
	}); err != nil {
		t.Fatalf("StoreDependencies(svelte v2): %v", err)
	}

	// Delete v1 — v2 should remain
	if err := st.DeleteProject(ctx, "svelte", "5.0.0"); err != nil {
		t.Fatalf("DeleteProject(svelte,5.0.0): %v", err)
	}

	// v1 should not exist
	if _, err := st.ListProjectDependencies(ctx, "svelte", "5.0.0"); !errors.Is(err, depsmanager.ErrProjectNotFound) {
		t.Fatalf("expected ErrProjectNotFound for deleted version, got: %v", err)
	}

	// v2 should still exist
	if deps, err := st.ListProjectDependencies(ctx, "svelte", "5.1.0"); err != nil || len(deps) == 0 {
		t.Fatalf("expected deps for remaining version, err=%v deps=%v", err, deps)
	}
}

func TestListDependencyForProject_NotFound(t *testing.T) {
	st := newInMemoryStorage(t)
	ctx := context.Background()

	_, err := st.ListProjectDependencies(ctx, "nope", "0.0.1")
	if !errors.Is(err, depsmanager.ErrProjectNotFound) {
		t.Fatalf("expected ErrProjectNotFound, got: %v", err)
	}
}

func TestDeleteProject_NotFound(t *testing.T) {
	st := newInMemoryStorage(t)
	ctx := context.Background()

	err := st.DeleteProject(ctx, "ghost", "9.9.9")
	if !errors.Is(err, depsmanager.ErrProjectNotFound) {
		t.Fatalf("expected ErrProjectNotFound, got: %v", err)
	}
}
