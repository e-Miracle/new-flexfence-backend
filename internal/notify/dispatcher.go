package notify

import (
	"context"
	"log"
	"strings"

	"github.com/flexfence/flexfence-backend/internal/domain"
	"github.com/flexfence/flexfence-backend/internal/mail"
	"github.com/flexfence/flexfence-backend/internal/push"
)

// PreferenceStore loads attendee notification settings and contact info.
type PreferenceStore interface {
	GetUserNotificationPreferences(userID string) (domain.UserNotificationPreferences, error)
	GetUserByID(id string) (domain.User, bool, error)
	ListUserDeviceTokens(userID string) ([]string, error)
}

// Dispatcher sends email and push based on user notification preferences.
type Dispatcher struct {
	identity PreferenceStore
	mailer   mail.Mailer
	fcm      *push.FCMSender
}

func NewDispatcher(identity PreferenceStore, mailer mail.Mailer, fcm *push.FCMSender) *Dispatcher {
	return &Dispatcher{
		identity: identity,
		mailer:   mailer,
		fcm:      fcm,
	}
}

func (d *Dispatcher) DispatchGeofenceAlert(ctx context.Context, userID string, alert domain.GeofenceAlert) {
	if d == nil || d.identity == nil {
		return
	}
	prefs, err := d.identity.GetUserNotificationPreferences(userID)
	if err != nil {
		log.Printf("notify: prefs load failed user=%s: %v", userID, err)
		return
	}
	if !shouldDeliverGeofenceAlert(alert.Type, prefs) {
		return
	}
	title, body := alertTitleBody(alert)
	user, ok, err := d.identity.GetUserByID(userID)
	if err != nil || !ok {
		return
	}
	if prefs.EmailNotificationsEnabled && d.mailer != nil && d.mailer.Enabled() && strings.TrimSpace(user.Email) != "" {
		if err := mail.SendGeofenceAlertEmail(ctx, d.mailer, user.Email, title, body); err != nil {
			log.Printf("notify: email failed user=%s: %v", userID, err)
		}
	}
	if prefs.PushNotificationsEnabled && d.fcm != nil && d.fcm.Enabled() {
		tokens, err := d.identity.ListUserDeviceTokens(userID)
		if err != nil {
			return
		}
		for _, token := range tokens {
			if err := d.fcm.SendToDevice(ctx, token, title, body); err != nil {
				log.Printf("notify: push failed user=%s: %v", userID, err)
			}
		}
	}
}

func shouldDeliverGeofenceAlert(alertType string, prefs domain.UserNotificationPreferences) bool {
	switch alertType {
	case "missed_check_in", "missed_check_out":
		return prefs.MissedCheckInOutEnabled
	default:
		return prefs.GeofenceNotificationsEnabled
	}
}

func alertTitleBody(alert domain.GeofenceAlert) (string, string) {
	title := strings.TrimSpace(alert.EventTitle)
	if title == "" {
		title = "FlexFence"
	}
	body := strings.TrimSpace(alert.Message)
	if body == "" {
		body = "You have a new geofence notification."
	}
	switch alert.Type {
	case "event_live":
		return "Event is live", body
	case "fence_removed":
		return "Geofence removed", body
	default:
		return title, body
	}
}
