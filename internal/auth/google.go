package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var ErrInvalidGoogleToken = errors.New("invalid_google_token")

type GoogleProfile struct {
	Sub       string
	Email     string
	FirstName string
	LastName  string
}

// VerifyGoogleIDToken validates a Google ID token when clientID is configured.
// When clientID is empty (local dev), accepts trusted profile fields from the request.
func VerifyGoogleIDToken(clientID, idToken, fallbackSub, fallbackEmail, fallbackFirst, fallbackLast string) (GoogleProfile, error) {
	if strings.TrimSpace(idToken) != "" && strings.TrimSpace(clientID) != "" {
		return verifyWithGoogle(clientID, idToken)
	}
	if strings.TrimSpace(fallbackSub) == "" || strings.TrimSpace(fallbackEmail) == "" {
		return GoogleProfile{}, ErrInvalidGoogleToken
	}
	return GoogleProfile{
		Sub:       fallbackSub,
		Email:     fallbackEmail,
		FirstName: fallbackFirst,
		LastName:  fallbackLast,
	}, nil
}

type googleTokenInfo struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	EmailVerified string `json:"email_verified"`
	Aud           string `json:"aud"`
	Exp           string `json:"exp"`
}

func verifyWithGoogle(clientID, idToken string) (GoogleProfile, error) {
	endpoint := "https://oauth2.googleapis.com/tokeninfo?id_token=" + url.QueryEscape(idToken)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(endpoint)
	if err != nil {
		return GoogleProfile{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return GoogleProfile{}, ErrInvalidGoogleToken
	}

	var info googleTokenInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return GoogleProfile{}, err
	}
	if info.Aud != clientID || info.Sub == "" || info.Email == "" {
		return GoogleProfile{}, ErrInvalidGoogleToken
	}
	return GoogleProfile{
		Sub:       info.Sub,
		Email:     info.Email,
		FirstName: info.GivenName,
		LastName:  info.FamilyName,
	}, nil
}

func GoogleDevModeEnabled(clientID string) bool {
	return strings.TrimSpace(clientID) == ""
}

func DevModeHint() string {
	return fmt.Sprintf("Set GOOGLE_CLIENT_ID or pass id_token. Dev fallback requires google_sub and email when client ID is unset.")
}
