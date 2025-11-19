package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"anti-ddos/config"
	"anti-ddos/modules"
	"anti-ddos/pkg/logger"
	"anti-ddos/utils"

	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

func main() {
	logger.InitLogger()
	// NOTE: make sure to write logs to file before exit
	defer func() {
		err := logger.Log.Sync()
		if err != nil {
			logger.Log.Panic("Log sync error: %v\n", zap.Error(err))
		}
	}()

	cfg := config.LoadConfig()
	logger.Log.Info("Application Configuration Loaded",
		zap.String("port", cfg.Port),
		zap.Any("detector_config", cfg.Detector),
		zap.Any("rate_limiter_config", cfg.RateLimiter),
	)

	app := fiber.New()

	d := modules.NewDetector(cfg.Detector)
	rl := modules.NewRateLimiter(cfg.RateLimiter, d)

	ctx, cancel := context.WithCancel(context.Background())
	go rl.Cleanup(ctx)
	go d.Cleanup(ctx)

	app.Get("/", func(c fiber.Ctx) error {
		return c.SendString("Hello world from Aegirus")
	})

	app.Get("/api/test",
		d.DetectorMiddleware(),
		rl.RateLimiterMiddleware(),

		func(c fiber.Ctx) error {
			logger.Log.Info("Request passed",
				zap.String("ip", c.IP()),
				zap.String("path", c.Path()),
			)
			return c.SendString("Request successful")
		})

	// Graceful Shutdown
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
	_ = app.Shutdown()
}
