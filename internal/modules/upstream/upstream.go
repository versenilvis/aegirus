package upstream

import (
	"errors"

	"aegirus/pkg/logger"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/proxy"
)

type Upstream struct {
	Name string
	LB   *RoundRobinBalancer
}

func NewUpstream(name string, servers []Server) (*Upstream, error) {
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
		target := u.LB.Next()

		if target == "" {
			logger.Log.Warn("No upstream servers available")
			return c.Status(502).SendString("Bad Gateway: No upstream available")
		}

		// debug
		// logger.Log.Debug("Load Balancing", zap.String("upstream", u.Name), zap.String("target", target))

		return proxy.Do(c, target)
	}
}
