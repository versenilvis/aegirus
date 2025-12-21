package upstream

import (
	"context"
	"errors"
	"time"

	"aegirus/pkg/logger"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/proxy"
	"go.uber.org/zap"
)

type Upstream struct {
	Name string
	LB   *RoundRobinBalancer
}

func NewUpstream(name string, servers []Target) (*Upstream, error) {
	if len(servers) == 0 {
		return nil, errors.New("no upstream servers list provided")
	}

	return &Upstream{
		Name: name,
		LB:   NewRoundRobinBalancer(servers),
	}, nil
}

func (u *Upstream) ProxyMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		server := u.LB.Next()

		if server == nil {
			logger.Log.Warn("No upstream servers available")
			return c.Status(502).SendString("Bad Gateway: No upstream available")
		}
		ctx, cancel := context.WithTimeout(c.Context(), 2*time.Second)
		defer cancel()

		c.SetContext(ctx)

		if err := proxy.Do(c, server.URL); err != nil {
			server.MarkDead()
			logger.Log.Warn("Upstream Failed",
				zap.String("target", server.URL),
				zap.Error(err),
			)
			return c.Status(502).SendString("Bad Gateway")
		}
		// debug
		// logger.Log.Debug("Load Balancing", zap.String("upstream", u.Name), zap.String("server", server))

		server.MarkAlive()
		return nil
	}
}
