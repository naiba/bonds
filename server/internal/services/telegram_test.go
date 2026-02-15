package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewTelegramServiceWithToken(t *testing.T) {
	svc, err := NewTelegramService("test-bot-token")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if svc == nil {
		t.Fatal("Expected non-nil service for valid token")
	}
	if !svc.IsAvailable() {
		t.Error("Expected IsAvailable to return true")
	}
}

func TestSendMessageSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("Failed to parse form: %v", err)
		}
		if r.FormValue("chat_id") != "12345" {
			t.Errorf("Expected chat_id '12345', got '%s'", r.FormValue("chat_id"))
		}
		if r.FormValue("text") != "hello" {
			t.Errorf("Expected text 'hello', got '%s'", r.FormValue("text"))
		}
		if r.FormValue("parse_mode") != "HTML" {
			t.Errorf("Expected parse_mode 'HTML', got '%s'", r.FormValue("parse_mode"))
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
	}))
	defer server.Close()

	svc := &TelegramService{
		botToken: "test-token",
		client:   server.Client(),
		apiBase:  server.URL,
	}
	err := svc.SendMessage(12345, "hello")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
}

func TestSendMessageAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":          false,
			"description": "Bad Request: chat not found",
		})
	}))
	defer server.Close()

	svc := &TelegramService{
		botToken: "test-token",
		client:   server.Client(),
		apiBase:  server.URL,
	}
	err := svc.SendMessage(12345, "hello")
	if err == nil {
		t.Fatal("Expected error for API failure")
	}
}

func TestFormatReminderMessage(t *testing.T) {
	msg := FormatReminderMessage("John Doe", "Call about birthday")
	expected := "🔔 <b>Reminder</b>\n\n<b>John Doe</b>: Call about birthday"
	if msg != expected {
		t.Errorf("Expected %q, got %q", expected, msg)
	}
}

func TestSendReminderSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("Failed to parse form: %v", err)
		}
		expectedText := FormatReminderMessage("Jane", "Birthday tomorrow")
		if r.FormValue("text") != expectedText {
			t.Errorf("Expected text %q, got %q", expectedText, r.FormValue("text"))
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
	}))
	defer server.Close()

	svc := &TelegramService{
		botToken: "test-token",
		client:   server.Client(),
		apiBase:  server.URL,
	}
	err := svc.SendReminder(12345, "Jane", "Birthday tomorrow")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
}
