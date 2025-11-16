// Package modules - Rate limiter
package modules

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"anti-ddos/pkg/logger"

	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

type Client struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type RateLimiter struct {
	clients       map[string]*Client
	mu            sync.RWMutex
	maxRequests   int
	refillRate    float64
	burst         int
	blockDuration time.Duration

	// Metrics
	blockedRequests int
}

var (
	cleanupInterval = getEnvAsInt("RATE_LIMIT_CLEANUP_INTERVAL", 60)
	clientTTL       = getEnvAsInt("RATE_LIMIT_CLIENT_TTL", 180)
)

// ------------------- Helper functions -------------------

func getEnvAsInt(name string, defaultVal int) int {
	if valStr, exists := os.LookupEnv(name); exists {
		if val, err := strconv.Atoi(valStr); err == nil {
			return val
		}
	}
	return defaultVal
}

func getEnvAsFloat(name string, defaultVal float64) float64 {
	if valStr, exists := os.LookupEnv(name); exists {
		if val, err := strconv.ParseFloat(valStr, 64); err == nil {
			return val
		}
	}
	return defaultVal
}

// ------------------- IP extraction -------------------

func getClientIP(c fiber.Ctx) string {
	if forwarded := c.Get("Forwarded"); forwarded != "" {
		if ip := parseForwardedHeader(forwarded); ip != "" {
			return ip
		}
	}

	headers := []string{
		"X-Forwarded-For", "X-Real-IP", "X-Client-IP", "CF-Connecting-IP", "True-Client-IP",
	}
	for _, h := range headers {
		if ip := c.Get(h); ip != "" {
			ips := strings.Split(ip, ",")
			if len(ips) > 0 {
				clientIP := strings.TrimSpace(ips[0])
				if net.ParseIP(clientIP) != nil {
					return clientIP
				}
			}
		}
	}
	ip := c.IP()
	if ip != "" && net.ParseIP(ip) != nil {
		return ip
	}
	return ""
}

func parseForwardedHeader(forwarded string) string {
	entries := strings.SplitSeq(forwarded, ",")
	for entry := range entries {
		entry = strings.TrimSpace(entry)
		if strings.HasPrefix(entry, "for=") {
			forPart, _ := strings.CutPrefix(entry, "for=")
			forPart = strings.Trim(forPart, `"`)
			if idx := strings.Index(forPart, ";"); idx != -1 {
				forPart = forPart[:idx]
			}
			clientIP := strings.TrimSpace(forPart)
			if net.ParseIP(clientIP) != nil {
				return clientIP
			}
		}
	}
	return ""
}

// ------------------- RateLimiter -------------------

func NewRateLimiter(maxRequests int, refillRate float64, burst int, blockSec int) *RateLimiter {
	return &RateLimiter{
		clients:       make(map[string]*Client),
		maxRequests:   maxRequests,
		refillRate:    refillRate,
		burst:         burst,
		blockDuration: time.Duration(blockSec) * time.Second,
	}
}

func (rl *RateLimiter) getKey(ip, endpoint string) string {
	return fmt.Sprintf("%s|%s", ip, endpoint)
}

func (rl *RateLimiter) getLimiter(ip, endpoint string) *rate.Limiter {
	key := rl.getKey(ip, endpoint)

	rl.mu.RLock()
	client, exists := rl.clients[key]
	rl.mu.RUnlock()

	if !exists {
		rl.mu.Lock()
		defer rl.mu.Unlock()

		client, exists = rl.clients[key]
		if !exists {
			client = &Client{
				limiter:  rate.NewLimiter(rate.Limit(rl.refillRate), rl.burst),
				lastSeen: time.Now(),
			}
			rl.clients[key] = client
		}
	} else {
		rl.mu.Lock()
		client.lastSeen = time.Now()
		rl.mu.Unlock()
	}

	return client.limiter
}

// RateLimiterMiddleware :
// ab -n 20 -c 1 http://localhost:8080/api/v1/download
// https://httpd.apache.org/docs/2.4/programs/ab.html
// windows user: https://www.apachelounge.com/download/
func (rl *RateLimiter) RateLimiterMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		ip := getClientIP(c)
		endpoint := c.Path()
		limiter := rl.getLimiter(ip, endpoint)
		if !limiter.Allow() {
			rl.mu.Lock()
			rl.blockedRequests++
			rl.mu.Unlock()

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
		zap.Int("cleanup_interval_seconds", cleanupInterval),
		zap.Int("client_ttl_seconds", clientTTL),
	)

	ticker := time.NewTicker(time.Duration(cleanupInterval) * time.Second)
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
				if now.Sub(client.lastSeen) > time.Duration(clientTTL)*time.Second {
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

// Metrics :
// Optional: get metrics
func (rl *RateLimiter) Metrics() (activeClients int, blocked int) {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	return len(rl.clients), rl.blockedRequests
}
