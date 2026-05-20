package middleware

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

var rateLimitLua = redis.NewScript(`
local current = redis.call("INCR", KEYS[1])
if current == 1 then
  redis.call("PEXPIRE", KEYS[1], ARGV[1])
end
local ttl = redis.call("PTTL", KEYS[1])
return { current, ttl }
`)

type RedisRateLimitStore struct {
	client   *redis.Client
	prefix   string
	fallback RateLimitStore
}

func NewRedisRateLimitStore(redisURL, prefix string, fallback RateLimitStore) (*RedisRateLimitStore, error) {
	if redisURL == "" {
		return nil, errors.New("RATE_LIMIT_REDIS_URL is required")
	}
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("invalid RATE_LIMIT_REDIS_URL: %w", err)
	}
	client := redis.NewClient(opts)
	return &RedisRateLimitStore{
		client:   client,
		prefix:   prefix,
		fallback: fallback,
	}, nil
}

func (s *RedisRateLimitStore) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, time.Duration, error) {
	if limit <= 0 {
		limit = 20
	}
	if window <= 0 {
		window = time.Minute
	}

	if strings.TrimSpace(s.prefix) == "" {
		s.prefix = "pekan:ratelimit"
	}
	redisKey := s.prefix + ":" + key
	windowMillis := window.Milliseconds()
	if windowMillis <= 0 {
		windowMillis = 1000
	}

	result, err := rateLimitLua.Run(ctx, s.client, []string{redisKey}, windowMillis).Result()
	if err != nil {
		if s.fallback != nil {
			return s.fallback.Allow(ctx, key, limit, window)
		}
		return false, time.Second, err
	}

	values, ok := result.([]interface{})
	if !ok || len(values) < 2 {
		if s.fallback != nil {
			return s.fallback.Allow(ctx, key, limit, window)
		}
		return false, time.Second, errors.New("invalid redis rate limit response")
	}

	count, err := toInt64(values[0])
	if err != nil {
		if s.fallback != nil {
			return s.fallback.Allow(ctx, key, limit, window)
		}
		return false, time.Second, err
	}
	ttlMs, err := toInt64(values[1])
	if err != nil {
		if s.fallback != nil {
			return s.fallback.Allow(ctx, key, limit, window)
		}
		return false, time.Second, err
	}

	if count > int64(limit) {
		retryAfter := time.Duration(ttlMs) * time.Millisecond
		if retryAfter <= 0 {
			retryAfter = window
		}
		return false, retryAfter, nil
	}
	return true, 0, nil
}

func (s *RedisRateLimitStore) Close() error {
	if s == nil || s.client == nil {
		return nil
	}
	return s.client.Close()
}

func (s *RedisRateLimitStore) Client() *redis.Client {
	if s == nil {
		return nil
	}
	return s.client
}

func toInt64(v interface{}) (int64, error) {
	switch t := v.(type) {
	case int64:
		return t, nil
	case int:
		return int64(t), nil
	case uint64:
		return int64(t), nil
	case string:
		var out int64
		_, err := fmt.Sscan(t, &out)
		if err != nil {
			return 0, fmt.Errorf("invalid integer value: %v", v)
		}
		return out, nil
	default:
		return 0, fmt.Errorf("unsupported integer type: %T", v)
	}
}
