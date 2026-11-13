package database

import (
	"context"
	"errors"
	"testing"
)

func TestInitMemoryEnablesForeignKeys(t *testing.T) {
	db := openMemory(t)
	var on int
	if err := db.Repo.GORM(context.Background()).Raw("PRAGMA foreign_keys").Scan(&on).Error; err != nil {
		t.Fatal(err)
	}
	if on != 1 {
		t.Fatalf("foreign_keys = %d", on)
	}
}

func TestTransactionRollsBackAndCommits(t *testing.T) {
	db := openMemory(t)
	ctx := context.Background()
	if err := db.Repo.GORM(ctx).Exec("CREATE TABLE items (id INTEGER PRIMARY KEY, name TEXT NOT NULL)").Error; err != nil {
		t.Fatal(err)
	}

	err := db.Transactor.Run(ctx, func(ctx context.Context) error {
		if err := db.Repo.GORM(ctx).Exec("INSERT INTO items (name) VALUES ('gone')").Error; err != nil {
			return err
		}
		return errors.New("rollback")
	})
	if err == nil {
		t.Fatal("expected rollback error")
	}
	if n := countItems(t, db, ctx); n != 0 {
		t.Fatalf("rolled back row still visible: %d", n)
	}

	err = db.Transactor.Run(ctx, func(ctx context.Context) error {
		if err := db.Repo.GORM(ctx).Exec("INSERT INTO items (name) VALUES ('kept')").Error; err != nil {
			return err
		}
		if n := countItems(t, db, ctx); n != 1 {
			t.Fatalf("row not visible inside transaction: %d", n)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if n := countItems(t, db, ctx); n != 1 {
		t.Fatalf("committed rows = %d", n)
	}
}

func TestNestedTransactionUsesSavepoint(t *testing.T) {
	db := openMemory(t)
	ctx := context.Background()
	if err := db.Repo.GORM(ctx).Exec("CREATE TABLE items (id INTEGER PRIMARY KEY, name TEXT NOT NULL)").Error; err != nil {
		t.Fatal(err)
	}

	err := db.Transactor.Run(ctx, func(ctx context.Context) error {
		if err := db.Repo.GORM(ctx).Exec("INSERT INTO items (name) VALUES ('outer')").Error; err != nil {
			return err
		}
		innerErr := db.Transactor.Run(ctx, func(ctx context.Context) error {
			if err := db.Repo.GORM(ctx).Exec("INSERT INTO items (name) VALUES ('inner')").Error; err != nil {
				return err
			}
			return errors.New("inner")
		})
		if innerErr == nil {
			t.Fatal("expected inner error")
		}
		if n := countItems(t, db, ctx); n != 1 {
			t.Fatalf("savepoint rollback removed the outer row: %d", n)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if n := countItems(t, db, ctx); n != 1 {
		t.Fatalf("committed rows = %d", n)
	}
}

func TestOutsideTransactionIsImmediate(t *testing.T) {
	db := openMemory(t)
	ctx := context.Background()
	if err := db.Repo.GORM(ctx).Exec("CREATE TABLE items (id INTEGER PRIMARY KEY, name TEXT NOT NULL)").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Repo.GORM(ctx).Exec("INSERT INTO items (name) VALUES ('plain')").Error; err != nil {
		t.Fatal(err)
	}
	if n := countItems(t, db, ctx); n != 1 {
		t.Fatalf("rows = %d", n)
	}
}

func openMemory(t *testing.T) *DB {
	t.Helper()
	db, err := Init(Config{Path: ":memory:"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func countItems(t *testing.T, db *DB, ctx context.Context) int {
	t.Helper()
	var n int
	if err := db.Repo.GORM(ctx).Raw("SELECT COUNT(*) FROM items").Scan(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}
