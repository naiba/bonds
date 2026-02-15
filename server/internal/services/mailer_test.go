package services

import (
	"testing"

	"github.com/naiba/bonds/internal/config"
)

func TestNewSMTPMailerWithEmptyConfig(t *testing.T) {
	cfg := &config.SMTPConfig{
		Host:     "",
		Port:     "587",
		Username: "",
		Password: "",
		From:     "",
	}

	mailer, err := NewSMTPMailer(cfg)
	if err != nil {
		t.Fatalf("NewSMTPMailer returned error: %v", err)
	}

	_, ok := mailer.(*NoopMailer)
	if !ok {
		t.Fatalf("Expected NoopMailer when SMTP host is empty, got %T", mailer)
	}
}

func TestNewSMTPMailerReturnsNoopWhenNotConfigured(t *testing.T) {
	cfg := &config.SMTPConfig{}

	mailer, err := NewSMTPMailer(cfg)
	if err != nil {
		t.Fatalf("NewSMTPMailer returned error: %v", err)
	}

	err = mailer.Send("user@example.com", "Hello", "<p>World</p>")
	if err != nil {
		t.Fatalf("NoopMailer Send returned error: %v", err)
	}

	mailer.Close()
}
