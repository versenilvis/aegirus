// Package modules - Detector & IP Blocker
package modules

import (
	"context"
	"sync"
	"time"

	"anti-ddos/config"
	"anti-ddos/pkg/logger"
	"anti-ddos/utils"

	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

type Suspicious struct {
	Score       int
	LastSeen    time.Time
	BannedUntil time.Time
}

type Detector struct {
	ips map[string]*Suspicious
	mu  sync.RWMutex
	cfg config.DetectorConfig
}

func NewDetector(cfg config.DetectorConfig) *Detector {
	return &Detector{
		ips: make(map[string]*Suspicious),
		cfg: cfg,
	}
}

func (d *Detector) DetectorMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		ip := utils.GetRealIP(c)
		now := time.Now()

		// first check if IP is banned
		d.mu.RLock()

		info, exists := d.ips[ip]

		if exists && now.Before(info.BannedUntil) {
			remaining := info.BannedUntil.Sub(now)
			// WARN: if you don't unlock, next request will wait forever and server will be deadlock
			// like "return"
			d.mu.RUnlock()

			// NOTE: this is just for bot crawler and api testing tool
			c.Set("Retry-After", remaining.String())
			return c.Status(403).JSON(fiber.Map{
				"error":       "IP is banned temporarily",
				"retry_after": remaining.String(),
			})
		}
		d.mu.RUnlock()

		// if not banned
		d.mu.Lock()

		if !exists {
			info, exists = d.ips[ip]
			if !exists {
				info = &Suspicious{}
				d.ips[ip] = info
			}
		}

		// WARN: Please understand this code block carefully, this can't be indispensable
		if now.Sub(info.LastSeen) > d.cfg.ScoreDecayTime {
			info.Score = 0
		}

		// Rule A: No User-Agent (amateur bot script)
		if c.Get("User-Agent") == "" {
			info.Score += 2
		}

		// Rule B: (optional) like check path contains "wp-admin" or ".env" ..., that is vulnerability scanning

		info.LastSeen = now

		if info.Score >= d.cfg.BanThreshold {
			info.BannedUntil = now.Add(d.cfg.BanDuration)

			logger.Log.Warn("IP Banned by Detector",
				zap.String("ip", ip),
				zap.Int("score", info.Score),
			)
			info.Score = 0
		}

		d.mu.Unlock()

		return c.Next()
	}
}

func (d *Detector) Cleanup(ctx context.Context) {
	ticker := time.NewTicker(d.cfg.CleanupTime)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			d.mu.Lock()
			now := time.Now()
			count := 0
			for ip, info := range d.ips {
				if now.After(info.BannedUntil) && now.Sub(info.LastSeen) > 10*time.Minute {
					delete(d.ips, ip)
					count++
				}
			}
			d.mu.Unlock()
			if count > 0 {
				logger.Log.Info("Detector cleanup finished", zap.Int("removed_ips", count))
			}
		}
	}
}

// BanIP :
// NOTE: this is optional, you can call this in /login to prevent brute force attack, or access to admin panel ->  BAN
func (d *Detector) BanIP(ip string, duration time.Duration) {
	d.mu.Lock()
	defer d.mu.Unlock()

	info, exists := d.ips[ip]
	if !exists {
		info = &Suspicious{}
		d.ips[ip] = info
	}

	info.BannedUntil = time.Now().Add(duration)
	info.Score = d.cfg.BanThreshold

	logger.Log.Info("IP Banned manually via internal call", zap.String("ip", ip))
}
