package upstream

import (
	"errors"
	"time"

	"aegirus/pkg/logger"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/proxy"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

type Upstream struct {
	Name string
	LB   *RoundRobinBalancer
}

var proxyClient = &fasthttp.Client{
	ReadTimeout:                   10 * time.Second, // wait 10 secs before marking server as dead
	WriteTimeout:                  10 * time.Second, // max timeout for aegirus to send data to the main servers
	MaxConnsPerHost:               2000,             // to prevent DDoS
	MaxIdleConnDuration:           90 * time.Second, // no need to create tcp handshake again, in 90s next request will use old connection
	MaxIdemponentCallAttempts:     5,                // max 5 attempts to send request to the same server
	ReadBufferSize:                4096,
	WriteBufferSize:               4096,
	MaxResponseBodySize:           100 * 1024 * 1024, // max response body size
	DisableHeaderNamesNormalizing: false,             // false means TURN ON nanme normalization
	DisablePathNormalizing:        false,             // false means TURN ON path normalization
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

		// https://docs.gofiber.io/next/whats_new#fasthttp-transport-integration
		err := proxy.DoTimeout(c, server.URL, 10*time.Second, proxyClient)
		if err != nil {
			server.MarkDead()

			logger.Log.Warn("Upstream Failed",
				zap.String("upstream", u.Name),
				zap.String("target", server.URL),
				zap.Error(err),
			)
			return c.Status(502).SendString("Bad Gateway")
		}

		server.MarkAlive()
		return nil
	}
}
