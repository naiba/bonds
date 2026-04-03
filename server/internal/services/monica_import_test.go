package services

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestParseMonicaExport_ValidFixture(t *testing.T) {
	data, err := os.ReadFile("../testdata/monica_export.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}
	export, err := ParseMonicaExport(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if export.Version != "1.0-preview.1" {
		t.Errorf("expected version 1.0-preview.1, got %s", export.Version)
	}
	// u9a8cu8bc1 3 u4e2au8054u7cfbu4eba
	contacts := getCollectionByType(export.Account.Data, "contacts")
	if len(contacts) != 3 {
		t.Errorf("expected 3 contacts, got %d", len(contacts))
	}
	// u9a8cu8bc1 2 u6761 relationship
	relationships := getCollectionByType(export.Account.Data, "relationships")
	if len(relationships) != 2 {
		t.Errorf("expected 2 relationships, got %d", len(relationships))
	}
	// u9a8cu8bc1 instance.genders u975eu7a7a
	if len(export.Account.Instance.Genders) == 0 {
		t.Error("expected at least 1 gender in instance")
	}
	// u89e3u6790u7b2cu4e00u4e2au8054u7cfbu4ebauff08Johnuff09u5e76u9a8cu8bc1u5b50u8d44u6e90
	var john MonicaContact
	if err := json.Unmarshal(contacts[0], &john); err != nil {
		t.Fatalf("failed to unmarshal contact: %v", err)
	}
	if john.Properties.Birthdate == nil {
		t.Error("expected John to have birthdate")
	}
	notes := getCollectionByType(john.Data, "notes")
	if len(notes) == 0 {
		t.Error("expected John to have at least 1 note")
	}
}

func TestParseMonicaExport_InvalidJSON(t *testing.T) {
	_, err := ParseMonicaExport([]byte("not valid json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseMonicaExport_WrongVersion(t *testing.T) {
	jsonData := `{"version": "2.0", "account": {"uuid": "test"}}`
	_, err := ParseMonicaExport([]byte(jsonData))
	if err == nil {
		t.Error("expected error for wrong version")
	}
	if !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("expected 'unsupported' in error message, got: %v", err)
	}
}

func TestParseMonicaExport_EmptyContacts(t *testing.T) {
	jsonData := `{"version": "1.0-preview.1", "account": {"uuid": "test", "data": [], "instance": {}}}`
	export, err := ParseMonicaExport([]byte(jsonData))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	contacts := getCollectionByType(export.Account.Data, "contacts")
	if len(contacts) != 0 {
		t.Errorf("expected 0 contacts, got %d", len(contacts))
	}
}
