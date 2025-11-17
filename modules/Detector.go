// Package modules - Detector
package modules

import (
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
)

type Suspicious struct {
	score       int
	lastSeen    time.Time
	bannedUntil time.Time
}

type Detector struct {
	ips map[string]*Suspicious
	mu  sync.Mutex
}

func NewDetector() *Detector {
	return &Detector{
		ips: make(map[string]*Suspicious),
	}
}

func (d *Detector) Middleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		ip := getClientIP(c)

		d.mu.Lock()
		info, ok := d.ips[ip]
		if !ok {
			info = &Suspicious{}
			d.ips[ip] = info
		}

		now := time.Now()

		if now.Before(info.bannedUntil) {
			d.mu.Unlock()
			return c.Status(403).JSON(fiber.Map{
				"error": "IP banned",
			})
		}

		if now.Sub(info.lastSeen) < 50*time.Millisecond {
			info.score += 3
		}

		if c.Get("User-Agent") == "" {
			info.score += 2
		}

		if info.score > 10 {
			info.bannedUntil = now.Add(5 * time.Minute)
			info.score = 0
		}

		info.lastSeen = now
		d.mu.Unlock()

		return c.Next()
	}
}
