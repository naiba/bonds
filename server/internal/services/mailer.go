package services

import (
	"errors"
	"fmt"
	"log"
	"net/smtp"
	"time"

	"github.com/jordan-wright/email"
	"github.com/naiba/bonds/internal/config"
)

var ErrMailerNotConfigured = errors.New("mailer not configured")

type Mailer interface {
	Send(to, subject, htmlBody string) error
	Close()
}

type SMTPMailer struct {
	pool *email.Pool
	from string
}

func NewSMTPMailer(cfg *config.SMTPConfig) (Mailer, error) {
	if cfg.Host == "" {
		log.Println("SMTP not configured, using NoopMailer")
		return &NoopMailer{}, nil
	}

	addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)

	pool, err := email.NewPool(addr, 4, auth)
	if err != nil {
		return nil, fmt.Errorf("failed to create SMTP pool: %w", err)
	}

	return &SMTPMailer{
		pool: pool,
		from: cfg.From,
	}, nil
}

func (m *SMTPMailer) Send(to, subject, htmlBody string) error {
	e := email.NewEmail()
	e.From = m.from
	e.To = []string{to}
	e.Subject = subject
	e.HTML = []byte(htmlBody)

	return m.pool.Send(e, 10*time.Second)
}

func (m *SMTPMailer) Close() {
	m.pool.Close()
}

type NoopMailer struct{}

func (m *NoopMailer) Send(to, subject, _ string) error {
	log.Printf("[NoopMailer] Would send email to=%s subject=%q", to, subject)
	return nil
}

func (m *NoopMailer) Close() {}
