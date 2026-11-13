package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type txKey struct{}

// DB owns one SQLite pool. Repo and Transactor share that pool.
// Close is the only shutdown path; the two handles do not close it.
type DB struct {
	Repo       Repo
	Transactor Transactor

	sql *sql.DB
}

// Repo is the handle repositories use for ordinary GORM calls.
// A write through it is not wrapped in a transaction.
type Repo struct {
	base *gorm.DB
}

// Transactor is the handle services use to start a transaction.
// It does not expose GORM.
type Transactor struct {
	base *gorm.DB
}

// Init opens the pool, checks that foreign keys are on, and pings.
// It does not migrate.
func Init(cfg Config) (*DB, error) {
	dsn, err := buildDSN(cfg)
	if err != nil {
		return nil, err
	}

	gormDB, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		SkipDefaultTransaction: true,
	})
	if err != nil {
		return nil, fmt.Errorf("database: open: %w", err)
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, fmt.Errorf("database: pool: %w", err)
	}

	maxOpen, maxIdle := poolLimits(cfg)
	sqlDB.SetMaxOpenConns(maxOpen)
	sqlDB.SetMaxIdleConns(maxIdle)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("database: ping: %w", err)
	}

	var foreignKeys int
	if err := gormDB.Raw("PRAGMA foreign_keys").Scan(&foreignKeys).Error; err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("database: foreign keys: %w", err)
	}
	if foreignKeys != 1 {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("database: foreign keys are off")
	}

	return &DB{
		Repo:       Repo{base: gormDB},
		Transactor: Transactor{base: gormDB},
		sql:        sqlDB,
	}, nil
}

// Close closes the shared pool.
func (db *DB) Close() error {
	if db == nil || db.sql == nil {
		return nil
	}
	return db.sql.Close()
}

// GORM returns the transaction stored in ctx, or the plain pool when there is none.
func (r Repo) GORM(ctx context.Context) *gorm.DB {
	if ctx == nil {
		ctx = context.Background()
	}
	if tx, ok := txFromContext(ctx); ok {
		return tx.WithContext(ctx)
	}
	return r.base.WithContext(ctx)
}

// Run starts a transaction and passes a context that Repo.GORM will see.
// A Run inside a Run uses a savepoint. The transaction rolls back when fn returns an error.
func (t Transactor) Run(ctx context.Context, fn func(ctx context.Context) error) error {
	if fn == nil {
		return fmt.Errorf("database: transaction function is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	db := t.base
	if tx, ok := txFromContext(ctx); ok {
		db = tx
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(withTx(ctx, tx))
	})
}

func withTx(ctx context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

func txFromContext(ctx context.Context) (*gorm.DB, bool) {
	tx, ok := ctx.Value(txKey{}).(*gorm.DB)
	if !ok || tx == nil {
		return nil, false
	}
	return tx, true
}

func buildDSN(cfg Config) (string, error) {
	if cfg.Path == "" {
		return "", fmt.Errorf("database: path is required")
	}
	if containsQuery(cfg.Path) {
		return "", fmt.Errorf("database: path must not contain ? or #")
	}

	timeout := cfg.BusyTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	pragmas := fmt.Sprintf(
		"_pragma=foreign_keys(1)&_pragma=busy_timeout(%d)&_pragma=journal_mode(WAL)",
		timeout.Milliseconds(),
	)

	if cfg.Path == ":memory:" {
		return "file::memory:?cache=shared&" + pragmas, nil
	}
	return "file:" + cfg.Path + "?" + pragmas, nil
}

func containsQuery(path string) bool {
	for _, r := range path {
		if r == '?' || r == '#' {
			return true
		}
	}
	return false
}

func poolLimits(cfg Config) (maxOpen, maxIdle int) {
	maxOpen = cfg.MaxOpenConns
	if maxOpen <= 0 {
		maxOpen = 1
	}
	maxIdle = cfg.MaxIdleConns
	if maxIdle <= 0 || maxIdle > maxOpen {
		maxIdle = maxOpen
	}
	return maxOpen, maxIdle
}
