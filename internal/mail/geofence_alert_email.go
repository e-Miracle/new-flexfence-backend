package mail

import (
	"context"
	"fmt"
)

// SendGeofenceAlertEmail notifies an attendee about a geofence-related event.
func SendGeofenceAlertEmail(ctx context.Context, mailer Mailer, toEmail, title, body string) error {
	subject := fmt.Sprintf("FlexFence: %s", title)
	text := fmt.Sprintf(`Hello,

%s

%s

— FlexFence
`, title, body)
	return mailer.Send(ctx, Message{
		To:       []string{toEmail},
		Subject:  subject,
		TextBody: text,
	})
}
