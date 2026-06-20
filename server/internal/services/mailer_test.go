package services

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/naiba/bonds/internal/config"
	"github.com/naiba/bonds/internal/testutil"
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

func TestSMTPAuthReturnsNilWhenCredentialsEmpty(t *testing.T) {
	if auth := smtpAuth("smtp.example.com", "", ""); auth != nil {
		t.Fatalf("empty SMTP username and password must skip AUTH, got %T", auth)
	}
}

func TestSMTPAuthReturnsPlainAuthWhenCredentialsConfigured(t *testing.T) {
	if auth := smtpAuth("smtp.example.com", "user", ""); auth == nil {
		t.Fatal("configured SMTP username must enable AUTH")
	}
	if auth := smtpAuth("smtp.example.com", "", "password"); auth == nil {
		t.Fatal("configured SMTP password must enable AUTH")
	}
}

func TestDynamicMailerSendSkipsAuthForUnauthenticatedRelay(t *testing.T) {
	server := startFakeSMTPRelay(t)

	db := testutil.SetupTestDB(t)
	settings := NewSystemSettingService(db)

	if err := settings.Set("smtp.host", server.host); err != nil {
		t.Fatalf("failed to set smtp.host: %v", err)
	}
	if err := settings.Set("smtp.port", server.port); err != nil {
		t.Fatalf("failed to set smtp.port: %v", err)
	}
	if err := settings.Set("smtp.username", ""); err != nil {
		t.Fatalf("failed to set smtp.username: %v", err)
	}
	if err := settings.Set("smtp.password", ""); err != nil {
		t.Fatalf("failed to set smtp.password: %v", err)
	}
	if err := settings.Set("smtp.from", "sender@example.com"); err != nil {
		t.Fatalf("failed to set smtp.from: %v", err)
	}

	mailer := NewDynamicMailer(settings)
	if err := mailer.Send("recipient@example.com", "Subject", "<p>Hello</p>"); err != nil {
		t.Fatalf("DynamicMailer.Send returned error: %v", err)
	}

	server.assertNoAuth(t)
}

type fakeSMTPRelay struct {
	host string
	port string
	ln   net.Listener
	mu   sync.Mutex
	auth bool
}

func startFakeSMTPRelay(t *testing.T) *fakeSMTPRelay {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen on fake SMTP relay: %v", err)
	}

	relay := &fakeSMTPRelay{
		host: strings.Split(ln.Addr().String(), ":")[0],
		port: strings.Split(ln.Addr().String(), ":")[1],
		ln:   ln,
	}

	go relay.serve()

	t.Cleanup(func() {
		_ = ln.Close()
	})

	return relay
}

func (r *fakeSMTPRelay) serve() {
	for {
		conn, err := r.ln.Accept()
		if err != nil {
			return
		}
		go r.handleConn(conn)
	}
}

func (r *fakeSMTPRelay) handleConn(conn net.Conn) {
	defer conn.Close()

	writer := bufio.NewWriter(conn)
	reader := bufio.NewReader(conn)

	writeLine := func(format string, args ...any) {
		_, _ = fmt.Fprintf(writer, format+"\r\n", args...)
		_ = writer.Flush()
	}

	writeLine("220 fake-smtp-relay ESMTP ready")
	for {
		_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		command := strings.TrimSpace(line)
		upper := strings.ToUpper(command)

		switch {
		case strings.HasPrefix(upper, "EHLO "):
			writeLine("250-fake-smtp-relay greets you")
			writeLine("250-PIPELINING")
			writeLine("250-SIZE 35882577")
			writeLine("250-8BITMIME")
			writeLine("250 OK")
		case strings.HasPrefix(upper, "HELO "):
			writeLine("250 fake-smtp-relay greets you")
		case strings.HasPrefix(upper, "AUTH "):
			r.mu.Lock()
			r.auth = true
			r.mu.Unlock()
			writeLine("502 AUTH not supported")
			return
		case strings.HasPrefix(upper, "MAIL FROM:"):
			writeLine("250 OK")
		case strings.HasPrefix(upper, "RCPT TO:"):
			writeLine("250 OK")
		case upper == "DATA":
			writeLine("354 End data with <CR><LF>.<CR><LF>")
			for {
				_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
				dataLine, err := reader.ReadString('\n')
				if err != nil {
					return
				}
				if strings.TrimSpace(dataLine) == "." {
					break
				}
			}
			writeLine("250 OK")
		case upper == "QUIT":
			writeLine("221 Bye")
			return
		default:
			writeLine("250 OK")
		}
	}
}

func (r *fakeSMTPRelay) assertNoAuth(t *testing.T) {
	t.Helper()

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.auth {
		t.Fatal("unexpected AUTH command recorded by fake SMTP relay")
	}
}
