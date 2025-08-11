package storage

import (
	"context"
	"database/sql"
	"depsmanager"
	"depsmanager/pkg/diff"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

type Storage struct {
	db *sqlx.DB
}

func NewStorage(conf depsmanager.SQLLiteConfig) (*Storage, error) {
	db, err := sqlx.Open("sqlite3", fmt.Sprintf("%s?_foreign_keys=on&_busy_timeout=%v", conf.DBPath, conf.BusyTimeout))
	if err != nil {
		return nil, fmt.Errorf("sqlx.Open(): %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("db.Ping(): %w", err)
	}

	if err := createTable(db); err != nil {
		return nil, err
	}

	return &Storage{db: db}, nil
}

func (s *Storage) StoreDependencies(ctx context.Context, deps depsmanager.ProjectDependencyRecord) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("s.db.BeginTx(): %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	exec, err := tx.Exec("INSERT OR IGNORE INTO projects(name, updated_at, version) VALUES (?, ?, ?);",
		deps.Project.Name, deps.Project.UpdatedAt, deps.Project.Version)
	if err != nil {
		return fmt.Errorf("tx.Exec(project): %w", err)
	}

	rowsAffected, err := exec.RowsAffected()
	if err != nil {
		return fmt.Errorf("exec.RowsAffected(): %w", err)
	}
	// if no rows affected means that we got project in store already
	if rowsAffected == 0 {
		// close transaction to avoid database lock
		if err := tx.Rollback(); err != nil {
			return fmt.Errorf("tx.Rollback(): %w", err)
		}
		if err := s.UpdateProject(ctx, deps); err != nil {
			return fmt.Errorf("s.UpdateProject(project): %w", err)
		}
		return nil
	}

	id, err := exec.LastInsertId()
	if err != nil {
		return fmt.Errorf("exec.LastInsertId(): %w", err)
	}

	preparedDependency, err := tx.PrepareContext(ctx, "INSERT INTO dependency(project_id, dependency_name, score, updated_at) VALUES (?, ?, ?, ?)")
	if err != nil {
		return fmt.Errorf("tx.PrepareContext(): %w", err)
	}
	defer preparedDependency.Close()

	for _, dependency := range deps.Dependencies {
		_, err = preparedDependency.Exec(id, dependency.Name, dependency.Score, dependency.UpdatedAt)
		if err != nil {
			return fmt.Errorf("preparedDependency.Exec(): %w, dependencyName: %v", err, dependency.Name)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("tx.Commit(): %w", err)
	}

	return nil
}

func (s *Storage) UpdateProject(ctx context.Context, deps depsmanager.ProjectDependencyRecord) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("s.db.BeginTx(): %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// get project id
	projectId, err := s.getProjectIDTX(ctx, tx, deps.Project.Name, deps.Project.Version)
	if err != nil {
		return fmt.Errorf("s.getProjectIDTX(): %w", err)
	}

	// Fetch current dependencies from DB
	rows, err := tx.QueryContext(ctx, "SELECT dependency_name, score, updated_at FROM dependency WHERE project_id = ?", projectId)
	if err != nil {
		return fmt.Errorf("tx.QueryContext(): %w", err)
	}
	defer rows.Close()

	// get current dependencies for comparison
	var currentDeps []depsmanager.Dependency
	for rows.Next() {
		var dep depsmanager.Dependency
		if err := rows.Scan(&dep.Name, &dep.Score, &dep.UpdatedAt); err != nil {
			return fmt.Errorf("rows.Scan(&name, &score): %w", err)
		}
		currentDeps = append(currentDeps, dep)
	}
	if err = rows.Err(); err != nil {
		return fmt.Errorf("rows.Err(): %w", err)
	}

	// update timestamp
	_, err = tx.Exec("UPDATE projects SET updated_at = ? WHERE id = ?",
		deps.Project.UpdatedAt, projectId)
	if err != nil {
		return fmt.Errorf("tx.Exec(project): %w", err)
	}

	// Compare with new dependencies
	toDel, toAdd := diff.DiffDependencies(currentDeps, deps.Dependencies)

	// nothing changed
	if len(toAdd) == 0 && len(toDel) == 0 {
		if err = tx.Commit(); err != nil {
			return fmt.Errorf("tx.Commit(): %w", err)
		}
		return nil
	}

	// Delete old dependencies
	if err := s.deleteDependenciesFromProject(ctx, tx, projectId, toDel); err != nil {
		return fmt.Errorf("s.deleteDependenciesFromProject(): %w", err)
	}

	// Insert new dependencies
	stmt, err := tx.PrepareContext(ctx, "INSERT INTO dependency(project_id, dependency_name, score, updated_at) VALUES (?, ?, ?, ?)")
	if err != nil {
		return fmt.Errorf("tx.PrepareContext(): %w", err)
	}
	defer stmt.Close()

	for _, dep := range toAdd {
		_, err = stmt.Exec(projectId, dep.Name, dep.Score, dep.UpdatedAt)
		if err != nil {
			return fmt.Errorf("stmt.Exec(projectId, dep.Name, dep.Score): %w", err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("tx.Commit(): %w", err)
	}

	return nil
}

func (s *Storage) DeleteProject(ctx context.Context, projectName, version string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("s.db.BeginTx(): %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	projectId, err := s.getProjectIDTX(ctx, tx, projectName, version)
	if err != nil {
		return fmt.Errorf("s.getProjectIDTX(): %w", err)
	}

	execContext, err := tx.ExecContext(ctx, "DELETE FROM projects WHERE id = ?", projectId)
	if err != nil {
		return fmt.Errorf("tx.ExecContext(delete project): %w", err)
	}

	_, err = execContext.RowsAffected()
	if err != nil {
		return fmt.Errorf("execContext.RowsAffected(): %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("tx.Commit(): %w", err)
	}

	return nil
}

func (s *Storage) ListProjectDependencies(ctx context.Context, projectName, version string) ([]depsmanager.Dependency, error) {
	projectId, err := s.getProjectID(ctx, projectName, version)
	if err != nil {
		return nil, fmt.Errorf("s.getProjectID(): %w", err)
	}

	dependenciesRows, err := s.db.QueryContext(ctx, "SELECT dependency_name, score, updated_at FROM dependency WHERE project_id = ?", projectId)
	if err != nil {
		return nil, fmt.Errorf("s.db.QueryContext(): %w", err)
	}
	defer dependenciesRows.Close()

	result := []depsmanager.Dependency{}
	for dependenciesRows.Next() {
		var dep depsmanager.Dependency
		if err = dependenciesRows.Scan(&dep.Name, &dep.Score, &dep.UpdatedAt); err != nil {
			return nil, fmt.Errorf("rows.Scan(): %w", err)
		}
		result = append(result, dep)
	}
	if err = dependenciesRows.Err(); err != nil {
		return nil, fmt.Errorf("rows.Err(): %w", err)
	}

	return result, nil
}

func (s *Storage) ListProjects(ctx context.Context) ([]depsmanager.Project, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT name, updated_at, version FROM projects")
	if err != nil {
		return nil, fmt.Errorf("db.QueryContext(): %w", err)
	}
	defer rows.Close()

	projects := []depsmanager.Project{}
	for rows.Next() {
		var p depsmanager.Project
		if err := rows.Scan(&p.Name, &p.UpdatedAt, &p.Version); err != nil {
			return nil, fmt.Errorf("rows.Scan(): %w", err)
		}
		projects = append(projects, p)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows.Err(): %w", err)
	}

	return projects, nil
}
func (s *Storage) GetProjectsByDependency(ctx context.Context, depName string) ([]depsmanager.Project, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT DISTINCT p.name, p.updated_at, p.version
		FROM projects p
		JOIN dependency d ON d.project_id = p.id
		WHERE d.dependency_name = ?
		ORDER BY p.name, p.version
	`, depName)
	if err != nil {
		return nil, fmt.Errorf("db.QueryContext(GetProjectsByDependency): %w", err)
	}
	defer rows.Close()

	var projects []depsmanager.Project
	for rows.Next() {
		var p depsmanager.Project
		if err := rows.Scan(&p.Name, &p.UpdatedAt, &p.Version); err != nil {
			return nil, fmt.Errorf("rows.Scan(project): %w", err)
		}
		projects = append(projects, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows.Err(): %w", err)
	}
	if len(projects) == 0 {
		return nil, depsmanager.ErrProjectNotFound
	}

	return projects, nil
}

func (s *Storage) GetDependenciesByExactScore(ctx context.Context, score float64) ([]string, error) {
	const eps = 1e-9
	low, high := score-eps, score+eps

	rows, err := s.db.QueryContext(ctx, `
		SELECT DISTINCT dependency_name
		FROM dependency
		WHERE score BETWEEN ? AND ?
		ORDER BY dependency_name
	`, low, high)
	if err != nil {
		return nil, fmt.Errorf("db.QueryContext(GetDependenciesByExactScore): %w", err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("rows.Scan(dependency_name): %w", err)
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows.Err(): %w", err)
	}
	return names, nil
}

func (s *Storage) AddDependency(ctx context.Context, projectName, version string, dep depsmanager.Dependency) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("db.BeginTx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	projectID, err := s.getProjectIDTX(ctx, tx, projectName, version)
	if err != nil {
		return fmt.Errorf("getProjectIDTX(%s,%s): %w", projectName, version, err)
	}

	// Insert a single dependency row
	res, err := tx.ExecContext(ctx,
		`INSERT INTO dependency(project_id, dependency_name, score, updated_at) 
         VALUES (?, ?, ?, ?)`,
		projectID, dep.Name, dep.Score, dep.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("INSERT dependency(%s): %w", dep.Name, err)
	}
	aff, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("RowsAffected: %w", err)
	}
	if aff == 0 {
		return depsmanager.ErrDependencyAlreadyExists
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("tx.Commit: %w", err)
	}
	return nil
}

func (s *Storage) DeleteDependency(ctx context.Context, projectName, version, depName string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("db.BeginTx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	projectID, err := s.getProjectIDTX(ctx, tx, projectName, version)
	if err != nil {
		return fmt.Errorf("getProjectIDTX(%s,%s): %w", projectName, version, err)
	}

	res, err := tx.ExecContext(ctx,
		`DELETE FROM dependency WHERE project_id = ? AND dependency_name = ?`,
		projectID, depName,
	)
	if err != nil {
		return fmt.Errorf("DELETE dependency(%s): %w", depName, err)
	}
	aff, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("RowsAffected: %w", err)
	}
	if aff == 0 {
		return depsmanager.ErrDependencyNotFound
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("tx.Commit: %w", err)
	}
	return nil
}

func (s *Storage) UpdateDependency(ctx context.Context, projectName, version string, dep depsmanager.Dependency) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("db.BeginTx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	projectID, err := s.getProjectIDTX(ctx, tx, projectName, version)
	if err != nil {
		return fmt.Errorf("getProjectIDTX(%s,%s): %w", projectName, version, err)
	}

	res, err := tx.ExecContext(ctx,
		`UPDATE dependency 
           SET score = ?, updated_at = ?
         WHERE project_id = ? AND dependency_name = ?`,
		dep.Score, dep.UpdatedAt, projectID, dep.Name,
	)
	if err != nil {
		return fmt.Errorf("UPDATE dependency(%s): %w", dep.Name, err)
	}
	aff, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("RowsAffected: %w", err)
	}
	if aff == 0 {
		return depsmanager.ErrDependencyNotFound
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("tx.Commit: %w", err)
	}
	return nil
}

func (s *Storage) Close() error {
	return s.db.Close()
}

func (s *Storage) deleteDependenciesFromProject(ctx context.Context, tx *sql.Tx, projectID int64, deps []depsmanager.Dependency) error {
	for _, dep := range deps {
		_, err := tx.ExecContext(ctx, "DELETE FROM dependency WHERE project_id = ? AND dependency_name = ?", projectID, dep.Name)
		if err != nil {
			return fmt.Errorf("tx.ExecContext(delete from dependency): %w", err)
		}
	}

	return nil
}

func (s *Storage) getProjectID(ctx context.Context, name, version string) (int64, error) {
	var id int64
	err := s.db.QueryRowContext(ctx,
		"SELECT id FROM projects WHERE name = ? AND version = ?", name, version,
	).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, depsmanager.ErrProjectNotFound
		}
		return 0, fmt.Errorf("getProjectID(): %w", err)
	}
	return id, nil
}

func (s *Storage) getProjectIDTX(ctx context.Context, tx *sql.Tx, name, version string) (int64, error) {
	var id int64
	err := tx.QueryRowContext(ctx,
		"SELECT id FROM projects WHERE name = ? AND version = ?", name, version,
	).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, depsmanager.ErrProjectNotFound
		}
		return 0, fmt.Errorf("tx.QueryRowContext(): %w", err)
	}
	return id, nil
}

func createTable(db *sqlx.DB) error {
	createProjectQuery := `
	CREATE TABLE IF NOT EXISTS projects (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			version TEXT NOT NULL,
			updated_at INTEGER NOT NULL,
		    UNIQUE(name, version)
		);
	`

	createDependenciesQuery := `
		CREATE TABLE IF NOT EXISTS dependency (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			project_id INTEGER NOT NULL,
			dependency_name TEXT NOT NULL,
			score REAL NOT NULL,
			updated_at INTEGER NOT NULL,
		FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
		UNIQUE(project_id, dependency_name)
		);`

	_, err := db.Exec(createProjectQuery)
	if err != nil {
		return fmt.Errorf("db.Exec(createProjectQuery): %w", err)
	}

	_, err = db.Exec(createDependenciesQuery)
	if err != nil {
		return fmt.Errorf("db.Exec(createDependenciesQuery): %w", err)
	}

	_, err = db.Exec(`
    CREATE INDEX IF NOT EXISTS idx_dependency_project_name
    ON dependency(project_id, dependency_name);
`)
	if err != nil {
		return fmt.Errorf("db.Exec(index project name): %w", err)
	}

	_, err = db.Exec(`
    CREATE INDEX IF NOT EXISTS idx_dependency_project
    ON dependency(project_id);
`)
	if err != nil {
		return fmt.Errorf("db.Exec(index project id): %w", err)
	}

	_, err = db.Exec(`
    CREATE INDEX IF NOT EXISTS idx_dependency_score ON dependency(score);
`)
	if err != nil {
		return fmt.Errorf("db.Exec(index project id): %w", err)
	}

	return nil
}
