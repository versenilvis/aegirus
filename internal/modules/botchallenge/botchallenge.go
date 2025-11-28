// Package botchallenge - Bot Challenge
package botchallenge

import (
	_ "embed"
	"encoding/json"
	"html"
	"strconv"
	"strings"
	"sync"
	"time"

	"aegirus/internal/config"
	"aegirus/internal/utils"
	"aegirus/pkg/logger"

	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

type attemptState struct {
	Count        int
	WindowStart  time.Time
	BlockedUntil time.Time
}

type BotChallenge struct {
	cfg    config.BotChallengeConfig
	tokens *TokenManager
	pow    *POW

	mu       sync.Mutex
	attempts map[string]*attemptState
}

func (bc *BotChallenge) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		bc.mu.Lock()
		now := time.Now()
		for ip, state := range bc.attempts {
			if now.After(state.BlockedUntil) && now.Sub(state.WindowStart) > time.Duration(bc.cfg.POWWindow)*2 {
				delete(bc.attempts, ip)
			}
		}
		bc.mu.Unlock()
	}
}

func NewBotChallenge(cfg config.BotChallengeConfig) *BotChallenge {
	diff := cfg.Difficulty
	if diff < 1 {
		diff = 1
	}
	if diff > 6 {
		diff = 6
	}
	bc := &BotChallenge{
		cfg:      cfg,
		tokens:   NewTokenManager(cfg),
		pow:      NewPOW(diff),
		attempts: make(map[string]*attemptState),
	}
	go bc.cleanupLoop()

	return bc
}

func (bc *BotChallenge) checkRateLimit(ip string) bool {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	state, exists := bc.attempts[ip]
	now := time.Now()

	if !exists {
		bc.attempts[ip] = &attemptState{
			Count:       1,
			WindowStart: now,
		}
		return true
	}

	if now.Before(state.BlockedUntil) {
		return false
	}

	// SỬA: Dùng config
	window := time.Duration(bc.cfg.POWWindow)
	if now.Sub(state.WindowStart) > window {
		state.Count = 1
		state.WindowStart = now
		return true
	}

	state.Count++
	// SỬA: Dùng config
	if state.Count > bc.cfg.MaxPOWAttempts {
		state.BlockedUntil = now.Add(time.Duration(bc.cfg.POWBanDuration))
		logger.Log.Warn("PoW Rate Limit Exceeded", zap.String("ip", ip), zap.Int("attempts", state.Count))
		return false
	}

	return true
}

//go:embed templates/challenge.html
var challengeHTML string

func (bc *BotChallenge) renderTemplate(salt string, difficulty int, reqID string) string {
	htmlFile := challengeHTML
	htmlFile = strings.ReplaceAll(htmlFile, "{{COOKIE_NAME}}", html.EscapeString(bc.cfg.CookieName))
	htmlFile = strings.ReplaceAll(htmlFile, "{{TTL}}", strconv.Itoa(int(time.Duration(bc.cfg.CookieTTL).Seconds())))
	saltJSON, _ := json.Marshal(salt)
	htmlFile = strings.ReplaceAll(htmlFile, "{{SALT}}", string(saltJSON))
	htmlFile = strings.ReplaceAll(htmlFile, "{{DIFFICULTY}}", strconv.Itoa(difficulty))
	htmlFile = strings.ReplaceAll(htmlFile, "{{REQUEST_ID}}", html.EscapeString(reqID))
	return htmlFile
}

func (bc *BotChallenge) BotChallengeMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		if !bc.cfg.Enabled {
			return c.Next()
		}

		reqID, ok := c.Locals("requestid").(string)
		if !ok {
			reqID = "unknown"
		}

		// just for debug ui (/?ui=1)
		// if c.Query("ui") == "1" {
		// 	c.Set("Content-Type", "text/html; charset=utf-8")
		//
		// 	return c.SendString(bc.renderTemplate("", bc.pow.Difficulty, reqID))
		// }

		ip := utils.GetRealIP(c)
		cookie := c.Cookies(bc.cfg.CookieName)

		if bc.tokens.Validate(ip, cookie) {
			return c.Next()
		}

		salt := c.Query("salt")
		nonceStr := c.Query("nonce")

		if salt != "" && nonceStr != "" {
			if !bc.checkRateLimit(ip) {
				goto ServeChallenge
			}
			nonce, err := strconv.Atoi(nonceStr)
			if err == nil {
				if bc.pow.VerifyPOW(salt, nonce) {
					logger.Log.Info("POW Solved Successfully", zap.String("ip", ip))

					token := bc.tokens.Generate(ip)

					c.Cookie(&fiber.Cookie{
						Name:     bc.cfg.CookieName,
						Value:    token,
						MaxAge:   int(time.Duration(bc.cfg.CookieTTL).Seconds()),
						Path:     "/",
						SameSite: "Lax",
						HTTPOnly: true,
						Secure:   true,
					})

					return c.Redirect().To(c.Path())
				} else {
					logger.Log.Warn("POW Verification Failed", zap.String("ip", ip))
				}
			}
		}
	ServeChallenge:
		c.Locals("is_challenge", true)
		newSalt, err := bc.pow.GenerateSalt()
		if err != nil {
			logger.Log.Error("Failed to generate salt", zap.Error(err))
			return c.Status(500).SendString("Internal Server Error")
		}
		c.Set("Content-Type", "text/html; charset=utf-8")
		c.Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
		return c.SendString(bc.renderTemplate(newSalt, bc.pow.Difficulty, reqID))
	}
}
