package push

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const firebaseMessagingScope = "https://www.googleapis.com/auth/firebase.messaging"

// FCMSender delivers notifications via Firebase Cloud Messaging HTTP v1.
type FCMSender struct {
	projectID   string
	tokenSource oauth2.TokenSource
	client      *http.Client
}

// NewFCMSender builds a v1 sender from a Firebase project ID and service-account JSON file path.
// Returns nil when project ID or credentials path is empty (push disabled).
func NewFCMSender(projectID, credentialsPath string) *FCMSender {
	projectID = strings.TrimSpace(projectID)
	credentialsPath = strings.TrimSpace(credentialsPath)
	if projectID == "" || credentialsPath == "" {
		return nil
	}

	ctx := context.Background()
	raw, err := os.ReadFile(credentialsPath)
	if err != nil {
		fmt.Printf("fcm: could not read service account file %s: %v\n", credentialsPath, err)
		return nil
	}
	creds, err := google.CredentialsFromJSON(ctx, raw, firebaseMessagingScope)
	if err != nil {
		fmt.Printf("fcm: invalid service account JSON in %s: %v\n", credentialsPath, err)
		return nil
	}

	return &FCMSender{
		projectID:   projectID,
		tokenSource: creds.TokenSource,
		client:      &http.Client{Timeout: 15 * time.Second},
	}
}

func (f *FCMSender) Enabled() bool {
	return f != nil && f.projectID != "" && f.tokenSource != nil
}

type v1SendRequest struct {
	Message v1Message `json:"message"`
}

type v1Message struct {
	Token        string            `json:"token"`
	Notification *v1Notification   `json:"notification,omitempty"`
	Data         map[string]string `json:"data,omitempty"`
}

type v1Notification struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

// SendToDevice sends a push notification to one device token.
func (f *FCMSender) SendToDevice(ctx context.Context, token, title, body string) error {
	if !f.Enabled() {
		return fmt.Errorf("fcm not configured")
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return fmt.Errorf("device token is required")
	}

	accessToken, err := f.tokenSource.Token()
	if err != nil {
		return fmt.Errorf("fcm oauth token: %w", err)
	}

	payload := v1SendRequest{
		Message: v1Message{
			Token: token,
			Notification: &v1Notification{
				Title: strings.TrimSpace(title),
				Body:  strings.TrimSpace(body),
			},
			Data: map[string]string{
				"type": "geofence_alert",
			},
		},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://fcm.googleapis.com/v1/projects/%s/messages:send", f.projectID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken.AccessToken)
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")

	res, err := f.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return fmt.Errorf("fcm v1 status %d: %s", res.StatusCode, strings.TrimSpace(string(bodyBytes)))
	}
	return nil
}
