package upstream

import (
	"errors"
	"strings"
	"time"

	"aegirus/pkg/logger"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/proxy"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

const (
	ProxyTimeout = 10 * time.Second
)

type Upstream struct {
	Name string
	LB   *RoundRobinBalancer
}

var proxyClient = &fasthttp.Client{
	ReadTimeout:                   ProxyTimeout,     // wait before marking server as dead
	WriteTimeout:                  ProxyTimeout,     // max timeout for aegirus to send data to the main servers
	MaxConnsPerHost:               2000,             // to prevent DDoS
	MaxIdleConnDuration:           90 * time.Second, // no need to create tcp handshake again, in 90s next request will use old connection
	MaxIdemponentCallAttempts:     5,                // max 5 attempts to send request to the same server
	ReadBufferSize:                4096,
	WriteBufferSize:               4096,
	MaxResponseBodySize:           100 * 1024 * 1024, // max response body size
	DisableHeaderNamesNormalizing: false,             // false means TURN ON name normalization
	DisablePathNormalizing:        false,             // false means TURN ON path normalization
}

func isNetworkError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()

	switch {
	case strings.Contains(msg, "connection refused"):
		return true
	case strings.Contains(msg, "timeout"):
		return true
	case strings.Contains(msg, "deadline"):
		return true
	case strings.Contains(msg, "reset by peer"):
		return true
	default:
		return false
	}
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
		body := append([]byte(nil), c.Request().Body()...)

		maxAttempts := 2
		var lastErr error

		for attempts := 0; attempts < maxAttempts; attempts++ {
			server := u.LB.Next()
			if server == nil {
				break
			}

			c.Request().SetBodyRaw(body)

			err := proxy.DoTimeout(c, server.URL, ProxyTimeout, proxyClient)
			if err == nil {
				server.MarkAlive()
				return nil
			}

			lastErr = err
			if isNetworkError(err) {
				server.MarkDeadImmediately()
				logger.Log.Warn("Target failed (network), retrying",
					zap.String("url", server.URL),
					zap.Error(err),
				)
				continue
			}

			server.MarkDead()
			logger.Log.Warn("Target failed, retrying",
				zap.String("url", server.URL),
				zap.Error(err),
			)
			continue
		}

		logger.Log.Warn("All upstreams failed",
			zap.String("upstream", u.Name),
			zap.Error(lastErr),
		)

		return c.Status(503).SendString("Service Unavailable")
	}
}
