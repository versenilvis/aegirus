package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

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

	app := fiber.New()

	d := modules.NewDetector()

	rl := modules.NewRateLimiter(10, 5.0, 10, 60, d)

	ctx, cancel := context.WithCancel(context.Background())

	go rl.Cleanup(ctx)
	go d.Cleanup(ctx)

	app.Get("/", func(c fiber.Ctx) error {
		return c.SendString("Hello World")
	})

	app.Get("/api/test",
		d.DetectorMiddleware(),
		rl.RateLimiterMiddleware(),

		func(c fiber.Ctx) error {
			logger.Log.Info("Request passed all shields",
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
