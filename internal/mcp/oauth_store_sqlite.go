package mcp

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"time"

	_ "modernc.org/sqlite"
)

const sqliteOAuthSchema = `
CREATE TABLE IF NOT EXISTS oauth_records (
    key TEXT PRIMARY KEY,
    value BLOB NOT NULL,
    expires_at INTEGER NOT NULL
) WITHOUT ROWID
`

type sqliteOAuthStore struct {
	db  *sql.DB
	now func() time.Time
}

func newSQLiteOAuthStore(ctx context.Context, path string) (*sqliteOAuthStore, error) {
	if path == "" {
		return nil, errors.New("OAuth SQLite path is required")
	}
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve OAuth SQLite path: %w", err)
	}
	directory := filepath.Dir(absolutePath)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return nil, fmt.Errorf("create OAuth SQLite directory: %w", err)
	}
	if err := os.Chmod(directory, 0o700); err != nil {
		return nil, fmt.Errorf("secure OAuth SQLite directory permissions: %w", err)
	}
	directoryInfo, err := os.Stat(directory)
	if err != nil {
		return nil, fmt.Errorf("inspect OAuth SQLite directory: %w", err)
	}
	if !directoryInfo.IsDir() {
		return nil, fmt.Errorf("OAuth SQLite directory %q is not a directory", directory)
	}
	if runtime.GOOS != "windows" && directoryInfo.Mode().Perm()&0o077 != 0 {
		return nil, fmt.Errorf("OAuth SQLite directory %q must be private (mode 0700)", directory)
	}
	file, err := os.OpenFile(absolutePath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("create OAuth SQLite database: %w", err)
	}
	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("secure OAuth SQLite database permissions: %w", err)
	}
	if err := file.Close(); err != nil {
		return nil, fmt.Errorf("close new OAuth SQLite database: %w", err)
	}

	databaseURL := &url.URL{Scheme: "file", Path: filepath.ToSlash(absolutePath)}
	query := databaseURL.Query()
	query.Set("_txlock", "immediate")
	query.Add("_pragma", "busy_timeout(5000)")
	query.Add("_pragma", "journal_mode(WAL)")
	query.Add("_pragma", "secure_delete(ON)")
	query.Add("_pragma", "synchronous(FULL)")
	query.Set("_dqs", "false")
	databaseURL.RawQuery = query.Encode()

	db, err := sql.Open("sqlite", databaseURL.String())
	if err != nil {
		return nil, fmt.Errorf("open OAuth SQLite database: %w", err)
	}
	// OAuth records are small and infrequently written. One connection avoids
	// intra-process SQLITE_BUSY errors while BEGIN IMMEDIATE preserves atomic
	// consume and refresh-rotation behavior.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("connect to OAuth SQLite database: %w", err)
	}
	if _, err := db.ExecContext(ctx, sqliteOAuthSchema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("initialize OAuth SQLite database: %w", err)
	}
	if _, err := db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS oauth_records_expires_at ON oauth_records(expires_at)`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("index OAuth SQLite database: %w", err)
	}
	return &sqliteOAuthStore{db: db, now: time.Now}, nil
}

func (s *sqliteOAuthStore) Close() error {
	if err := s.db.Close(); err != nil {
		return fmt.Errorf("close OAuth SQLite database: %w", err)
	}
	return nil
}

func (s *sqliteOAuthStore) Put(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if _, err := s.db.ExecContext(ctx, `
INSERT INTO oauth_records(key, value, expires_at) VALUES (?, ?, ?)
ON CONFLICT(key) DO UPDATE SET value = excluded.value, expires_at = excluded.expires_at
`, key, value, s.expiresAt(ttl)); err != nil {
		return fmt.Errorf("store OAuth record: %w", err)
	}
	if err := s.cleanupExpired(ctx, s.db); err != nil {
		return err
	}
	return nil
}

func (s *sqliteOAuthStore) Get(ctx context.Context, key string) ([]byte, error) {
	var value []byte
	err := s.db.QueryRowContext(
		ctx,
		`SELECT value FROM oauth_records WHERE key = ? AND expires_at > ?`,
		key,
		s.now().UnixMilli(),
	).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errOAuthRecordNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("load OAuth record: %w", err)
	}
	return value, nil
}

func (s *sqliteOAuthStore) Consume(ctx context.Context, key string) ([]byte, error) {
	var value []byte
	err := s.db.QueryRowContext(
		ctx,
		`DELETE FROM oauth_records WHERE key = ? AND expires_at > ? RETURNING value`,
		key,
		s.now().UnixMilli(),
	).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errOAuthRecordNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("consume OAuth record: %w", err)
	}
	return value, nil
}

func (s *sqliteOAuthStore) Delete(ctx context.Context, key string) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM oauth_records WHERE key = ?`, key); err != nil {
		return fmt.Errorf("delete OAuth record: %w", err)
	}
	return nil
}

func (s *sqliteOAuthStore) RotateRefresh(
	ctx context.Context,
	oldKey, usedKey, familyKey, newKey string,
	usedValue, newValue []byte,
	ttl time.Duration,
) (bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("begin OAuth refresh rotation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	now := s.now().UnixMilli()
	expiresAt := s.expiresAt(ttl)
	result, err := tx.ExecContext(ctx, `
UPDATE oauth_records
SET value = ?, expires_at = ?
WHERE key = ? AND value = ? AND expires_at > ?
`, []byte(newKey), expiresAt, familyKey, []byte(oldKey), now)
	if err != nil {
		return false, fmt.Errorf("lock OAuth refresh family: %w", err)
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("inspect OAuth refresh family update: %w", err)
	}
	if updated != 1 {
		return false, nil
	}

	result, err = tx.ExecContext(ctx, `DELETE FROM oauth_records WHERE key = ? AND expires_at > ?`, oldKey, now)
	if err != nil {
		return false, fmt.Errorf("consume OAuth refresh token: %w", err)
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("inspect OAuth refresh consume: %w", err)
	}
	if deleted != 1 {
		return false, nil
	}

	if err := sqlitePutOAuthRecord(ctx, tx, usedKey, usedValue, expiresAt); err != nil {
		return false, err
	}
	if err := sqlitePutOAuthRecord(ctx, tx, newKey, newValue, expiresAt); err != nil {
		return false, err
	}
	if err := s.cleanupExpired(ctx, tx); err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit OAuth refresh rotation: %w", err)
	}
	return true, nil
}

func (s *sqliteOAuthStore) RevokeRefreshFamily(ctx context.Context, familyKey string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin OAuth refresh family revocation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var currentKey []byte
	err = tx.QueryRowContext(
		ctx,
		`SELECT value FROM oauth_records WHERE key = ? AND expires_at > ?`,
		familyKey,
		s.now().UnixMilli(),
	).Scan(&currentKey)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("load OAuth refresh family: %w", err)
	}
	if err == nil {
		if _, err := tx.ExecContext(ctx, `DELETE FROM oauth_records WHERE key = ?`, string(currentKey)); err != nil {
			return fmt.Errorf("revoke active OAuth refresh token: %w", err)
		}
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM oauth_records WHERE key = ?`, familyKey); err != nil {
		return fmt.Errorf("delete OAuth refresh family: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit OAuth refresh family revocation: %w", err)
	}
	return nil
}

func (s *sqliteOAuthStore) expiresAt(ttl time.Duration) int64 {
	return s.now().Add(ttl).UnixMilli()
}

type sqliteOAuthExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func sqlitePutOAuthRecord(ctx context.Context, executor sqliteOAuthExecutor, key string, value []byte, expiresAt int64) error {
	if _, err := executor.ExecContext(ctx, `
INSERT INTO oauth_records(key, value, expires_at) VALUES (?, ?, ?)
ON CONFLICT(key) DO UPDATE SET value = excluded.value, expires_at = excluded.expires_at
`, key, value, expiresAt); err != nil {
		return fmt.Errorf("store OAuth record: %w", err)
	}
	return nil
}

func (s *sqliteOAuthStore) cleanupExpired(ctx context.Context, executor sqliteOAuthExecutor) error {
	if _, err := executor.ExecContext(ctx, `DELETE FROM oauth_records WHERE expires_at <= ?`, s.now().UnixMilli()); err != nil {
		return fmt.Errorf("clean expired OAuth records: %w", err)
	}
	return nil
}
