package search

import (
	"testing"
)

func TestIndexAndSearch(t *testing.T) {
	dir := t.TempDir()
	engine, err := NewBleveEngine(dir + "/test.bleve")
	if err != nil {
		t.Fatalf("NewBleveEngine failed: %v", err)
	}
	defer engine.Close()

	if err := engine.IndexContact("c1", "v1", "Alice", "Smith", "", "Engineer"); err != nil {
		t.Fatalf("IndexContact failed: %v", err)
	}
	if err := engine.IndexContact("c2", "v1", "Bob", "Johnson", "Bobby", ""); err != nil {
		t.Fatalf("IndexContact failed: %v", err)
	}
	if err := engine.IndexContact("c3", "v1", "Charlie", "Brown", "", "Designer"); err != nil {
		t.Fatalf("IndexContact failed: %v", err)
	}

	resp, err := engine.Search("v1", "Alice", 10, 0)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if resp.Total == 0 {
		t.Fatal("Expected at least 1 result for 'Alice'")
	}

	found := false
	for _, r := range resp.Results {
		if r.ID == "contact:c1" {
			found = true
			if r.Type != "contact" {
				t.Errorf("Expected type 'contact', got '%s'", r.Type)
			}
			break
		}
	}
	if !found {
		t.Error("Expected to find contact:c1 in results")
	}
}

func TestSearchChinese(t *testing.T) {
	dir := t.TempDir()
	engine, err := NewBleveEngine(dir + "/test.bleve")
	if err != nil {
		t.Fatalf("NewBleveEngine failed: %v", err)
	}
	defer engine.Close()

	if err := engine.IndexContact("c1", "v1", "张三", "李四", "", "工程师"); err != nil {
		t.Fatalf("IndexContact failed: %v", err)
	}

	resp, err := engine.Search("v1", "张三", 10, 0)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if resp.Total == 0 {
		t.Fatal("Expected at least 1 result for Chinese name '张三'")
	}
}

func TestSearchFiltersVault(t *testing.T) {
	dir := t.TempDir()
	engine, err := NewBleveEngine(dir + "/test.bleve")
	if err != nil {
		t.Fatalf("NewBleveEngine failed: %v", err)
	}
	defer engine.Close()

	if err := engine.IndexContact("c1", "vault-a", "Alice", "Smith", "", ""); err != nil {
		t.Fatalf("IndexContact failed: %v", err)
	}
	if err := engine.IndexContact("c2", "vault-b", "Alice", "Jones", "", ""); err != nil {
		t.Fatalf("IndexContact failed: %v", err)
	}

	resp, err := engine.Search("vault-a", "Alice", 10, 0)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if resp.Total != 1 {
		t.Fatalf("Expected 1 result in vault-a, got %d", resp.Total)
	}
	if resp.Results[0].ID != "contact:c1" {
		t.Errorf("Expected contact:c1, got '%s'", resp.Results[0].ID)
	}
}

func TestDeleteDocument(t *testing.T) {
	dir := t.TempDir()
	engine, err := NewBleveEngine(dir + "/test.bleve")
	if err != nil {
		t.Fatalf("NewBleveEngine failed: %v", err)
	}
	defer engine.Close()

	if err := engine.IndexContact("c1", "v1", "Alice", "Smith", "", ""); err != nil {
		t.Fatalf("IndexContact failed: %v", err)
	}

	resp, err := engine.Search("v1", "Alice", 10, 0)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if resp.Total == 0 {
		t.Fatal("Expected to find Alice before deletion")
	}

	if err := engine.DeleteDocument("contact:c1"); err != nil {
		t.Fatalf("DeleteDocument failed: %v", err)
	}

	resp, err = engine.Search("v1", "Alice", 10, 0)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if resp.Total != 0 {
		t.Errorf("Expected 0 results after deletion, got %d", resp.Total)
	}
}
