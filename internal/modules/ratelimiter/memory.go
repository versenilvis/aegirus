package ratelimiter

import (
	"context"
	"sync"
	"time"

	"aegirus/pkg/logger"

	"go.uber.org/zap"
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
			threshold := time.Now().UnixNano() - int64(5*time.Minute)
			targetSize := 90000

			for k, c := range m.clients {
				if c.lastSeen < threshold {
					delete(m.clients, k)
					if len(m.clients) <= targetSize {
						break
					}
				}
			}

			if len(m.clients) > targetSize {
				type clientEntry struct {
					key      string
					lastSeen int64
				}
				entries := make([]clientEntry, 0, len(m.clients))
				for k, c := range m.clients {
					entries = append(entries, clientEntry{key: k, lastSeen: c.lastSeen})
				}

				for i := 0; i < len(entries) && len(m.clients) > targetSize; i++ {
					oldestIdx := i
					for j := i + 1; j < len(entries); j++ {
						if entries[j].lastSeen < entries[oldestIdx].lastSeen {
							oldestIdx = j
						}
					}
					if oldestIdx != i {
						entries[i], entries[oldestIdx] = entries[oldestIdx], entries[i]
					}
					delete(m.clients, entries[i].key)
				}
			}
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

func (m *MemoryStrategy) CleanupRoutine(ctx context.Context, interval, ttl time.Duration) {
	defer func() {
		if r := recover(); r != nil {
			logger.Log.Error("panic in CleanupRoutine", zap.Any("panic", r))
		}
	}()

	ticker := time.NewTicker(time.Duration(interval))
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.mu.RLock()
			now := time.Now().UnixNano()
			expiredKeys := make([]string, 0)
			for key, client := range m.clients {
				if time.Duration(now-client.lastSeen) > ttl {
					expiredKeys = append(expiredKeys, key)
				}
			}
			m.mu.RUnlock()

			if len(expiredKeys) > 0 {
				m.mu.Lock()
				for _, key := range expiredKeys {
					delete(m.clients, key)
				}
				m.mu.Unlock()
			}
		}
	}
}
