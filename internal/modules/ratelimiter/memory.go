package ratelimiter

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type MemoryStrategy struct {
	clients map[string]*Client
	mu      sync.RWMutex
}

func NewMemoryStrategy() *MemoryStrategy {
	return &MemoryStrategy{
		clients: make(map[string]*Client),
	}
}

func (m *MemoryStrategy) Allow(ctx context.Context, key string, limit int, windows time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	client, exists := m.clients[key]
	if !exists {
		if len(m.clients) > 100000 {
			m.clients = make(map[string]*Client)
		}

		r := rate.Limit(float64(limit) / windows.Seconds())

		client = &Client{
			limiter:  rate.NewLimiter(r, limit),
			lastSeen: time.Now().UnixNano(),
		}

		m.clients[key] = client
	}

	client.lastSeen = time.Now().UnixNano()

	return client.limiter.Allow(), nil
}

func (m *MemoryStrategy) CleanupRoutine(interval, ttl time.Duration) {
	ticker := time.NewTicker(time.Duration(interval))
	for range ticker.C {
		m.mu.Lock()
		now := time.Now().UnixNano()
		for key, client := range m.clients {
			if time.Duration(now-client.lastSeen) > ttl {
				delete(m.clients, key)
			}
		}
		m.mu.Unlock()
	}
}
