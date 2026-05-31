package mail

import (
	"context"
	"fmt"
)

// SendUserPasswordResetOTP emails a mobile password reset code.
func SendUserPasswordResetOTP(ctx context.Context, mailer Mailer, toEmail, code string, expireMinutes int) error {
	subject := "Reset your FlexFence password"
	body := fmt.Sprintf(`Hello,

Your FlexFence password reset code is: %s

This code expires in %d minutes. If you did not request a password reset, you can ignore this email.

— FlexFence
`, code, expireMinutes)

	return mailer.Send(ctx, Message{
		To:       []string{toEmail},
		Subject:  subject,
		TextBody: body,
	})
}
