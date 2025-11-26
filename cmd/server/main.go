package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"anti-ddos/config"
	"anti-ddos/modules"
	"anti-ddos/pkg/logger"
	"anti-ddos/utils"

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
			logger.Log.Panic("Log sync error: %v\n", zap.Error(err))
		}
	}()

	cfg := config.LoadConfig()
	logger.Log.Info("Application Configuration Loaded", zap.Object("config", cfg))

	app := fiber.New()

	d := modules.NewDetector(cfg.Detector)
	rl := modules.NewRateLimiter(cfg.RateLimiter, d)

	ctx, cancel := context.WithCancel(context.Background())
	go rl.Cleanup(ctx)
	go d.Cleanup(ctx)

	app.Use(rl.RateLimiterMiddleware())
	app.Use(d.DetectorMiddleware())

	app.Get("/", func(c fiber.Ctx) error {
		return c.SendString("Hello world from Aegirus")
	})

	app.Get("/api/test",
		func(c fiber.Ctx) error {
			// WARN: REMEMBER TO REMOVE TRUSTED_PROXIES VARIABLE IN YOUR .env IF YOU ARE USING YOUR VPS/LOCAL MACHINE,
			// BUT IF YOU ARE USING RAILWAY OR NGINX,... SET IT TO TRUE IF YOUR SERVICE IS RUNNING
			ip := utils.GetRealIP(c)

			logger.Log.Info("Request passed",
				zap.String("ip", ip),
				zap.String("path", c.Path()),
				zap.String("HEADER_XFF", c.Get("X-Forwarded-For")), // just for debug in logs
			)
			return c.SendString("Request successful")
		})

	// Graceful shutdown for deploy environment like Railway or Render,
	// you will see air exixting message but you won't see it on deploy environment
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
}
