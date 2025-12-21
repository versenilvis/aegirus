package utils

import (
	"sync"
	"time"
)

// NOTE: I think it's not necessary to make it as a env variable
// if you think no, just put it in the config and use .env variable

const LogThrottleDuration = 5 * time.Second

type LogState struct {
	WindowStart time.Time
	Count       int64
	mu          sync.Mutex
}
type SmartLogger struct {
	logs sync.Map
	done chan struct{}
}

func NewSmartLogger() *SmartLogger {
	sl := &SmartLogger{
		done: make(chan struct{}),
	}
	go sl.cleanupLoop()
	return sl
}

func (sl *SmartLogger) ShouldLog(key string) (bool, int64, time.Duration) {
	now := time.Now()

	val, _ := sl.logs.LoadOrStore(key, &LogState{
		WindowStart: now,
		Count:       0,
	})

	state := val.(*LogState)

	state.mu.Lock()
	defer state.mu.Unlock()

	if now.Sub(state.WindowStart) < LogThrottleDuration {
		state.Count++
		return false, 0, 0
	}

	// window ended → emit log
	count := state.Count
	duration := now.Sub(state.WindowStart)

	state.WindowStart = now
	state.Count = 1

	return true, count, duration
}

func (sl *SmartLogger) cleanupLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
		case <-sl.done:
			return
		}
		now := time.Now()
		sl.logs.Range(func(key, value any) bool {
			state := value.(*LogState)
			state.mu.Lock()
			last := state.WindowStart
			state.mu.Lock()
			if now.Sub(last) > 3*LogThrottleDuration {
				sl.logs.Delete(key)
			}

			return true
		})
	}
}

func (sl *SmartLogger) Shutdown() {
	close(sl.done)
}
