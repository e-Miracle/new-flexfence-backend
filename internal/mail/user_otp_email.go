package mail

import (
	"context"
	"fmt"
)

// SendUserEmailVerificationOTP emails a mobile sign-up verification code.
func SendUserEmailVerificationOTP(ctx context.Context, mailer Mailer, toEmail, code string, expireMinutes int) error {
	subject := "Verify your FlexFence email"
	body := fmt.Sprintf(`Hello,

Your FlexFence email verification code is: %s

This code expires in %d minutes. If you did not create a FlexFence account, you can ignore this email.

— FlexFence
`, code, expireMinutes)

	return mailer.Send(ctx, Message{
		To:       []string{toEmail},
		Subject:  subject,
		TextBody: body,
	})
}
