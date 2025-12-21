package upstream

import (
	"sync/atomic"
)

type RoundRobinBalancer struct {
	servers []string // backend list like ["127.0.0.1:8080", "127.0.0.1:8081"]
	counter uint64
}

func NewRoundRobinBalancer(servers []string) *RoundRobinBalancer {
	return &RoundRobinBalancer{
		servers: servers,
		counter: 0,
	}
}

// Next returns the next backend server in the list
func (rb *RoundRobinBalancer) Next() string {
	if len(rb.servers) == 0 {
		return ""
	}

	current := atomic.AddUint64(&rb.counter, 1)
	// current return the current value like 1, but in arr, index start from 0, so we need to minus 1
	index := (current - 1) % uint64(len(rb.servers))

	return rb.servers[index]
}
