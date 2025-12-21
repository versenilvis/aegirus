package upstream

import (
	"sync/atomic"
	"time"

	"aegirus/pkg/logger"

	"go.uber.org/zap"
)

type Server struct {
	URL    string
	Weight int

	// for health check
	fails     int32
	deadUntil int64
}

type RoundRobinBalancer struct {
	servers []*Server // backend list like ["127.0.0.1:8080", "127.0.0.1:8081"]
	counter uint64
}

type Target struct {
	Addr   string
	Weight int
}

func (s *Server) IsAlive() bool {
	deadUntil := atomic.LoadInt64(&s.deadUntil)
	if deadUntil == 0 {
		return true // alive
	}

	if time.Now().UnixNano() > deadUntil {
		return true
	}

	return false
}

func (s *Server) MarkAlive() {
	if atomic.LoadInt32(&s.fails) > 0 {
		atomic.StoreInt32(&s.fails, 0)
		atomic.StoreInt64(&s.deadUntil, 0)

		logger.Log.Warn("Upstream marked ALIVE",
			zap.String("target", s.URL),
		)
	}
}

func (s *Server) MarkDead() {
	fails := atomic.AddInt32(&s.fails, 1)

	if fails >= 3 {
		cooldown := time.Now().Add(10 * time.Second).UnixNano()
		atomic.StoreInt64(&s.deadUntil, cooldown)

		logger.Log.Warn("Upstream marked DEAD (threshold)",
			zap.String("target", s.URL),
			zap.Int32("fails", fails),
			zap.Duration("cooldown", 10*time.Second),
		)
	}
}

func (s *Server) MarkDeadImmediately() {
	now := time.Now().UnixNano()
	prev := atomic.LoadInt64(&s.deadUntil)

	if prev == 0 || now > prev {
		cooldown := time.Now().Add(10 * time.Second)
		atomic.StoreInt64(&s.deadUntil, cooldown.UnixNano())
		atomic.StoreInt32(&s.fails, 0)

		logger.Log.Warn("Upstream marked DEAD (immediate)",
			zap.String("target", s.URL),
			zap.Duration("cooldown", 10*time.Second),
		)
		return
	}
}

func NewRoundRobinBalancer(targets []Target) *RoundRobinBalancer {
	var servers []*Server
	uniqueServers := make([]*Server, len(targets))
	for i, t := range targets {
		uniqueServers[i] = &Server{
			URL:    t.Addr,
			Weight: t.Weight,
		}
	}

	for _, s := range uniqueServers {
		w := s.Weight
		if w < 1 {
			w = 1
		}

		for i := 0; i < w; i++ {
			servers = append(servers, s)
		}
	}

	return &RoundRobinBalancer{
		servers: servers,
		counter: 0,
	}
}

// Next returns the next backend server in the list
func (rb *RoundRobinBalancer) Next() *Server {
	if len(rb.servers) == 0 {
		return nil
	}
	for i := 0; i < len(rb.servers); i++ {

		current := atomic.AddUint64(&rb.counter, 1)
		// current return the current value like 1, but in arr, index start from 0, so we need to minus 1
		index := (current - 1) % uint64(len(rb.servers))

		server := rb.servers[index]
		// passive check, return if server is alive
		if server.IsAlive() {
			return server
		}
	}
	return nil
}
