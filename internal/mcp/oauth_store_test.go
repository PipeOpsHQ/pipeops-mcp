package mcp

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"
)

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
