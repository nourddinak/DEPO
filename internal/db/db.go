package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	_ "modernc.org/sqlite"
)

// DB wraps standard sql.DB with coordinated write locking to guarantee safe concurrency in SQLite WAL mode.
type DB struct {
	*sql.DB
	writeMu sync.Mutex
}

// Open initializes or connects to SQLite at the given path, enabling WAL mode and foreign keys.
func Open(dbPath string) (*DB, error) {
	dir := filepath.Dir(dbPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("creating db directory: %w", err)
		}
	}

	// SQLite connection string with WAL mode and pragmas
	dsn := fmt.Sprintf("%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)&_pragma=synchronous(NORMAL)", dbPath)

	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}

	// Optimize connection pooling for SQLite WAL: multiple concurrent readers, coordinated writer
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)

	// Ensure database pragmas are active
	pragmas := []string{
		"PRAGMA journal_mode = WAL;",
		"PRAGMA foreign_keys = ON;",
		"PRAGMA busy_timeout = 5000;",
		"PRAGMA synchronous = NORMAL;",
	}
	for _, pragma := range pragmas {
		if _, err := sqlDB.Exec(pragma); err != nil {
			sqlDB.Close()
			return nil, fmt.Errorf("executing pragma %q: %w", pragma, err)
		}
	}

	database := &DB{
		DB: sqlDB,
	}

	// Run schema migrations
	if err := database.Migrate(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	return database, nil
}

// WithWriteLock executes a write operation inside the single-writer mutex lock.
func (d *DB) WithWriteLock(fn func() error) error {
	d.writeMu.Lock()
	defer d.writeMu.Unlock()
	return fn()
}

// ExecWrite executes a modifying SQL query guarded by the write lock.
func (d *DB) ExecWrite(query string, args ...any) (sql.Result, error) {
	d.writeMu.Lock()
	defer d.writeMu.Unlock()
	return d.Exec(query, args...)
}

// Transact executes a function within a database transaction guarded by the write lock.
func (d *DB) Transact(fn func(tx *sql.Tx) error) error {
	d.writeMu.Lock()
	defer d.writeMu.Unlock()

	tx, err := d.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
