package database

import (
	"context"
	"fmt"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type txKey struct{}

// Repo is the handle repositories use. Writes are autocommit unless the
// context came from Transactor.Run. Close shuts down the shared pool.
type Repo struct {
	base *gorm.DB
}

// Transactor is the handle services use to start a transaction.
// It uses the same pool as Repo.
type Transactor struct {
	base *gorm.DB
}

// New opens one pool and returns two handles over it.
func New(cfg Config) (*Repo, *Transactor, error) {
	gormDB, err := gorm.Open(sqlite.Open(buildDSN(cfg)), &gorm.Config{
		SkipDefaultTransaction: true,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("database: open: %w", err)
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, nil, fmt.Errorf("database: pool: %w", err)
	}
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, nil, fmt.Errorf("database: ping: %w", err)
	}

	return &Repo{base: gormDB}, &Transactor{base: gormDB}, nil
}

// Close closes the shared pool.
func (r *Repo) Close() error {
	sqlDB, err := r.base.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// GORM returns the transaction stored in ctx, or the plain pool when there is none.
func (r *Repo) GORM(ctx context.Context) *gorm.DB {
	if tx, ok := txFromContext(ctx); ok {
		return tx.WithContext(ctx)
	}
	return r.base.WithContext(ctx)
}

// Run starts a transaction and passes a context that Repo.GORM will see.
// A Run inside a Run uses a savepoint. The transaction rolls back when fn returns an error.
func (t *Transactor) Run(ctx context.Context, fn func(ctx context.Context) error) error {
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
	return tx, ok && tx != nil
}

func buildDSN(cfg Config) string {
	pragmas := fmt.Sprintf(
		"_pragma=foreign_keys(1)&_pragma=busy_timeout(%d)&_pragma=journal_mode(WAL)", cfg.BusyTimeout.Milliseconds(),
	)

	if cfg.Path == ":memory:" {
		return "file::memory:?cache=shared&" + pragmas
	}
	return "file:" + cfg.Path + "?" + pragmas
}
