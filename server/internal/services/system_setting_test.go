package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/testutil"
)

func TestSystemSetting_SetAndGet(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := NewSystemSettingService(db)

	if err := svc.Set("test.key", "hello"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	val, err := svc.Get("test.key")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val != "hello" {
		t.Errorf("expected 'hello', got '%s'", val)
	}
}

func TestSystemSetting_SetUpdatesExisting(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := NewSystemSettingService(db)

	if err := svc.Set("update.key", "v1"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	if err := svc.Set("update.key", "v2"); err != nil {
		t.Fatalf("Set update failed: %v", err)
	}

	val, err := svc.Get("update.key")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val != "v2" {
		t.Errorf("expected 'v2', got '%s'", val)
	}
}

func TestSystemSetting_GetNotFound(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := NewSystemSettingService(db)

	_, err := svc.Get("nonexistent.key")
	if err == nil {
		t.Fatal("expected error for nonexistent key")
	}
	if err != ErrSystemSettingNotFound {
		t.Errorf("expected ErrSystemSettingNotFound, got %v", err)
	}
}

func TestSystemSetting_GetWithDefault(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := NewSystemSettingService(db)

	val := svc.GetWithDefault("missing.key", "default_val")
	if val != "default_val" {
		t.Errorf("expected 'default_val', got '%s'", val)
	}

	if err := svc.Set("existing.key", "real_val"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	val = svc.GetWithDefault("existing.key", "default_val")
	if val != "real_val" {
		t.Errorf("expected 'real_val', got '%s'", val)
	}
}

func TestSystemSetting_GetBool(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := NewSystemSettingService(db)

	if !svc.GetBool("missing.bool", true) {
		t.Error("expected true for missing key with default true")
	}
	if svc.GetBool("missing.bool", false) {
		t.Error("expected false for missing key with default false")
	}

	if err := svc.Set("bool.key", "true"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	if !svc.GetBool("bool.key", false) {
		t.Error("expected true for value 'true'")
	}

	if err := svc.Set("bool.key2", "1"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	if !svc.GetBool("bool.key2", false) {
		t.Error("expected true for value '1'")
	}

	if err := svc.Set("bool.key3", "false"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	if svc.GetBool("bool.key3", true) {
		t.Error("expected false for value 'false'")
	}
}

func TestSystemSetting_GetAll(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := NewSystemSettingService(db)

	items, err := svc.GetAll()
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items initially, got %d", len(items))
	}

	if err := svc.Set("a.key", "alpha"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	if err := svc.Set("b.key", "beta"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	items, err = svc.GetAll()
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].Key != "a.key" {
		t.Errorf("expected first key 'a.key', got '%s'", items[0].Key)
	}
	if items[1].Key != "b.key" {
		t.Errorf("expected second key 'b.key', got '%s'", items[1].Key)
	}
}

func TestSystemSetting_BulkSet(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := NewSystemSettingService(db)

	if err := svc.Set("bulk.existing", "old"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	err := svc.BulkSet([]dto.SystemSettingItem{
		{Key: "bulk.existing", Value: "updated"},
		{Key: "bulk.new", Value: "created"},
	})
	if err != nil {
		t.Fatalf("BulkSet failed: %v", err)
	}

	val, err := svc.Get("bulk.existing")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val != "updated" {
		t.Errorf("expected 'updated', got '%s'", val)
	}

	val, err = svc.Get("bulk.new")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val != "created" {
		t.Errorf("expected 'created', got '%s'", val)
	}
}

func TestSystemSetting_Delete(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := NewSystemSettingService(db)

	if err := svc.Set("del.key", "value"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	if err := svc.Delete("del.key"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err := svc.Get("del.key")
	if err != ErrSystemSettingNotFound {
		t.Errorf("expected ErrSystemSettingNotFound after delete, got %v", err)
	}
}

func TestSystemSetting_DeleteNotFound(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := NewSystemSettingService(db)

	err := svc.Delete("nonexistent")
	if err != ErrSystemSettingNotFound {
		t.Errorf("expected ErrSystemSettingNotFound, got %v", err)
	}
}
