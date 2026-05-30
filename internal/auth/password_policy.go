package auth

import (
	"strings"
	"unicode"
)

const MinPasswordLength = 8

// PasswordRequirements describes the password policy for clients and documentation.
var PasswordRequirements = []string{
	"At least 8 characters",
	"At least one uppercase letter (A–Z)",
	"At least one lowercase letter (a–z)",
	"At least one number (0–9)",
	"At least one special character (e.g. !@#$%)",
	"Not a commonly used password",
}

// PasswordValidationError is returned when a password fails policy checks.
type PasswordValidationError struct {
	Code    string
	Message string
}

func (e *PasswordValidationError) Error() string {
	return e.Message
}

// ValidatePassword checks password strength. Returns nil when the password is acceptable.
func ValidatePassword(password string) *PasswordValidationError {
	var missing []string

	if len(password) < MinPasswordLength {
		return &PasswordValidationError{
			Code:    "password_too_short",
			Message: "Password must be at least 8 characters",
		}
	}
	if !hasUppercase(password) {
		missing = append(missing, "an uppercase letter")
	}
	if !hasLowercase(password) {
		missing = append(missing, "a lowercase letter")
	}
	if !hasDigit(password) {
		missing = append(missing, "a number")
	}
	if !hasSpecialChar(password) {
		missing = append(missing, "a special character")
	}
	if len(missing) > 0 {
		return &PasswordValidationError{
			Code:    "password_requirements_not_met",
			Message: "Password must include " + joinRequirements(missing),
		}
	}
	if isCommonPassword(password) {
		return &PasswordValidationError{
			Code:    "password_too_common",
			Message: "This password is too common. Choose a stronger, unique password.",
		}
	}
	return nil
}

func joinRequirements(parts []string) string {
	switch len(parts) {
	case 0:
		return ""
	case 1:
		return parts[0]
	case 2:
		return parts[0] + " and " + parts[1]
	default:
		return strings.Join(parts[:len(parts)-1], ", ") + ", and " + parts[len(parts)-1]
	}
}

func hasUppercase(s string) bool {
	for _, r := range s {
		if unicode.IsUpper(r) {
			return true
		}
	}
	return false
}

func hasLowercase(s string) bool {
	for _, r := range s {
		if unicode.IsLower(r) {
			return true
		}
	}
	return false
}

func hasDigit(s string) bool {
	for _, r := range s {
		if unicode.IsDigit(r) {
			return true
		}
	}
	return false
}

func hasSpecialChar(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return true
		}
	}
	return false
}

var commonPasswords = map[string]struct{}{
	"password":       {},
	"password1":      {},
	"password12":     {},
	"password123":    {},
	"password1234":   {},
	"12345678":       {},
	"123456789":      {},
	"1234567890":     {},
	"qwerty123":      {},
	"qwertyuiop":     {},
	"admin123":       {},
	"letmein":        {},
	"welcome":        {},
	"welcome1":       {},
	"iloveyou":       {},
	"monkey123":      {},
	"dragon123":      {},
	"football":       {},
	"baseball":       {},
	"abc12345":       {},
	"changeme":       {},
	"trustno1":       {},
	"passw0rd":       {},
	"flexfence":      {},
	"flexfence123":   {},
	"password!":      {},
	"password!1":     {},
	"qwerty!123":     {},
	"administrator":  {},
	"login123":       {},
	"secret123":      {},
	"test1234":       {},
	"welcome123":     {},
	"password@1":     {},
	"password@123":   {},
	"password1!":     {},
	"welcome1!":      {},
	"admin123!":      {},
}

func isCommonPassword(password string) bool {
	normalized := strings.ToLower(strings.TrimSpace(password))
	_, found := commonPasswords[normalized]
	return found
}
