package services

import (
	"strings"
	"testing"
)

// Command-line arguments are readable by every local user, so the password
// must travel in the environment and never in --dbname.
func TestPostgresClientConnection_MovesThePasswordOutOfTheArguments(t *testing.T) {
	cases := []struct {
		name           string
		dsn            string
		wantConnection string
		wantPassword   string
	}{
		{"url", "postgres://bonds:s3cret@db.example.com:5432/bonds?sslmode=disable", "postgres://bonds@db.example.com:5432/bonds?sslmode=disable", "s3cret"},
		{"url with escaped characters", "postgresql://bonds:p%40ss%2Fword@db.example.com/bonds", "postgresql://bonds@db.example.com/bonds", "p@ss/word"},
		{"keywords", "host=db.example.com user=bonds password=s3cret dbname=bonds", "host=db.example.com user=bonds dbname=bonds", "s3cret"},
		{"quoted keyword value", `host=db.example.com password='two words \'quoted\'' dbname=bonds`, "host=db.example.com dbname=bonds", "two words 'quoted'"},
		{"password first", "password=s3cret host=db.example.com", "host=db.example.com", "s3cret"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			connection, environment := postgresClientConnection(tc.dsn)
			if connection != tc.wantConnection {
				t.Errorf("connection = %q, want %q", connection, tc.wantConnection)
			}
			if strings.Contains(connection, tc.wantPassword) {
				t.Errorf("connection %q still contains the password", connection)
			}
			if len(environment) != 1 || environment[0] != "PGPASSWORD="+tc.wantPassword {
				t.Errorf("environment = %q, want PGPASSWORD=%s", environment, tc.wantPassword)
			}
		})
	}
}

func TestPostgresClientConnection_LeavesPasswordlessDSNAlone(t *testing.T) {
	for _, dsn := range []string{
		"postgres://bonds@db.example.com/bonds",
		"host=db.example.com user=bonds dbname=bonds",
		"host=db.example.com user=bonds passfile=/run/secrets/pgpass",
	} {
		connection, environment := postgresClientConnection(dsn)
		if connection != dsn || len(environment) != 0 {
			t.Errorf("postgresClientConnection(%q) = %q, %q; want it unchanged", dsn, connection, environment)
		}
	}
}
