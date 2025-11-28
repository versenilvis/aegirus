package utils

import (
	"sync"
	"time"
)

// NOTE: I think it's not necessary to make it as a env variable
// if you think no, just put it in the config and use .env variable

const LogThrottleDuration = 5 * time.Second

type LogState struct {
	LastLogTime  time.Time
	BlockedCount int64
	mu           sync.Mutex
}

type SmartLogger struct {
	logs sync.Map
}

func NewSmartLogger() *SmartLogger {
	sl := &SmartLogger{}
	go sl.cleanupLoop()
	return sl
}

func (sl *SmartLogger) ShouldLog(ip string) (bool, int64) {
	val, loaded := sl.logs.LoadOrStore(ip, &LogState{
		LastLogTime:  time.Now(),
		BlockedCount: 1,
	})

	state := val.(*LogState)

	if !loaded {
		return true, 1
	}

	state.mu.Lock()
	defer state.mu.Unlock()

	now := time.Now()

	if now.Sub(state.LastLogTime) < LogThrottleDuration {
		state.BlockedCount++
		return false, 0
	}

	count := state.BlockedCount
	state.BlockedCount = 1
	state.LastLogTime = now

	return true, count
}

func (sl *SmartLogger) cleanupLoop() {
	for {
		time.Sleep(1 * time.Minute)
		now := time.Now()
		sl.logs.Range(func(key, value any) bool {
			state := value.(*LogState)
			if now.Sub(state.LastLogTime) > 2*time.Minute {
				sl.logs.Delete(key)
			}
			return true
		})
	}
}
