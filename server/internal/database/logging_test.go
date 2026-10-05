package database

import (
	"bytes"
	"log"
	"strings"
	"testing"

	"github.com/naiba/bonds/internal/config"
	"github.com/naiba/bonds/internal/models"
)

func TestDebugSQLDoesNotLogOAuthCredentials(t *testing.T) {
	var output bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&output)
	t.Cleanup(func() { log.SetOutput(previous) })
	db, err := Connect(&config.DatabaseConfig{Driver: "sqlite", DSN: ":memory:"}, true)
	if err != nil {
		t.Fatal(err)
	}
	conn, _ := db.DB()
	t.Cleanup(func() { _ = conn.Close() })
	if err := db.AutoMigrate(&models.OAuthProvider{}); err != nil {
		t.Fatal(err)
	}
	p := models.OAuthProvider{Name: "blog", Type: "oidc", ClientID: "sensitive-client-id", ClientSecret: "sensitive-client-secret"}
	if err := db.Create(&p).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&p).Update("client_secret", "replacement-secret").Error; err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{p.ClientID, "sensitive-client-secret", "replacement-secret"} {
		if strings.Contains(output.String(), secret) {
			t.Fatal("SQL logger exposed a bound parameter")
		}
	}
	if !strings.Contains(output.String(), "INSERT INTO") || !strings.Contains(output.String(), "UPDATE") {
		t.Fatal("debug SQL structure should still be available")
	}
}
