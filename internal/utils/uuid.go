// Package utils - UUID
package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
)

func GenerateID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		log.Printf("failed to generate random ID: %v", err)
		return ""
	}

	return hex.EncodeToString(b)
}

func GenerateRayID() string {
	return fmt.Sprintf("%s-aegirus", GenerateID())
}
