// Package utils - UUID
package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

func GenerateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func GenerateRayID() string {
	return fmt.Sprintf("%s-aegirus", GenerateID())
}
