package services

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFallbackAuthTransport_BasicAuthSuccess(t *testing.T) {
	var receivedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	transport := &fallbackAuthTransport{
		username: "user",
		password: "pass",
	}
	client := &http.Client{Transport: transport}

	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
	if !strings.HasPrefix(receivedAuth, "Basic ") {
		t.Errorf("Expected Basic auth header, got '%s'", receivedAuth)
	}
	if transport.useDigest {
		t.Error("Expected useDigest to remain false")
	}
}

func TestFallbackAuthTransport_DigestFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")

		if strings.HasPrefix(auth, "Digest ") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
			return
		}

		w.Header().Set("Www-Authenticate", `Digest realm="test", nonce="abc123", qop="auth"`)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("unauthorized"))
	}))
	defer server.Close()

	transport := &fallbackAuthTransport{
		username: "user",
		password: "pass",
	}
	client := &http.Client{Transport: transport}

	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	if !transport.useDigest {
		t.Error("Expected useDigest to be true after fallback")
	}
}

func TestFallbackAuthTransport_DigestCached(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")

		if strings.HasPrefix(auth, "Digest ") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
			return
		}

		w.Header().Set("Www-Authenticate", `Digest realm="test", nonce="abc123", qop="auth"`)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("unauthorized"))
	}))
	defer server.Close()

	transport := &fallbackAuthTransport{
		username: "user",
		password: "pass",
	}
	client := &http.Client{Transport: transport}

	resp1, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("First request failed: %v", err)
	}
	resp1.Body.Close()

	if !transport.useDigest {
		t.Fatal("Expected useDigest after first request")
	}

	resp2, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("Second request failed: %v", err)
	}
	resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Errorf("Second request: expected 200, got %d", resp2.StatusCode)
	}
}

func TestFallbackAuthTransport_NonDigest401(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Www-Authenticate", `Bearer realm="api"`)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("unauthorized"))
	}))
	defer server.Close()

	transport := &fallbackAuthTransport{
		username: "user",
		password: "pass",
	}
	client := &http.Client{Transport: transport}

	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected 401 passthrough, got %d", resp.StatusCode)
	}
	if transport.useDigest {
		t.Error("Expected useDigest to remain false for non-digest 401")
	}
}
