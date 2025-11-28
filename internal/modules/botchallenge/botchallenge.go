// Package botchallenge - Bot Challenge
package botchallenge

import (
	_ "embed"
	"html"
	"strconv"
	"strings"
	"time"

	"aegirus/internal/config"
	"aegirus/internal/utils"
	"aegirus/pkg/logger"

	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

type BotChallenge struct {
	cfg    config.BotChallengeConfig
	tokens *TokenManager
	pow    *POW
}

func NewBotChallenge(cfg config.BotChallengeConfig) *BotChallenge {
	diff := cfg.Difficulty
	if diff < 1 {
		diff = 1
	}
	if diff > 6 {
		diff = 6
	}
	return &BotChallenge{
		cfg:    cfg,
		tokens: NewTokenManager(cfg),
		pow:    NewPOW(diff),
	}
}

//go:embed templates/challenge.html
var challengeHTML string

func (bc *BotChallenge) renderTemplate(salt string, difficulty int, reqID string) string {
	htmlFile := challengeHTML
	htmlFile = strings.ReplaceAll(htmlFile, "{{COOKIE_NAME}}", html.EscapeString(bc.cfg.CookieName))
	htmlFile = strings.ReplaceAll(htmlFile, "{{TTL}}", strconv.Itoa(int(time.Duration(bc.cfg.CookieTTL).Seconds())))
	htmlFile = strings.ReplaceAll(htmlFile, "{{SALT}}", html.EscapeString(salt))
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
		c.Locals("is_challenge", true)
		newSalt, err := bc.pow.GenerateSalt()
		if err != nil {
			logger.Log.Error("Failed to generate salt", zap.Error(err))
			return c.Status(500).SendString("Internal Server Error")
		}
		c.Set("Content-Type", "text/html; charset=utf-8")
		// Không cache trang challenge này
		c.Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")

		return c.SendString(bc.renderTemplate(newSalt, bc.pow.Difficulty, reqID))
	}
}
