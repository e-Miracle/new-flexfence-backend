package auth

import (
	"crypto/rand"
	"encoding/hex"
)

// GenerateQRToken returns a random token for event join QR codes.
func GenerateQRToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "qr_" + hex.EncodeToString(b), nil
}
