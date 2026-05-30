package auth

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// GenerateNumericOTP returns a zero-padded numeric code of the given length.
func GenerateNumericOTP(length int) (string, error) {
	if length < 4 || length > 8 {
		return "", fmt.Errorf("otp length must be between 4 and 8")
	}
	max := int64(1)
	for i := 0; i < length; i++ {
		max *= 10
	}
	n, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		return "", err
	}
	format := fmt.Sprintf("%%0%dd", length)
	return fmt.Sprintf(format, n.Int64()), nil
}

// MaskEmail hides part of the local address for display (e.g. o***@acme.test).
func MaskEmail(email string) string {
	at := -1
	for i, c := range email {
		if c == '@' {
			at = i
			break
		}
	}
	if at <= 0 {
		return email
	}
	local := email[:at]
	domain := email[at:]
	if len(local) <= 1 {
		return "*" + domain
	}
	return string(local[0]) + "***" + domain
}
