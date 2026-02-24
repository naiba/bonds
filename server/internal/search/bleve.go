package search

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/lang/cjk"
	"github.com/blevesearch/bleve/v2/mapping"
	bleveSearch "github.com/blevesearch/bleve/v2/search"
)

type BleveEngine struct {
	index     bleve.Index
	indexPath string
}

type bleveDocument struct {
	EntityType  string `json:"entity_type"`
	VaultID     string `json:"vault_id"`
	FirstName   string `json:"first_name,omitempty"`
	LastName    string `json:"last_name,omitempty"`
	Nickname    string `json:"nickname,omitempty"`
	JobPosition string `json:"job_position,omitempty"`
	ContactID   string `json:"contact_id,omitempty"`
	Title       string `json:"title,omitempty"`
	Body        string `json:"body,omitempty"`
}

func newIndexMapping() mapping.IndexMapping {
	indexMapping := bleve.NewIndexMapping()
	indexMapping.DefaultAnalyzer = cjk.AnalyzerName

	keywordFieldMapping := bleve.NewKeywordFieldMapping()

	defaultMapping := bleve.NewDocumentMapping()
	defaultMapping.AddFieldMappingsAt("entity_type", keywordFieldMapping)
	defaultMapping.AddFieldMappingsAt("vault_id", keywordFieldMapping)
	defaultMapping.AddFieldMappingsAt("contact_id", keywordFieldMapping)
	defaultMapping.AddFieldMappingsAt("first_name", bleve.NewTextFieldMapping())
	defaultMapping.AddFieldMappingsAt("last_name", bleve.NewTextFieldMapping())
	defaultMapping.AddFieldMappingsAt("nickname", bleve.NewTextFieldMapping())
	defaultMapping.AddFieldMappingsAt("job_position", bleve.NewTextFieldMapping())
	defaultMapping.AddFieldMappingsAt("title", bleve.NewTextFieldMapping())
	defaultMapping.AddFieldMappingsAt("body", bleve.NewTextFieldMapping())

	indexMapping.DefaultMapping = defaultMapping

	return indexMapping
}

func NewBleveEngine(indexPath string) (*BleveEngine, error) {
	var idx bleve.Index
	var err error

	if _, statErr := os.Stat(indexPath); os.IsNotExist(statErr) {
		idx, err = bleve.New(indexPath, newIndexMapping())
	} else {
		idx, err = bleve.Open(indexPath)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to open/create bleve index: %w", err)
	}
	return &BleveEngine{index: idx, indexPath: indexPath}, nil
}

func (e *BleveEngine) IndexContact(id, vaultID, firstName, lastName, nickname, jobPosition string) error {
	doc := bleveDocument{
		EntityType:  "contact",
		VaultID:     vaultID,
		FirstName:   firstName,
		LastName:    lastName,
		Nickname:    nickname,
		JobPosition: jobPosition,
	}
	return e.index.Index("contact:"+id, doc)
}

func (e *BleveEngine) IndexNote(id string, vaultID, contactID, title, body string) error {
	doc := bleveDocument{
		EntityType: "note",
		VaultID:    vaultID,
		ContactID:  contactID,
		Title:      title,
		Body:       body,
	}
	return e.index.Index("note:"+id, doc)
}

func (e *BleveEngine) DeleteDocument(id string) error {
	return e.index.Delete(id)
}

func (e *BleveEngine) Search(vaultID, query string, limit, offset int) (*SearchResponse, error) {
	start := time.Now()

	matchQuery := bleve.NewMatchQuery(query)
	vaultQuery := bleve.NewTermQuery(vaultID)
	vaultQuery.SetField("vault_id")

	conjunction := bleve.NewConjunctionQuery(matchQuery, vaultQuery)

	searchRequest := bleve.NewSearchRequestOptions(conjunction, limit, offset, false)
	searchRequest.Highlight = bleve.NewHighlightWithStyle("html")
	searchRequest.Fields = []string{"entity_type", "vault_id", "first_name", "last_name", "nickname", "title", "body", "contact_id"}

	searchResult, err := e.index.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	var contacts, notes []SearchResult
	for _, hit := range searchResult.Hits {
		entityType := entityTypeFromHit(hit)
		id := stripIDPrefix(hit.ID)
		r := SearchResult{
			ID:         id,
			Type:       entityType,
			Name:       nameFromHit(hit),
			Score:      hit.Score,
			Highlights: fragmentsToMap(hit.Fragments),
		}
		switch entityType {
		case "contact":
			contacts = append(contacts, r)
		case "note":
			notes = append(notes, r)
		}
	}
	if contacts == nil {
		contacts = []SearchResult{}
	}
	if notes == nil {
		notes = []SearchResult{}
	}

	return &SearchResponse{
		Contacts: contacts,
		Notes:    notes,
		Total:    int(searchResult.Total),
		TookMs:   time.Since(start).Milliseconds(),
	}, nil
}

func (e *BleveEngine) Close() error {
	return e.index.Close()
}

func (e *BleveEngine) Rebuild() error {
	if err := e.index.Close(); err != nil {
		return fmt.Errorf("failed to close bleve index: %w", err)
	}
	if err := os.RemoveAll(e.indexPath); err != nil {
		return fmt.Errorf("failed to remove bleve index directory: %w", err)
	}
	idx, err := bleve.New(e.indexPath, newIndexMapping())
	if err != nil {
		return fmt.Errorf("failed to recreate bleve index: %w", err)
	}
	e.index = idx
	return nil
}

func entityTypeFromHit(hit *bleveSearch.DocumentMatch) string {
	if hit.Fields != nil {
		if et, ok := hit.Fields["entity_type"].(string); ok {
			return et
		}
	}
	return ""
}

func stripIDPrefix(id string) string {
	if idx := strings.IndexByte(id, ':'); idx >= 0 {
		return id[idx+1:]
	}
	return id
}

func nameFromHit(hit *bleveSearch.DocumentMatch) string {
	if hit.Fields == nil {
		return ""
	}
	entityType, _ := hit.Fields["entity_type"].(string)
	if entityType == "contact" {
		first, _ := hit.Fields["first_name"].(string)
		last, _ := hit.Fields["last_name"].(string)
		name := strings.TrimSpace(first + " " + last)
		if name == "" {
			name, _ = hit.Fields["nickname"].(string)
		}
		return name
	}
	title, _ := hit.Fields["title"].(string)
	return title
}

func fragmentsToMap(fragments bleveSearch.FieldFragmentMap) map[string][]string {
	if len(fragments) == 0 {
		return nil
	}
	result := make(map[string][]string, len(fragments))
	for k, v := range fragments {
		result[k] = v
	}
	return result
}
