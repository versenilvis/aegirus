package main

import (
	"context"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"aegirus/internal/config"
	"aegirus/internal/modules/detector"
	"aegirus/internal/modules/ratelimiter"
	"aegirus/internal/modules/upstream"
	"aegirus/internal/utils"
	"aegirus/pkg/logger"

	"github.com/gofiber/fiber/v3"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"

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
	)

	app := fiber.New()

	var rdb *redis.Client
	if cfg.Redis.Addr != "" {
		opt, err := redis.ParseURL(cfg.Redis.Addr)
		if err != nil {
			opt = &redis.Options{
				Addr:     cfg.Redis.Addr,
				Password: cfg.Redis.Password,
				DB:       cfg.Redis.DB,
			}
		}
		rdb = redis.NewClient(opt)

		if err := rdb.Ping(context.Background()).Err(); err != nil {
			logger.Log.Error("Cannot connect to Redis, fallback to Memory", zap.Error(err))
			rdb = nil
		}
	}

	targets := []upstream.Target{
		{Addr: "http://127.0.0.1:3001", Weight: 3}, // Server A
		{Addr: "http://127.0.0.1:3002", Weight: 1}, // Server B
	}

	testService, err := upstream.NewUpstream("local-test", targets)
	if err != nil {
		logger.Log.Panic("Init upstream failed", zap.Error(err))
	}

	d := detector.NewDetector(cfg.Detector)
	rl := ratelimiter.NewManager(cfg.RateLimiter, d, rdb)

	smartLog := utils.NewSmartLogger()
	// var wg sync.WaitGroup
	_, cancel := context.WithCancel(context.Background())
	// wg.Add(2)

	app.Use(func(c fiber.Ctx) error {
		reqID := utils.GenerateRayID()

		c.Locals("requestid", reqID)
		c.Set("X-Aegirus-Ray-ID", reqID)

		err := c.Next()

		ip := utils.GetRealIP(c)
		status := c.Response().StatusCode()

		if status == 403 || status == 429 {
			key := ip + "|" + strconv.Itoa(status)
			shouldLog, count, window := smartLog.ShouldLog(key)

			if shouldLog {
				logLevel := zap.WarnLevel
				msg := "Request Blocked"
				if count > 1 {
					msg = "Attack Flood Detected (Throttled)"
				} else {
					logLevel = zap.InfoLevel
				}
				logger.Log.Log(logLevel, msg,
					zap.String("req_id", reqID),
					zap.String("ip", ip),
					zap.Int("status", status),
					zap.Int64("recent_blocked_count", count),
					zap.Duration("window", window),
				)
			}
			return err
		}

		if rand.Intn(100) < cfg.LogSamplingRate {
			logger.Log.Info("Request passed",
				zap.String("req_id", reqID),
				zap.String("ip", ip),
				zap.String("path", c.Path()),
				zap.String("HEADER_XFF", c.Get("X-Forwarded-For")),
				zap.String("method", c.Method()),
				zap.Int("status", status),
				zap.String("user_agent", c.Get("User-Agent")),
			)
		}
		return err
	})

	app.Use(d.DetectorMiddleware())
	app.Use(rl.RateLimiterMiddleware())

	app.Get("/", func(c fiber.Ctx) error {
		return c.SendString("Hello world from Aegirus")
	})

	app.Get("/api/test", func(c fiber.Ctx) error {
		return c.SendString("Request successful")
	})

	// run mock server first then run main server
	// try to test with: curl localhost:8080/whateverroute, just dont be / or /api/test
	// you will see it calls A 3 times and B 1 time
	app.All("/*", testService.ProxyMiddleware())

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

	// Shutdown rate limiter (stops cleanup goroutine if using memory strategy)
	rl.Shutdown()

	// Shutdown Fiber app
	if err := app.Shutdown(); err != nil {
		logger.Log.Error("Error during shutdown", zap.Error(err))
	}

	// Close Redis client if it was created
	if rdb != nil {
		if err := rdb.Close(); err != nil {
			logger.Log.Error("Error closing Redis client", zap.Error(err))
		} else {
			logger.Log.Info("Redis client closed")
		}
	}

	// wg.Wait()
	logger.Log.Info("Cleanup done")
}
