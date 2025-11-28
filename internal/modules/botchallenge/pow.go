// Package botchallenge - Proof of Work
package botchallenge

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

const DefaultDifficulty = 3

type POW struct {
	Difficulty int
}

func NewPOW(difficulty int) *POW {
	if difficulty < 0 {
		difficulty = 0
	}
	// SHA-256 hex string max length is 64 characters
	if difficulty > 64 {
		difficulty = 64
	}
	return &POW{Difficulty: difficulty}
}

func (p *POW) GenerateSalt() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (p *POW) VerifyPOW(salt string, nonce int) bool {
	data := fmt.Sprintf("%s%d", salt, nonce)
	hash := sha256.Sum256([]byte(data))
	hashStr := hex.EncodeToString(hash[:])

	prefix := strings.Repeat("0", p.Difficulty)
	return strings.HasPrefix(hashStr, prefix)
}
