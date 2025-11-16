package main

import (
	"context"

	"anti-ddos/modules"
	"anti-ddos/pkg/logger"
	"anti-ddos/utils"

	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

func main() {
	logger.InitLogger()
	app := fiber.New()

	rl := modules.NewRateLimiter(10, 5, 10, 60)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go rl.Cleanup(ctx)

	app.Get("/api/test", rl.RateLimiterMiddleware(), func(c fiber.Ctx) error {
		logger.Log.Info("Request passed RateLimiter",
			zap.String("ip", c.IP()),
			zap.String("path", c.Path()),
		)
		return c.SendString("Request successful")
	})
	if err := app.Listen(":" + utils.GetEnv("BACKEND_PORT", "8080")); err != nil {
		cancel()
		panic(err)
	}
}
