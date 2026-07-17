package mcp

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestOAuthStoreBasicContract(t *testing.T) {
	tests := []struct {
		name     string
		newStore func(*testing.T) oauthStore
	}{
		{
			name: "memory",
			newStore: func(*testing.T) oauthStore {
				return newMemoryOAuthStore()
			},
		},
		{
			name: "sqlite",
			newStore: func(t *testing.T) oauthStore {
				store, err := newSQLiteOAuthStore(context.Background(), filepath.Join(t.TempDir(), "oauth.db"))
				if err != nil {
					t.Fatalf("open SQLite store: %v", err)
				}
				t.Cleanup(func() { _ = store.Close() })
				return store
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			store := test.newStore(t)
			if err := store.Put(ctx, "record", []byte("first"), time.Minute); err != nil {
				t.Fatalf("put record: %v", err)
			}
			if err := store.Put(ctx, "record", []byte("updated"), time.Minute); err != nil {
				t.Fatalf("update record: %v", err)
			}
			value, err := store.Get(ctx, "record")
			if err != nil || string(value) != "updated" {
				t.Fatalf("get updated record: value=%q err=%v", value, err)
			}
			value, err = store.Consume(ctx, "record")
			if err != nil || string(value) != "updated" {
				t.Fatalf("consume record: value=%q err=%v", value, err)
			}
			if _, err := store.Consume(ctx, "record"); !errors.Is(err, errOAuthRecordNotFound) {
				t.Fatalf("consume used record error = %v", err)
			}
			if err := store.Put(ctx, "delete", []byte("value"), time.Minute); err != nil {
				t.Fatalf("put delete record: %v", err)
			}
			if err := store.Delete(ctx, "delete"); err != nil {
				t.Fatalf("delete record: %v", err)
			}
			if _, err := store.Get(ctx, "delete"); !errors.Is(err, errOAuthRecordNotFound) {
				t.Fatalf("deleted record error = %v", err)
			}
		})
	}
}

func TestSQLiteOAuthStorePersistsAndExpiresRecords(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "oauth.db")
	store, err := newSQLiteOAuthStore(ctx, path)
	if err != nil {
		t.Fatalf("open SQLite store: %v", err)
	}

	if err := store.Put(ctx, "persistent", []byte("value"), time.Hour); err != nil {
		t.Fatalf("put persistent record: %v", err)
	}
	if runtime.GOOS != "windows" {
		assertPrivateSQLiteFiles(t, path)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close SQLite store: %v", err)
	}

	store, err = newSQLiteOAuthStore(ctx, path)
	if err != nil {
		t.Fatalf("reopen SQLite store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	value, err := store.Get(ctx, "persistent")
	if err != nil || string(value) != "value" {
		t.Fatalf("persistent record after reopen: value=%q err=%v", value, err)
	}

	now := time.Now()
	store.now = func() time.Time { return now }
	if err := store.Put(ctx, "expiring", []byte("short"), time.Minute); err != nil {
		t.Fatalf("put expiring record: %v", err)
	}
	now = now.Add(2 * time.Minute)
	if _, err := store.Get(ctx, "expiring"); !errors.Is(err, errOAuthRecordNotFound) {
		t.Fatalf("expired record error = %v", err)
	}
	if err := store.Put(ctx, "cleanup-trigger", []byte("value"), time.Hour); err != nil {
		t.Fatalf("trigger expired cleanup: %v", err)
	}
	var count int
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM oauth_records WHERE key = ?`, "expiring").Scan(&count); err != nil {
		t.Fatalf("count expired records: %v", err)
	}
	if count != 0 {
		t.Fatalf("expired record count = %d, want 0", count)
	}

}

func TestSQLiteOAuthStoreSecuresPublicDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX directory permissions are not available on Windows")
	}
	directory := t.TempDir()
	if err := os.Chmod(directory, 0o755); err != nil {
		t.Fatalf("make test directory public: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(directory, 0o700) })

	store, err := newSQLiteOAuthStore(context.Background(), filepath.Join(directory, "oauth.db"))
	if err != nil {
		t.Fatalf("open SQLite store: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close SQLite store: %v", err)
	}
	info, err := os.Stat(directory)
	if err != nil {
		t.Fatalf("stat SQLite directory: %v", err)
	}
	if permissions := info.Mode().Perm(); permissions != 0o700 {
		t.Fatalf("secured SQLite directory permissions = %o, want 700", permissions)
	}
}

func assertPrivateSQLiteFiles(t *testing.T, path string) {
	t.Helper()
	directoryInfo, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatalf("stat SQLite directory: %v", err)
	}
	if permissions := directoryInfo.Mode().Perm(); permissions != 0o700 {
		t.Fatalf("SQLite directory permissions = %o, want 700", permissions)
	}
	for _, filename := range []string{path, path + "-wal", path + "-shm"} {
		info, err := os.Stat(filename)
		if err != nil {
			t.Fatalf("stat SQLite file %s: %v", filepath.Base(filename), err)
		}
		if permissions := info.Mode().Perm(); permissions != 0o600 {
			t.Fatalf("SQLite file %s permissions = %o, want 600", filepath.Base(filename), permissions)
		}
	}
}

func TestSQLiteOAuthStoreConsumeIsAtomic(t *testing.T) {
	ctx := context.Background()
	store, err := newSQLiteOAuthStore(ctx, filepath.Join(t.TempDir(), "oauth.db"))
	if err != nil {
		t.Fatalf("open SQLite store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Put(ctx, "single-use", []byte("value"), time.Minute); err != nil {
		t.Fatalf("put single-use record: %v", err)
	}

	var successes atomic.Int32
	var unexpected atomic.Int32
	var wait sync.WaitGroup
	for range 32 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			value, err := store.Consume(ctx, "single-use")
			switch {
			case err == nil && string(value) == "value":
				successes.Add(1)
			case errors.Is(err, errOAuthRecordNotFound):
			default:
				unexpected.Add(1)
			}
		}()
	}
	wait.Wait()
	if successes.Load() != 1 || unexpected.Load() != 0 {
		t.Fatalf("atomic consume successes=%d unexpected=%d", successes.Load(), unexpected.Load())
	}
}

func TestSQLiteOAuthStoreRefreshRotationAndRevocation(t *testing.T) {
	ctx := context.Background()
	store, err := newSQLiteOAuthStore(ctx, filepath.Join(t.TempDir(), "oauth.db"))
	if err != nil {
		t.Fatalf("open SQLite store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Put(ctx, "refresh:old", []byte("old-record"), time.Minute); err != nil {
		t.Fatalf("put old refresh: %v", err)
	}
	if err := store.Put(ctx, "family", []byte("refresh:old"), time.Minute); err != nil {
		t.Fatalf("put refresh family: %v", err)
	}

	var rotations atomic.Int32
	var unexpected atomic.Int32
	var wait sync.WaitGroup
	for range 16 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			rotated, err := store.RotateRefresh(
				ctx,
				"refresh:old",
				"refresh:used",
				"family",
				"refresh:new",
				[]byte("used-record"),
				[]byte("new-record"),
				time.Minute,
			)
			if err != nil {
				unexpected.Add(1)
				return
			}
			if rotated {
				rotations.Add(1)
			}
		}()
	}
	wait.Wait()
	if rotations.Load() != 1 || unexpected.Load() != 0 {
		t.Fatalf("atomic rotations=%d unexpected=%d", rotations.Load(), unexpected.Load())
	}
	if value, err := store.Get(ctx, "refresh:used"); err != nil || string(value) != "used-record" {
		t.Fatalf("used refresh tombstone: value=%q err=%v", value, err)
	}
	if err := store.RevokeRefreshFamily(ctx, "family"); err != nil {
		t.Fatalf("revoke refresh family: %v", err)
	}
	if _, err := store.Get(ctx, "refresh:new"); !errors.Is(err, errOAuthRecordNotFound) {
		t.Fatalf("active refresh survived family revocation: %v", err)
	}
}

func TestRedisOAuthStoreRefreshRotation(t *testing.T) {
	rawURL := os.Getenv("PIPEOPS_TEST_REDIS_URL")
	if rawURL == "" {
		t.Skip("PIPEOPS_TEST_REDIS_URL is not set")
	}
	ctx := context.Background()
	store, err := newRedisOAuthStore(ctx, rawURL)
	if err != nil {
		t.Fatalf("connect Redis store: %v", err)
	}
	defer func() { _ = store.client.Close() }()

	suffix, err := randomOAuthValue("test_", 12)
	if err != nil {
		t.Fatalf("generate test suffix: %v", err)
	}
	oldKey := "refresh:" + suffix
	usedKey := "refresh-used:" + suffix
	newKey := "refresh:new:" + suffix
	familyKey := "refresh-family:" + suffix
	for _, key := range []string{oldKey, usedKey, newKey, familyKey} {
		key := key
		t.Cleanup(func() { _ = store.Delete(ctx, key) })
	}

	if err := store.Put(ctx, oldKey, []byte("old-record"), time.Minute); err != nil {
		t.Fatalf("store old refresh: %v", err)
	}
	if err := store.Put(ctx, familyKey, []byte(oldKey), time.Minute); err != nil {
		t.Fatalf("store refresh family: %v", err)
	}
	rotated, err := store.RotateRefresh(
		ctx,
		oldKey,
		usedKey,
		familyKey,
		newKey,
		[]byte("used-record"),
		[]byte("new-record"),
		time.Minute,
	)
	if err != nil || !rotated {
		t.Fatalf("rotate refresh: rotated=%v err=%v", rotated, err)
	}
	if _, err := store.Get(ctx, oldKey); !errors.Is(err, errOAuthRecordNotFound) {
		t.Fatalf("old refresh still exists: %v", err)
	}
	if value, err := store.Get(ctx, usedKey); err != nil || string(value) != "used-record" {
		t.Fatalf("used tombstone: value=%q err=%v", value, err)
	}
	if value, err := store.Get(ctx, newKey); err != nil || string(value) != "new-record" {
		t.Fatalf("new refresh: value=%q err=%v", value, err)
	}

	if err := store.RevokeRefreshFamily(ctx, familyKey); err != nil {
		t.Fatalf("revoke refresh family: %v", err)
	}
	if _, err := store.Get(ctx, newKey); !errors.Is(err, errOAuthRecordNotFound) {
		t.Fatalf("active refresh survived family revocation: %v", err)
	}
}
