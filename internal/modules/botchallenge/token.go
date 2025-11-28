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
}

func NewTokenManager(cfg config.BotChallengeConfig) *TokenManager {
	tm := &TokenManager{
		cfg: cfg,
	}
	tm.rotateKeys()
	go tm.startKeyRotation()
	return tm
}

func (tm *TokenManager) startKeyRotation() {
	ticker := time.NewTicker(time.Duration(tm.cfg.KeyRotationInterval))
	for range ticker.C {
		tm.rotateKeys()
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

	if time.Now().Unix()-ts > int64(time.Duration(tm.cfg.CookieTTL).Seconds()) {
		return false
	}

	data := fmt.Sprintf("%s:%d", ip, ts)

	tm.mu.RLock()
	defer tm.mu.RUnlock()

	if tm.checkSignature(tm.currentKey, data, clientSig) {
		return true
	}

	if len(tm.previousKey) > 0 && tm.checkSignature(tm.previousKey, data, clientSig) {
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
