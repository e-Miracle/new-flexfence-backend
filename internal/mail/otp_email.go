package mail

import (
	"context"
	"fmt"
)

// SendBusinessLoginOTP emails a dashboard sign-in verification code.
func SendBusinessLoginOTP(ctx context.Context, mailer Mailer, toEmail, code string, expireMinutes int, dashboardURL string) error {
	subject := "Your FlexFence sign-in code"
	body := fmt.Sprintf(`Hello,

Your FlexFence verification code is: %s

This code expires in %d minutes. If you did not try to sign in, you can ignore this email.
`, code, expireMinutes)
	if dashboardURL != "" {
		body += fmt.Sprintf("\nSign in: %s\n", dashboardURL)
	}
	body += "\n— FlexFence\n"

	return mailer.Send(ctx, Message{
		To:       []string{toEmail},
		Subject:  subject,
		TextBody: body,
	})
}
