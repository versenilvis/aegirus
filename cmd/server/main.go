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

	detector := modules.NewDetector()
	rl := modules.NewRateLimiter(10, 5, 10, 60)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go rl.Cleanup(ctx)
	app.Get("", func(c fiber.Ctx) error {
		return c.SendString("Hello World")
	})
	app.Get("/api/test",
		rl.RateLimiterMiddleware(),
		detector.Middleware(),
		func(c fiber.Ctx) error {
			logger.Log.Info("Request passed RateLimiter",
				zap.String("ip", c.IP()),
				zap.String("path", c.Path()),
			)
			return c.SendString("Request successful")
		})
	port := utils.GetEnv("PORT", "8080")
	if err := app.Listen(":" + port); err != nil {
		cancel()
		panic(err)
	}
}
