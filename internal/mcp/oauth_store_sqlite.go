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
	"strings"
	"syscall"
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

// newSQLiteOAuthStore opens the OAuth SQLite database at path, falling back to
// writable locations when the preferred path is not usable (common when a
// root-owned PVC is mounted over /data without fsGroup for the nonroot user).
func newSQLiteOAuthStore(ctx context.Context, path string) (*sqliteOAuthStore, error) {
	candidates := oauthSQLitePathCandidates(path)
	if len(candidates) == 0 {
		return nil, errors.New("OAuth SQLite path is required")
	}

	var errs []error
	for i, candidate := range candidates {
		store, err := openSQLiteOAuthStoreAt(ctx, candidate)
		if err == nil {
			if i > 0 {
				// Preferred path failed (typically permission denied on /data).
				// Ephemeral /tmp keeps the process healthy; fix PVC ownership for persistence.
				fmt.Fprintf(os.Stderr, "pipeops-mcp: OAuth SQLite path %q is not usable; using %q instead (sessions may not persist across restarts). Fix volume ownership (fsGroup 65532) or set PIPEOPS_OAUTH_SQLITE_PATH to a writable path. First error: %v\n",
					candidates[0], candidate, errors.Join(errs...))
			}
			return store, nil
		}
		errs = append(errs, fmt.Errorf("%s: %w", candidate, err))
	}
	return nil, fmt.Errorf("open OAuth SQLite store: no writable path among %v: %w", candidates, errors.Join(errs...))
}

// oauthSQLitePathCandidates returns preferred path first, then always-writable
// fallbacks for restricted container volumes.
func oauthSQLitePathCandidates(preferred string) []string {
	preferred = strings.TrimSpace(preferred)
	if preferred == "" {
		preferred = defaultSQLitePath
	}

	seen := make(map[string]struct{}, 4)
	out := make([]string, 0, 4)
	add := func(p string) {
		p = strings.TrimSpace(p)
		if p == "" {
			return
		}
		abs, err := filepath.Abs(p)
		if err != nil {
			abs = p
		}
		if _, ok := seen[abs]; ok {
			return
		}
		seen[abs] = struct{}{}
		out = append(out, abs)
	}

	add(preferred)
	add(filepath.Join(os.TempDir(), "pipeops-mcp-oauth", "pipeops-mcp-oauth.db"))
	if home, err := os.UserHomeDir(); err == nil {
		add(filepath.Join(home, ".pipeops-mcp", "oauth", "pipeops-mcp-oauth.db"))
	}
	return out
}

func openSQLiteOAuthStoreAt(ctx context.Context, absolutePath string) (*sqliteOAuthStore, error) {
	if absolutePath == "" {
		return nil, errors.New("OAuth SQLite path is required")
	}
	directory := filepath.Dir(absolutePath)
	if err := ensureOAuthSQLiteDirectory(directory); err != nil {
		return nil, err
	}
	// Explicit write probe before opening the driver so fallback can run when
	// the directory exists but is not writable by this process (root-owned PVC).
	if err := assertWritableDirectory(directory); err != nil {
		return nil, fmt.Errorf("OAuth SQLite directory %q is not writable: %w", directory, err)
	}
	file, err := os.OpenFile(absolutePath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("create OAuth SQLite database: %w", err)
	}
	// Best-effort: PVC/CSI mount points and some container security policies
	// reject chmod even when the process can create and write the database.
	if err := os.Chmod(absolutePath, 0o600); err != nil && !isPermissionError(err) {
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

// ensureOAuthSQLiteDirectory creates the OAuth data directory and tightens
// permissions to 0700 when the platform allows it. Kubernetes volume mount
// roots often return EPERM on chmod even when the container can write; in that
// case we accept a non-private mode only if the directory is writable by the
// process (tokens are still encrypted at rest when encryption is configured).
func ensureOAuthSQLiteDirectory(directory string) error {
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("create OAuth SQLite directory: %w", err)
	}
	chmodErr := os.Chmod(directory, 0o700)

	directoryInfo, err := os.Stat(directory)
	if err != nil {
		return fmt.Errorf("inspect OAuth SQLite directory: %w", err)
	}
	if !directoryInfo.IsDir() {
		return fmt.Errorf("OAuth SQLite directory %q is not a directory", directory)
	}
	if runtime.GOOS == "windows" {
		return nil
	}

	perm := directoryInfo.Mode().Perm()
	if perm&0o077 == 0 {
		return nil
	}
	// Directory is group/other accessible. Fail hard only when chmod failed for
	// a reason other than platform denial, or when the path is not writable.
	if chmodErr != nil && !isPermissionError(chmodErr) {
		return fmt.Errorf("secure OAuth SQLite directory permissions: %w", chmodErr)
	}
	if err := assertWritableDirectory(directory); err != nil {
		if chmodErr != nil {
			return fmt.Errorf("secure OAuth SQLite directory permissions: %w (directory also not writable: %v)", chmodErr, err)
		}
		return fmt.Errorf("OAuth SQLite directory %q must be private (mode 0700) or writable by the process: %w", directory, err)
	}
	// chmod not permitted (common on volume mounts) but directory is usable.
	return nil
}

func isPermissionError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, os.ErrPermission) {
		return true
	}
	return errors.Is(err, syscall.EPERM) || errors.Is(err, syscall.EACCES)
}

func assertWritableDirectory(directory string) error {
	probe, err := os.CreateTemp(directory, ".oauth-write-check-*")
	if err != nil {
		return err
	}
	name := probe.Name()
	if err := probe.Close(); err != nil {
		_ = os.Remove(name)
		return err
	}
	if err := os.Remove(name); err != nil {
		return err
	}
	return nil
}
