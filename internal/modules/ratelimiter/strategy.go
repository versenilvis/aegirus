package ratelimiter

import (
	"context"
	"time"
)

type Strategy interface {
	// Allow: check if request is allowed
	// key: ip or user id
	// limit: limit of requests (e.g: 10 req)
	// windows: how long (e.g 1sec)
	Allow(ctx context.Context, key string, limit int, windows time.Duration) (bool, error)
}
