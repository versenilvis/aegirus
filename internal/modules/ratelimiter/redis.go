package ratelimiter

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisStrategy struct {
	client *redis.Client
}

func NewRedisStrategy(client *redis.Client) *RedisStrategy {
	return &RedisStrategy{
		client: client,
	}
}

func (r *RedisStrategy) Allow(ctx context.Context, key string, limit int, windows time.Duration) (bool, error) {
	// sliding window counter
	script := `
	local current_count = tonumber(redis.call('GET', KEYS[1]) or "0")
	local previous_count = tonumber(redis.call('GET', KEYS[2]) or "0")

	local weighted_count = math.floor((previous_count * tonumber(ARGV[2])) + current_count)

	if weighted_count >= tonumber(ARGV[1]) then
		return 0
	else
		redis.call('INCR', KEYS[1])
		if current_count == 0 then
			redis.call('EXPIRE', KEYS[1], ARGV[3])
		end
		return 1
	end
	`
	now := time.Now().UnixNano()
	windowsNano := windows.Nanoseconds()

	status, err := r.client.Eval(ctx, script, []string{key}, windowsNano, now).Int()
	if err != nil {
		return false, err
	}
	return status == 1, nil
}
