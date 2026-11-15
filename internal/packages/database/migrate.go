package database

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"sort"
	"strconv"
	"strings"
)

const schemaMigrations = `
CREATE TABLE IF NOT EXISTS schema_migrations (
    version BIGINT NOT NULL PRIMARY KEY,
    dirty BOOLEAN NOT NULL
)`

// Migrator applies versioned SQL files and records the current version.
type Migrator struct {
	sql   *sql.DB
	files fs.FS
}

// NewMigrator reads migration files from a directory.
func NewMigrator(db *DB, dir string) *Migrator {
	return NewMigratorFS(db, os.DirFS(dir))
}

// NewMigratorFS reads migration files from an embedded or other filesystem.
func NewMigratorFS(db *DB, files fs.FS) *Migrator {
	sqlDB, err := db.base.DB()
	if err != nil {
		return &Migrator{files: files}
	}
	return &Migrator{sql: sqlDB, files: files}
}

// Up applies pending migrations. n is how many to apply; 0 applies all of them.
func (m *Migrator) Up(ctx context.Context, n int) error {
	current, dirty, err := m.Version(ctx)
	if err != nil {
		return err
	}
	if dirty {
		return fmt.Errorf("database: migration %d is dirty", current)
	}
	steps, err := m.pending(current)
	if err != nil {
		return err
	}
	if n > 0 && n < len(steps) {
		steps = steps[:n]
	}
	for _, step := range steps {
		if err := m.setVersion(ctx, step.version, true); err != nil {
			return err
		}
		body, err := fs.ReadFile(m.files, step.up)
		if err != nil {
			return fmt.Errorf("database: read %s: %w", step.up, err)
		}
		if err := execScript(ctx, m.sql, string(body)); err != nil {
			return fmt.Errorf("database: migrate up %d: %w", step.version, err)
		}
		if err := m.setVersion(ctx, step.version, false); err != nil {
			return err
		}
	}
	return nil
}

// Down rolls back migrations. n is how many to roll back; 0 rolls back all of them.
func (m *Migrator) Down(ctx context.Context, n int) error {
	stepsLeft := n
	for {
		current, dirty, err := m.Version(ctx)
		if err != nil {
			return err
		}
		if current == 0 {
			if n == 0 {
				return nil
			}
			return fmt.Errorf("database: no migration to roll back")
		}
		if dirty {
			return fmt.Errorf("database: migration %d is dirty", current)
		}
		step, previous, err := m.step(current)
		if err != nil {
			return err
		}
		if err := m.setVersion(ctx, current, true); err != nil {
			return err
		}
		body, err := fs.ReadFile(m.files, step.down)
		if err != nil {
			return fmt.Errorf("database: read %s: %w", step.down, err)
		}
		if err := execScript(ctx, m.sql, string(body)); err != nil {
			return fmt.Errorf("database: migrate down %d: %w", current, err)
		}
		if previous == 0 {
			if err := m.clearVersion(ctx); err != nil {
				return err
			}
		} else if err := m.setVersion(ctx, previous, false); err != nil {
			return err
		}
		if n == 0 {
			continue
		}
		stepsLeft--
		if stepsLeft == 0 {
			return nil
		}
	}
}

// Version returns the applied version. Version 0 means no migration has been applied.
func (m *Migrator) Version(ctx context.Context) (int64, bool, error) {
	if err := m.ensureTable(ctx); err != nil {
		return 0, false, err
	}
	var version int64
	var dirty int
	err := m.sql.QueryRowContext(ctx, `SELECT version, dirty FROM schema_migrations LIMIT 1`).Scan(&version, &dirty)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("database: schema version: %w", err)
	}
	return version, dirty != 0, nil
}

// Force sets the recorded version and clears the dirty flag without running SQL.
func (m *Migrator) Force(ctx context.Context, version int64) error {
	if err := m.ensureTable(ctx); err != nil {
		return err
	}
	if version == 0 {
		return m.clearVersion(ctx)
	}
	return m.setVersion(ctx, version, false)
}

type migrationStep struct {
	version int64
	up      string
	down    string
}

func (m *Migrator) ensureTable(ctx context.Context) error {
	if m.sql == nil {
		return fmt.Errorf("database: pool is closed")
	}
	if _, err := m.sql.ExecContext(ctx, schemaMigrations); err != nil {
		return fmt.Errorf("database: schema_migrations: %w", err)
	}
	return nil
}

func (m *Migrator) setVersion(ctx context.Context, version int64, dirty bool) error {
	tx, err := m.sql.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	dirtyInt := 0
	if dirty {
		dirtyInt = 1
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM schema_migrations`); err != nil {
		_ = tx.Rollback()
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (version, dirty) VALUES (?, ?)`, version, dirtyInt); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (m *Migrator) clearVersion(ctx context.Context) error {
	_, err := m.sql.ExecContext(ctx, `DELETE FROM schema_migrations`)
	return err
}

func (m *Migrator) pending(current int64) ([]migrationStep, error) {
	steps, err := migrationFiles(m.files)
	if err != nil {
		return nil, err
	}
	var pending []migrationStep
	for _, step := range steps {
		if step.version > current {
			pending = append(pending, step)
		}
	}
	return pending, nil
}

func (m *Migrator) step(current int64) (migrationStep, int64, error) {
	steps, err := migrationFiles(m.files)
	if err != nil {
		return migrationStep{}, 0, err
	}
	var previous int64
	for _, step := range steps {
		if step.version == current {
			return step, previous, nil
		}
		previous = step.version
	}
	return migrationStep{}, 0, fmt.Errorf("database: down file for version %d is missing", current)
}

func migrationFiles(files fs.FS) ([]migrationStep, error) {
	entries, err := fs.ReadDir(files, ".")
	if err != nil {
		return nil, fmt.Errorf("database: read migrations: %w", err)
	}
	byVersion := map[int64]*migrationStep{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		version, kind, ok := parseMigrationName(entry.Name())
		if !ok {
			continue
		}
		step := byVersion[version]
		if step == nil {
			step = &migrationStep{version: version}
			byVersion[version] = step
		}
		switch kind {
		case "up":
			step.up = entry.Name()
		case "down":
			step.down = entry.Name()
		}
	}
	steps := make([]migrationStep, 0, len(byVersion))
	for _, step := range byVersion {
		if step.up == "" || step.down == "" {
			return nil, fmt.Errorf("database: version %d needs both up and down files", step.version)
		}
		steps = append(steps, *step)
	}
	sort.Slice(steps, func(i, j int) bool { return steps[i].version < steps[j].version })
	return steps, nil
}

func parseMigrationName(name string) (version int64, kind string, ok bool) {
	parts := strings.SplitN(name, "_", 2)
	if len(parts) != 2 {
		return 0, "", false
	}
	version, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, "", false
	}
	switch {
	case strings.HasSuffix(name, ".up.sql"):
		return version, "up", true
	case strings.HasSuffix(name, ".down.sql"):
		return version, "down", true
	default:
		return 0, "", false
	}
}

func execScript(ctx context.Context, db *sql.DB, script string) error {
	for _, stmt := range splitSQL(script) {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

func splitSQL(script string) []string {
	var stmts []string
	var b strings.Builder
	for _, line := range strings.Split(script, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}
		b.WriteString(line)
		b.WriteByte('\n')
		if strings.HasSuffix(trimmed, ";") {
			stmt := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(b.String()), ";"))
			if stmt != "" {
				stmts = append(stmts, stmt)
			}
			b.Reset()
		}
	}
	if tail := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(b.String()), ";")); tail != "" {
		stmts = append(stmts, tail)
	}
	return stmts
}
