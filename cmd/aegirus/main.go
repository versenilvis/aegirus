package main

import (
	"context"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"aegirus/internal/config"
	"aegirus/internal/modules/botchallenge"
	"aegirus/internal/modules/detector"
	"aegirus/internal/modules/ratelimiter"
	"aegirus/internal/utils"
	"aegirus/pkg/logger"

	"github.com/gofiber/fiber/v3"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

func main() {
	if err := godotenv.Load(); err != nil {
		// WARN: Do not change this to zap logger
		log.Println("INFO: No .env file found, using your deploy environment variables.")
	} else {
		log.Println("INFO: Using .env file")
	}

	logger.InitLogger()
	// NOTE: make sure to write logs to file before exit
	defer func() {
		err := logger.Log.Sync()
		if err != nil {
			log.Printf("Log sync error (may be benign): %v\n", err)
		}
	}()

	cfg := config.LoadConfig()
	logger.Log.Debug("Application Configuration Loaded",
		zap.String("app_env", cfg.AppEnv),
		zap.String("trusted_proxies", cfg.TrustedProxies),
		zap.Any("detector", cfg.Detector),
		zap.Any("rate_limiter", cfg.RateLimiter),
		zap.Any("bot_challenge", cfg.BotChallenge),
	)

	app := fiber.New()

	d := detector.NewDetector(cfg.Detector)
	rl := ratelimiter.NewRateLimiter(cfg.RateLimiter, d)
	bc := botchallenge.NewBotChallenge(cfg.BotChallenge)

	smartLog := utils.NewSmartLogger()
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	wg.Add(2)
	go func() {
		defer wg.Done()
		rl.Cleanup(ctx)
	}()
	go func() {
		defer wg.Done()
		d.Cleanup(ctx)
	}()

	app.Use(func(c fiber.Ctx) error {
		reqID := utils.GenerateRayID()

		c.Locals("requestid", reqID)
		c.Set("X-Aegirus-Ray-ID", reqID)

		err := c.Next()

		// WARN: REMEMBER TO REMOVE TRUSTED_PROXIES VARIABLE IN YOUR .env IF YOU ARE USING YOUR VPS/LOCAL MACHINE,
		// BUT IF YOU ARE USING RAILWAY, REDNER, SSH TUNNEL OR NGINX,... OR ANY 3RD PARTY SERVICE SET IT TO TRUE IF YOUR SERVICE IS RUNNING
		ip := utils.GetRealIP(c)
		status := c.Response().StatusCode()
		isChallenge, _ := c.Locals("is_challenge").(bool)
		if isChallenge {
			shouldLog, count := smartLog.ShouldLog(ip)

			if shouldLog {
				msg := "Bot Challenge Triggered"
				if count > 1 {
					msg = "Bot Challenge Flood (Throttled)"
				}
				logger.Log.Info(msg,
					zap.String("req_id", reqID),
					zap.String("ip", ip),
					zap.String("action", "serving_html_pow"),
					zap.Int64("recent_count", count),
				)
			}
			return err
		}
		if status == 403 || status == 429 {
			shouldLog, count := smartLog.ShouldLog(ip)

			if shouldLog {
				msg := "Request Blocked"
				if count > 1 {
					msg = "Attack Flood Detected (Throttled)"
				}
				logger.Log.Warn(msg,
					zap.String("req_id", reqID),
					zap.String("ip", ip),
					zap.Int("status", status),
					zap.Int64("recent_blocked_count", count),
				)
			}
			return err
		}
		// NOTE: set LOG_SAMPLING_RATE=100 in .env -> All log (Debug or development but not recommended)
		// set LOG_SAMPLING_RATE=10 (or 5) in .env -> 10% log (Production)
		// set LOG_SAMPLING_RATE=0 in .env -> 0% log (While being ddos, save I/O)
		if rand.Intn(100) < cfg.LogSamplingRate {
			logger.Log.Info("Request passed",
				zap.String("req_id", reqID),
				zap.String("ip", ip),
				zap.String("path", c.Path()),
				zap.String("HEADER_XFF", c.Get("X-Forwarded-For")), // just for debug in logs
				zap.String("method", c.Method()),
				zap.Int("status", status),
				zap.String("user_agent", c.Get("User-Agent")),
			)
		}
		return err
	})

	app.Use(d.DetectorMiddleware())
	app.Use(bc.BotChallengeMiddleware())
	app.Use(rl.RateLimiterMiddleware())

	app.Get("/", func(c fiber.Ctx) error {
		return c.SendString("Hello world from Aegirus")
	})

	app.Get("/api/test", func(c fiber.Ctx) error {
		return c.SendString("Request successful")
	})

	// Graceful shutdown for deploy environment like Railway or Render,
	// you will see air exiting message but you won't see it on deploy environment
	go func() {
		port := utils.GetEnv("PORT", "8080")
		if err := app.Listen(":" + port); err != nil {
			logger.Log.Panic("Server failed to start", zap.Error(err))
		}
	}()

	// Ctrl+C
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	logger.Log.Info("SHUTTING DOWN SERVER...")
	cancel()
	if err := app.Shutdown(); err != nil {
		logger.Log.Error("Error during shutdown", zap.Error(err))
	}
	wg.Wait()
	logger.Log.Info("Cleanup done")
}
