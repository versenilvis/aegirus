// Package botchallenge - Token
package botchallenge

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"aegirus/internal/config"
	"aegirus/pkg/logger"

	"go.uber.org/zap"
)

type TokenManager struct {
	cfg config.BotChallengeConfig

	mu          sync.RWMutex
	currentKey  []byte
	previousKey []byte

	stop    chan struct{}
	stopped sync.Once
}

func NewTokenManager(cfg config.BotChallengeConfig) *TokenManager {
	tm := &TokenManager{
		cfg:  cfg,
		stop: make(chan struct{}),
	}
	tm.rotateKeys()
	go tm.startKeyRotation()
	return tm
}

func (tm *TokenManager) Stop() {
	tm.stopped.Do(func() {
		close(tm.stop)
	})
}

func (tm *TokenManager) startKeyRotation() {
	ticker := time.NewTicker(time.Duration(tm.cfg.KeyRotationInterval))
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			tm.rotateKeys()
		case <-tm.stop:
			return
		}
	}
}

func (tm *TokenManager) rotateKeys() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	epoch := time.Now().Unix() / int64(time.Duration(tm.cfg.KeyRotationInterval).Seconds())

	tm.currentKey = tm.deriveKey(epoch)
	tm.previousKey = tm.deriveKey(epoch - 1)

	logger.Log.Debug("TokenManager: Keys Rotated", zap.Int64("epoch", epoch))
}

func (tm *TokenManager) deriveKey(epoch int64) []byte {
	input := fmt.Sprintf("%s:%d", tm.cfg.SecretKey, epoch)
	hash := sha256.Sum256([]byte(input))
	return hash[:]
}

func (tm *TokenManager) Generate(ip string) string {
	// NOTE: Copy key and release lock immediately
	// Crypto operations should be done outside the lock
	tm.mu.RLock()
	key := tm.currentKey
	tm.mu.RUnlock()

	now := time.Now().Unix()
	data := fmt.Sprintf("%s:%d", ip, now)

	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	signature := hex.EncodeToString(h.Sum(nil))

	return fmt.Sprintf("%d.%s", now, signature)
}

func (tm *TokenManager) Validate(ip, tokenString string) bool {
	if tokenString == "" {
		return false
	}

	parts := strings.Split(tokenString, ".")
	if len(parts) != 2 {
		return false
	}

	tsStr, clientSig := parts[0], parts[1]
	ts, err := strconv.ParseInt(tsStr, 10, 64)
	if err != nil {
		return false
	}

	if time.Now().Unix()-ts >= int64(time.Duration(tm.cfg.CookieTTL).Seconds()) {
		return false
	}

	data := fmt.Sprintf("%s:%d", ip, ts)

	// NOTE: Snapshot keys inside lock, then release immediately
	// This improves concurrency under high load
	tm.mu.RLock()
	currKey := tm.currentKey
	prevKey := tm.previousKey
	tm.mu.RUnlock()

	if tm.checkSignature(currKey, data, clientSig) {
		return true
	}

	// NOTE: previousKey is always initialized in NewTokenManager -> rotateKeys,
	// so we don't need to check len > 0
	if tm.checkSignature(prevKey, data, clientSig) {
		return true
	}

	return false
}

func (tm *TokenManager) checkSignature(key []byte, data, clientSig string) bool {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	expectedSig := hex.EncodeToString(h.Sum(nil))
	return hmac.Equal([]byte(clientSig), []byte(expectedSig))
}
