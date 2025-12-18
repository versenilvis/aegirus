package ratelimiter

import (
	"fmt"
	"time"

	"aegirus/internal/config"
	"aegirus/internal/utils"
	"aegirus/pkg/logger"

	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
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

type Manager struct {
	strategy Strategy
	cfg      config.RateLimiterConfig
	banip    IPBanner
}

func NewManager(cfg config.RateLimiterConfig, b IPBanner, rdb *redis.Client) *Manager {
	var strg Strategy

	if rdb != nil {
		logger.Log.Info("Using redis for rate limiter")
		strg = NewRedisStrategy(rdb)
	} else {
		logger.Log.Info("Using memory for rate limiter")
		mem := NewMemoryStrategy()
		go mem.CleanupRoutine(
			time.Duration(cfg.CleanupInterval),
			time.Duration(cfg.ClientTTL),
		)
		strg = mem
	}

	return &Manager{
		strategy: strg,
		cfg:      cfg,
		banip:    b,
	}
}

func (m *Manager) RateLimitMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		ip := utils.GetRealIP(c)
		path := c.Path()

		// e.g rate:192.168.1.1|/api/login
		key := fmt.Sprintf("rate: %s|%s", ip, path)

		allowed, err := m.strategy.Allow(c.Context(), key, m.cfg.Rate, time.Duration(m.cfg.WindowDuration))
		if err != nil {
			logger.Log.Error("Rate limiter strategy error (Fail Open)", zap.Error(err))
			return c.Next()
		}

		if !allowed {
			// Logic Ban IP và Log
			logger.Log.Warn("Rate limit exceeded", zap.String("ip", ip), zap.String("path", path))

			if m.banip != nil {
				m.banip.BanIP(ip, time.Duration(m.cfg.BanDuration))
			}

			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":   "Too many requests",
				"message": "Please slow down!",
			})
		}

		return c.Next()
	}
}
