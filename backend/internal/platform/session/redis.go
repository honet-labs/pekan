package session

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	maxFailedAttempts = 5
	lockoutDuration   = 15 * time.Minute
)

type RedisStore struct {
	client *redis.Client
}

func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{client: client}
}

func (s *RedisStore) lockoutKey(email, tenantID string) string {
	return fmt.Sprintf("lockout:%s:%s", tenantID, email)
}

func (s *RedisStore) denylistKey(tokenID string) string {
	return fmt.Sprintf("denylist:%s", tokenID)
}

func (s *RedisStore) IsAccountLocked(ctx context.Context, email, tenantID string) (bool, time.Duration, error) {
	if s.client == nil {
		return false, 0, nil
	}
	key := s.lockoutKey(email, tenantID)
	
	// Use a strict timeout for Redis operations to prevent hanging
	redisCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	count, err := s.client.Get(redisCtx, key).Int()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, 0, nil
		}
		// If Redis is down/slow, we log it but allow the login to proceed
		return false, 0, nil 
	}

	if count >= maxFailedAttempts {
		ttl := s.client.TTL(ctx, key).Val()
		if ttl <= 0 {
			ttl = lockoutDuration
		}
		return true, ttl, nil
	}
	return false, 0, nil
}

func (s *RedisStore) RecordFailedLogin(ctx context.Context, email, tenantID string) (bool, time.Duration, error) {
	if s.client == nil {
		return false, 0, nil // Degrade gracefully if no Redis
	}
	key := s.lockoutKey(email, tenantID)

	redisCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	pipe := s.client.TxPipeline()
	incr := pipe.Incr(redisCtx, key)
	pipe.Expire(redisCtx, key, lockoutDuration)
	_, err := pipe.Exec(redisCtx)
	if err != nil {
		return false, 0, err
	}

	count := incr.Val()
	if count >= maxFailedAttempts {
		ttl := s.client.TTL(ctx, key).Val()
		if ttl <= 0 {
			ttl = lockoutDuration
		}
		return true, ttl, nil
	}
	return false, 0, nil
}

func (s *RedisStore) ClearFailedLogin(ctx context.Context, email, tenantID string) error {
	if s.client == nil {
		return nil
	}
	key := s.lockoutKey(email, tenantID)
	redisCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return s.client.Del(redisCtx, key).Err()
}

func (s *RedisStore) RevokeToken(ctx context.Context, tokenID string, expiration time.Duration) error {
	if s.client == nil {
		return nil
	}
	if expiration <= 0 {
		return nil
	}
	key := s.denylistKey(tokenID)
	redisCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return s.client.Set(redisCtx, key, "revoked", expiration).Err()
}

func (s *RedisStore) IsTokenRevoked(ctx context.Context, tokenID string) (bool, error) {
	if s.client == nil {
		return false, nil
	}
	key := s.denylistKey(tokenID)
	redisCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	err := s.client.Get(redisCtx, key).Err()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
