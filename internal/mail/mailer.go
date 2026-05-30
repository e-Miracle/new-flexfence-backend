package mail

import (
	"context"
	"fmt"
	"log"
	"net/smtp"
	"strings"
)

// Message is a single outbound email.
type Message struct {
	To       []string
	Subject  string
	TextBody string
}

// Mailer sends email messages.
type Mailer interface {
	Send(ctx context.Context, msg Message) error
	Enabled() bool
}

// SMTPConfig holds SMTP connection settings from environment variables.
type SMTPConfig struct {
	Host      string
	Port      int
	Username  string
	Password  string
	FromEmail string
	FromName  string
}

func (c SMTPConfig) Enabled() bool {
	return strings.TrimSpace(c.Host) != "" && strings.TrimSpace(c.FromEmail) != ""
}

// SMTPMailer sends mail via SMTP with STARTTLS (typical port 587).
type SMTPMailer struct {
	cfg SMTPConfig
}

func NewSMTPMailer(cfg SMTPConfig) *SMTPMailer {
	return &SMTPMailer{cfg: cfg}
}

func (m *SMTPMailer) Enabled() bool {
	return m.cfg.Enabled()
}

func (m *SMTPMailer) Send(_ context.Context, msg Message) error {
	if !m.Enabled() {
		return fmt.Errorf("smtp is not configured")
	}
	if len(msg.To) == 0 {
		return fmt.Errorf("no recipients")
	}

	from := m.cfg.FromEmail
	fromHeader := from
	if name := strings.TrimSpace(m.cfg.FromName); name != "" {
		fromHeader = fmt.Sprintf("%s <%s>", name, from)
	}

	var body strings.Builder
	body.WriteString(fmt.Sprintf("From: %s\r\n", fromHeader))
	body.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(msg.To, ", ")))
	body.WriteString(fmt.Sprintf("Subject: %s\r\n", msg.Subject))
	body.WriteString("MIME-Version: 1.0\r\n")
	body.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	body.WriteString("\r\n")
	body.WriteString(msg.TextBody)

	addr := fmt.Sprintf("%s:%d", m.cfg.Host, m.cfg.Port)
	auth := smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
	return smtp.SendMail(addr, auth, from, msg.To, []byte(body.String()))
}

// LoggingMailer logs messages when SMTP is not configured (local development).
type LoggingMailer struct{}

func (LoggingMailer) Enabled() bool { return true }

func (LoggingMailer) Send(_ context.Context, msg Message) error {
	log.Printf("mail (dev log): to=%v subject=%q\n%s", msg.To, msg.Subject, msg.TextBody)
	return nil
}

// NewFromConfig returns SMTP mailer when configured, otherwise a logging fallback.
func NewFromConfig(cfg SMTPConfig) Mailer {
	if cfg.Enabled() {
		return NewSMTPMailer(cfg)
	}
	return LoggingMailer{}
}
