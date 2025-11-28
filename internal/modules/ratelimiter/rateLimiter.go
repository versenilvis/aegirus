// Package ratelimiter - Rate limiter
package ratelimiter

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"aegirus/internal/config"
	"aegirus/internal/utils"
	"aegirus/pkg/logger"

	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

type IPBanner interface {
	BanIP(ip string, duration time.Duration)
}

type Client struct {
	limiter  *rate.Limiter
	lastSeen int64
}

type RateLimiter struct {
	clients map[string]*Client
	mu      sync.RWMutex
	banip   IPBanner
	cfg     config.RateLimiterConfig

	blockedRequests int
}

func NewRateLimiter(cfg config.RateLimiterConfig, b IPBanner) *RateLimiter {
	return &RateLimiter{
		clients: make(map[string]*Client),
		cfg:     cfg,
		banip:   b,
	}
}

// ------------------- RateLimiter -------------------

func (rl *RateLimiter) getKey(ip, endpoint string) string {
	return fmt.Sprintf("%s|%s", ip, endpoint)
}

func (rl *RateLimiter) getLimiter(ip, endpoint string) *rate.Limiter {
	key := rl.getKey(ip, endpoint)

	rl.mu.RLock()
	client, exists := rl.clients[key]
	if exists {
		// NOTE: write in struct while RLock basically is light data race but with timestamp is acceptable for performance
		atomic.StoreInt64(&client.lastSeen, time.Now().UnixNano())
		rl.mu.RUnlock()
		return client.limiter
	}
	rl.mu.RUnlock()

	rl.mu.Lock()
	defer rl.mu.Unlock()

	client, exists = rl.clients[key]
	if !exists {
		// > 100k IP ~ 20MB RAM
		if len(rl.clients) > 100000 {
			rl.clients = make(map[string]*Client)
			logger.Log.Warn("RateLimiter map overflow! Resetting all clients to prevent memory leak.")
		}

		client = &Client{
			limiter:  rate.NewLimiter(rate.Limit(rl.cfg.RefillRate), rl.cfg.Burst),
			lastSeen: time.Now().UnixNano(),
		}
		rl.clients[key] = client
	} else {
		atomic.StoreInt64(&client.lastSeen, time.Now().UnixNano())
	}

	return client.limiter
}

// RateLimiterMiddleware :
// ab -n 20 -c 1 http://localhost:8080/api/v1/download
// https://httpd.apache.org/docs/2.4/programs/ab.html
// windows user: https://www.apachelounge.com/download/
func (rl *RateLimiter) RateLimiterMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		ip := utils.GetRealIP(c)
		endpoint := c.Path()
		limiter := rl.getLimiter(ip, endpoint)
		if !limiter.Allow() {
			rl.mu.Lock()
			rl.blockedRequests++
			rl.mu.Unlock()

			if rl.banip != nil {
				rl.banip.BanIP(ip, rl.cfg.BanDuration)
			}
			logger.Log.Warn("Rate limit exceeded",
				zap.String("ip", ip),
				zap.String("endpoint", endpoint),
				zap.Time("time", time.Now()),
			)

			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":   "Too many requests",
				"message": "Try again later",
			})
		}
		return c.Next()
	}
}

func (rl *RateLimiter) Cleanup(ctx context.Context) {
	logger.Log.Info("Starting rate limiter cleanup routine",
		zap.Duration("cleanup_interval_seconds", rl.cfg.CleanupInterval),
		zap.Duration("client_ttl_seconds", rl.cfg.ClientTTL),
	)

	ticker := time.NewTicker(rl.cfg.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Log.Info("Rate limiter cleanup stopped")
			return
		case <-ticker.C:
			rl.mu.Lock()
			removed := 0
			now := time.Now()
			for key, client := range rl.clients {
				lastSeenTime := time.Unix(0, atomic.LoadInt64(&client.lastSeen))
				if now.Sub(lastSeenTime) > rl.cfg.ClientTTL {
					delete(rl.clients, key)
					removed++
				}
			}
			rl.mu.Unlock()

			if removed > 0 {
				logger.Log.Info("Cleaned up expired clients",
					zap.Int("removed_count", removed),
					zap.Int("active_clients", len(rl.clients)),
				)
			}
		}
	}
}

// Metrics : log, dashboard, ...
// Optional: get metrics
func (rl *RateLimiter) Metrics() (activeClients int, blocked int) {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	return len(rl.clients), rl.blockedRequests
}
