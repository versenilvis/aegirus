// Package botchallenge - Bot Challenge
package botchallenge

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"html/template"
	"strconv"
	"sync"
	"time"

	"aegirus/internal/config"
	"aegirus/internal/utils"
	"aegirus/pkg/logger"

	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

//go:embed templates/challenge.html
var challengeHTML string

type attemptState struct {
	Count        int
	WindowStart  time.Time
	BlockedUntil time.Time
}

type BotChallenge struct {
	cfg    config.BotChallengeConfig
	tokens *TokenManager
	pow    *POW

	htmlFile *template.Template

	mu       sync.Mutex
	attempts map[string]*attemptState
	stopCh   chan struct{}

	closeOnce sync.Once
}

func (bc *BotChallenge) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			bc.mu.Lock()
			now := time.Now()
			for ip, state := range bc.attempts {
				if now.After(state.BlockedUntil) && now.Sub(state.WindowStart) > time.Duration(bc.cfg.POWWindow)*2 {
					delete(bc.attempts, ip)
				}
			}
			bc.mu.Unlock()
		case <-bc.stopCh:
			return
		}
	}
}

func (bc *BotChallenge) Close() {
	bc.closeOnce.Do(func() {
		if bc.tokens != nil {
			bc.tokens.Stop()
		}
		close(bc.stopCh)
	})
}

func NewBotChallenge(cfg config.BotChallengeConfig) *BotChallenge {
	diff := cfg.Difficulty
	if diff < 1 {
		diff = 1
	}
	if diff > 6 {
		diff = 6
	}

	htmlTempl := template.New("challenge").Funcs(template.FuncMap{
		"js": func(s string) template.JS {
			// s is already a JSON-encoded string (e.g., "abc123")
			// Return it as-is for JavaScript parsing
			return template.JS(s)
		},
	})
	htmlTempl, err := htmlTempl.Parse(challengeHTML)
	if err != nil {
		panic(err)
	}

	bc := &BotChallenge{
		cfg:      cfg,
		tokens:   NewTokenManager(cfg),
		pow:      NewPOW(diff),
		htmlFile: htmlTempl,
		attempts: make(map[string]*attemptState),
		stopCh:   make(chan struct{}),
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

	window := time.Duration(bc.cfg.POWWindow)
	if now.Sub(state.WindowStart) > window {
		state.Count = 1
		state.WindowStart = now
		return true
	}

	state.Count++
	if state.Count > bc.cfg.MaxPOWAttempts {
		state.BlockedUntil = now.Add(time.Duration(bc.cfg.POWBanDuration))
		logger.Log.Warn("PoW Rate Limit Exceeded", zap.String("ip", ip), zap.Int("attempts", state.Count))
		return false
	}

	return true
}

func (bc *BotChallenge) renderTemplate(salt string, difficulty int, reqID string) string {
	var buf bytes.Buffer

	// JSON-encode SALT to safely embed in JavaScript string literal
	saltJSON, err := json.Marshal(salt)
	if err != nil {
		logger.Log.Error("Failed to marshal salt to JSON", zap.Error(err))
		return "Internal Server Error"
	}

	data := map[string]string{
		"COOKIE_NAME": bc.cfg.CookieName,
		"TTL":         strconv.Itoa(int(time.Duration(bc.cfg.CookieTTL).Seconds())),
		"SALT":        string(saltJSON), // JSON-encoded string
		"DIFFICULTY":  strconv.Itoa(difficulty),
		"REQUEST_ID":  reqID,
	}

	if err := bc.htmlFile.Execute(&buf, data); err != nil {
		logger.Log.Error("Failed to render template", zap.Error(err))
		return "Internal Server Error"
	}

	return buf.String()
}

func (bc *BotChallenge) BotChallengeMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		path := c.Path()
		if utils.IsStaticAsset(path) {
			c.Set("Cache-Control", "public, max-age=86400")
			return c.Next()
		}
		if !bc.cfg.Enabled {
			return c.Next()
		}

		reqID, ok := c.Locals("requestid").(string)
		if !ok {
			reqID = "unknown"
		}

		// NOTE: just for ui debug only (/?ui=1)
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
			nonce, err := strconv.ParseInt(nonceStr, 10, 64)
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
						Secure:   bc.cfg.CookieSecure, // WARN: SET TO TRUE ON PRODUCTION (HTTPS)
					})

					// make sure cloudflare cache is disabled
					c.Set("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate, max-age=0")
					return c.Redirect().To(c.Path())
				} else {
					logger.Log.Warn("POW Verification Failed", zap.String("ip", ip))
				}
			}
		}
	ServeChallenge:
		c.Locals("is_challenge", true)
		// make sure cloudflare cache is disabled
		c.Set("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate, max-age=0")
		c.Set("Pragma", "no-cache")
		c.Set("Expires", "0")
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
