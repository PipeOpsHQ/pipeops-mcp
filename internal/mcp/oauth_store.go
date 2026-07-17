package mcp

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"time"

	redis "github.com/redis/go-redis/v9"
)

var errOAuthRecordNotFound = errors.New("oauth record not found")

type oauthStore interface {
	Put(context.Context, string, []byte, time.Duration) error
	Get(context.Context, string) ([]byte, error)
	Consume(context.Context, string) ([]byte, error)
	Delete(context.Context, string) error
	RotateRefresh(context.Context, string, string, string, string, []byte, []byte, time.Duration) (bool, error)
	RevokeRefreshFamily(context.Context, string) error
}

type redisOAuthStore struct {
	client *redis.Client
}

func newRedisOAuthStore(ctx context.Context, rawURL string) (*redisOAuthStore, error) {
	options, err := redis.ParseURL(rawURL)
	if err != nil {
		return nil, fmt.Errorf("parse OAuth Redis URL: %w", err)
	}
	client := redis.NewClient(options)
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("connect to OAuth Redis: %w", err)
	}
	return &redisOAuthStore{client: client}, nil
}

func (s *redisOAuthStore) Put(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if err := s.client.Set(ctx, oauthRedisKey(key), value, ttl).Err(); err != nil {
		return fmt.Errorf("store OAuth record: %w", err)
	}
	return nil
}

func (s *redisOAuthStore) Get(ctx context.Context, key string) ([]byte, error) {
	value, err := s.client.Get(ctx, oauthRedisKey(key)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, errOAuthRecordNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("load OAuth record: %w", err)
	}
	return value, nil
}

func (s *redisOAuthStore) Consume(ctx context.Context, key string) ([]byte, error) {
	value, err := s.client.GetDel(ctx, oauthRedisKey(key)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, errOAuthRecordNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("consume OAuth record: %w", err)
	}
	return value, nil
}

func (s *redisOAuthStore) Delete(ctx context.Context, key string) error {
	if err := s.client.Del(ctx, oauthRedisKey(key)).Err(); err != nil {
		return fmt.Errorf("delete OAuth record: %w", err)
	}
	return nil
}

var rotateRefreshScript = redis.NewScript(`
local current = redis.call("GET", KEYS[3])
if not current or current ~= ARGV[1] or not redis.call("GET", KEYS[1]) then
  return 0
end
redis.call("DEL", KEYS[1])
redis.call("SET", KEYS[2], ARGV[2], "PX", ARGV[5])
redis.call("SET", KEYS[4], ARGV[3], "PX", ARGV[5])
redis.call("SET", KEYS[3], ARGV[4], "PX", ARGV[5])
return 1
`)

func (s *redisOAuthStore) RotateRefresh(
	ctx context.Context,
	oldKey, usedKey, familyKey, newKey string,
	usedValue, newValue []byte,
	ttl time.Duration,
) (bool, error) {
	result, err := rotateRefreshScript.Run(ctx, s.client, []string{
		oauthRedisKey(oldKey),
		oauthRedisKey(usedKey),
		oauthRedisKey(familyKey),
		oauthRedisKey(newKey),
	}, oldKey, usedValue, newValue, newKey, ttl.Milliseconds()).Int()
	if err != nil {
		return false, fmt.Errorf("rotate OAuth refresh token: %w", err)
	}
	return result == 1, nil
}

var revokeRefreshFamilyScript = redis.NewScript(`
local current = redis.call("GET", KEYS[1])
if current then
  redis.call("DEL", ARGV[1] .. current)
end
redis.call("DEL", KEYS[1])
return 1
`)

func (s *redisOAuthStore) RevokeRefreshFamily(ctx context.Context, familyKey string) error {
	if err := revokeRefreshFamilyScript.Run(
		ctx,
		s.client,
		[]string{oauthRedisKey(familyKey)},
		oauthRedisKey(""),
	).Err(); err != nil {
		return fmt.Errorf("revoke OAuth refresh family: %w", err)
	}
	return nil
}

func oauthRedisKey(key string) string {
	return "pipeops:mcp:{oauth}:" + key
}

type memoryOAuthRecord struct {
	value     []byte
	expiresAt time.Time
}

type memoryOAuthStore struct {
	mu      sync.Mutex
	records map[string]memoryOAuthRecord
	now     func() time.Time
}

func newMemoryOAuthStore() *memoryOAuthStore {
	return &memoryOAuthStore{
		records: make(map[string]memoryOAuthRecord),
		now:     time.Now,
	}
}

func (s *memoryOAuthStore) Put(_ context.Context, key string, value []byte, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records[key] = memoryOAuthRecord{
		value:     append([]byte(nil), value...),
		expiresAt: s.now().Add(ttl),
	}
	return nil
}

func (s *memoryOAuthStore) Get(_ context.Context, key string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getLocked(key, false)
}

func (s *memoryOAuthStore) Consume(_ context.Context, key string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getLocked(key, true)
}

func (s *memoryOAuthStore) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.records, key)
	return nil
}

func (s *memoryOAuthStore) RotateRefresh(
	_ context.Context,
	oldKey, usedKey, familyKey, newKey string,
	usedValue, newValue []byte,
	ttl time.Duration,
) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	old, oldOK := s.records[oldKey]
	current, familyOK := s.records[familyKey]
	if !oldOK || !old.expiresAt.After(now) || !familyOK || !current.expiresAt.After(now) || string(current.value) != oldKey {
		delete(s.records, oldKey)
		return false, nil
	}
	delete(s.records, oldKey)
	expiresAt := now.Add(ttl)
	s.records[usedKey] = memoryOAuthRecord{value: append([]byte(nil), usedValue...), expiresAt: expiresAt}
	s.records[newKey] = memoryOAuthRecord{value: append([]byte(nil), newValue...), expiresAt: expiresAt}
	s.records[familyKey] = memoryOAuthRecord{value: []byte(newKey), expiresAt: expiresAt}
	return true, nil
}

func (s *memoryOAuthStore) RevokeRefreshFamily(_ context.Context, familyKey string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if family, ok := s.records[familyKey]; ok {
		delete(s.records, string(family.value))
	}
	delete(s.records, familyKey)
	return nil
}

func (s *memoryOAuthStore) getLocked(key string, consume bool) ([]byte, error) {
	record, ok := s.records[key]
	if !ok || !record.expiresAt.After(s.now()) {
		delete(s.records, key)
		return nil, errOAuthRecordNotFound
	}
	if consume {
		delete(s.records, key)
	}
	return append([]byte(nil), record.value...), nil
}

type credentialCipher struct {
	aead cipher.AEAD
}

func newCredentialCipher(encodedKey string) (*credentialCipher, error) {
	key, err := decodeEncryptionKey(encodedKey)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create OAuth credential cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create OAuth credential AEAD: %w", err)
	}
	return &credentialCipher{aead: aead}, nil
}

func decodeEncryptionKey(value string) ([]byte, error) {
	for _, encoding := range []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
		base64.URLEncoding,
		base64.RawURLEncoding,
	} {
		decoded, err := encoding.DecodeString(value)
		if err == nil && len(decoded) == 32 {
			return decoded, nil
		}
	}
	return nil, errors.New("PIPEOPS_OAUTH_ENCRYPTION_KEY must be a base64-encoded 32-byte key")
}

func (c *credentialCipher) Seal(value string, additionalData []byte) (string, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("generate credential nonce: %w", err)
	}
	sealed := c.aead.Seal(nil, nonce, []byte(value), additionalData)
	payload := append(nonce, sealed...)
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func (c *credentialCipher) Open(value string, additionalData []byte) (string, error) {
	payload, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil || len(payload) <= c.aead.NonceSize() {
		return "", errors.New("invalid encrypted OAuth credential")
	}
	nonce := payload[:c.aead.NonceSize()]
	plaintext, err := c.aead.Open(nil, nonce, payload[c.aead.NonceSize():], additionalData)
	if err != nil {
		return "", errors.New("invalid encrypted OAuth credential")
	}
	return string(plaintext), nil
}

func oauthLookupKey(kind, token string) string {
	digest := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%s:%x", kind, digest[:])
}
