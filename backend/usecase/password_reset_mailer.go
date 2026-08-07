package usecase

import (
	"context"
	"fmt"
	"net/smtp"
	"net/url"
	"os"
	"strings"
)

type smtpPasswordResetMailer struct {
	host, port, username, password, from, frontendBaseURL string
}

func NewPasswordResetMailerFromEnv() IPasswordResetMailer {
	return &smtpPasswordResetMailer{
		host: strings.TrimSpace(os.Getenv("SMTP_HOST")), port: strings.TrimSpace(os.Getenv("SMTP_PORT")),
		username: strings.TrimSpace(os.Getenv("SMTP_USERNAME")), password: os.Getenv("SMTP_PASSWORD"),
		from: strings.TrimSpace(os.Getenv("SMTP_FROM")), frontendBaseURL: strings.TrimRight(strings.TrimSpace(os.Getenv("FRONTEND_BASE_URL")), "/"),
	}
}

func (m *smtpPasswordResetMailer) SendPasswordReset(ctx context.Context, email, token string) error {
	if m.host == "" || m.from == "" || m.frontendBaseURL == "" {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	port := m.port
	if port == "" {
		port = "587"
	}
	link := m.frontendBaseURL + "/reset-password?token=" + url.QueryEscape(token)
	body := "From: " + m.from + "\r\nTo: " + email + "\r\nSubject: Music Discovery password reset\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n以下のリンクから30分以内にパスワードを再設定してください。\r\n" + link + "\r\n"
	var auth smtp.Auth
	if m.username != "" {
		auth = smtp.PlainAuth("", m.username, m.password, m.host)
	}
	if err := smtp.SendMail(fmt.Sprintf("%s:%s", m.host, port), auth, m.from, []string{email}, []byte(body)); err != nil {
		return err
	}
	return nil
}
