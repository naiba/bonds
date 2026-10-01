package search

import (
	"strings"
	"testing"

	bleveSearch "github.com/blevesearch/bleve/v2/search"
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
	for _, r := range resp.Contacts {
		if r.ID == "c1" {
			found = true
			if r.Type != "contact" {
				t.Errorf("Expected type 'contact', got '%s'", r.Type)
			}
			if r.Name != "Alice Smith" {
				t.Errorf("Expected name 'Alice Smith', got '%s'", r.Name)
			}
			break
		}
	}
	if !found {
		t.Error("Expected to find c1 in contacts")
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
	if len(resp.Contacts) != 1 || resp.Contacts[0].ID != "c1" {
		t.Errorf("Expected contact c1 in vault-a, got %+v", resp.Contacts)
	}
}

func TestSearch_returns_note_contact_id_when_note_matches(t *testing.T) {
	// Given
	engine, err := NewBleveEngine(t.TempDir() + "/test.bleve")
	if err != nil {
		t.Fatalf("NewBleveEngine failed: %v", err)
	}
	defer engine.Close()
	if err := engine.IndexNote("n1", "v1", "c1", "Project update", "Milestone reached"); err != nil {
		t.Fatalf("IndexNote failed: %v", err)
	}

	// When
	resp, err := engine.Search("v1", "Milestone", 10, 0)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	// Then
	if len(resp.Notes) != 1 {
		t.Fatalf("expected 1 note result, got %+v", resp.Notes)
	}
	if resp.Notes[0].ContactID != "c1" {
		t.Fatalf("note contact_id = %q, want %q", resp.Notes[0].ContactID, "c1")
	}
}

func TestSearchUntitledNoteUsesBoundedBodyPreview(t *testing.T) {
	engine, err := NewBleveEngine(t.TempDir() + "/test.bleve")
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()
	if err := engine.IndexNote("n1", "v1", "c1", "", "Favorite   tea\nis oolong"); err != nil {
		t.Fatal(err)
	}
	if err := engine.IndexNote("n2", "v1", "c2", "Named note", "Favorite tea is green"); err != nil {
		t.Fatal(err)
	}
	if err := engine.IndexNote("n3", "v2", "c3", "", "Favorite tea is private"); err != nil {
		t.Fatal(err)
	}
	resp, err := engine.Search("v1", "Favorite", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Notes) != 2 {
		t.Fatalf("notes = %+v, want two notes from v1", resp.Notes)
	}
	for _, note := range resp.Notes {
		switch note.ID {
		case "n1":
			if note.Name != "Favorite tea is oolong" {
				t.Errorf("untitled note name = %q", note.Name)
			}
		case "n2":
			if note.Name != "Named note" {
				t.Errorf("named note name = %q", note.Name)
			}
		default:
			t.Errorf("unexpected note from other vault: %+v", note)
		}
	}
	preview := nameFromHit(&bleveSearch.DocumentMatch{Fields: map[string]any{
		"entity_type": "note", "body": strings.Repeat("茶", 105),
	}})
	if preview != strings.Repeat("茶", 100)+"…" {
		t.Errorf("long unicode preview = %q", preview)
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

// Bug #31: Partial/prefix search must return results.
// e.g. searching "Ali" should find "Alice", "Bet" should find "Beton".
func TestSearchPartialMatch(t *testing.T) {
	dir := t.TempDir()
	engine, err := NewBleveEngine(dir + "/test.bleve")
	if err != nil {
		t.Fatalf("NewBleveEngine failed: %v", err)
	}
	defer engine.Close()

	if err := engine.IndexContact("c1", "v1", "Alice", "Smith", "", "Engineer"); err != nil {
		t.Fatalf("IndexContact failed: %v", err)
	}
	if err := engine.IndexContact("c2", "v1", "Jantje", "Beton", "", ""); err != nil {
		t.Fatalf("IndexContact failed: %v", err)
	}
	if err := engine.IndexNote("n1", "v1", "c1", "Meeting notes", "Discussion about project"); err != nil {
		t.Fatalf("IndexNote failed: %v", err)
	}

	tests := []struct {
		name   string
		query  string
		wantID string
		wantOK bool
	}{
		{"prefix of first name", "Ali", "c1", true},
		{"prefix of last name", "Smi", "c1", true},
		{"prefix of last name Beton", "Bet", "c2", true},
		{"prefix of note title", "Meet", "n1", true},
		{"full match still works", "Alice", "c1", true},
		{"no match", "Zzzzz", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := engine.Search("v1", tt.query, 10, 0)
			if err != nil {
				t.Fatalf("Search failed: %v", err)
			}
			if !tt.wantOK {
				if resp.Total != 0 {
					t.Errorf("Expected 0 results for %q, got %d", tt.query, resp.Total)
				}
				return
			}
			if resp.Total == 0 {
				t.Fatalf("Expected results for prefix %q, got 0", tt.query)
			}
			found := false
			for _, r := range resp.Contacts {
				if r.ID == tt.wantID {
					found = true
				}
			}
			for _, r := range resp.Notes {
				if r.ID == tt.wantID {
					found = true
				}
			}
			if !found {
				t.Errorf("Expected to find %s for query %q, contacts=%+v notes=%+v",
					tt.wantID, tt.query, resp.Contacts, resp.Notes)
			}
		})
	}
}
