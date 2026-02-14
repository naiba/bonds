package search

import (
	"fmt"
	"os"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/lang/cjk"
	"github.com/blevesearch/bleve/v2/mapping"
	bleveSearch "github.com/blevesearch/bleve/v2/search"
)

type BleveEngine struct {
	index bleve.Index
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
	return &BleveEngine{index: idx}, nil
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

	results := make([]SearchResult, len(searchResult.Hits))
	for i, hit := range searchResult.Hits {
		results[i] = SearchResult{
			ID:         hit.ID,
			Type:       entityTypeFromHit(hit),
			Score:      hit.Score,
			Highlights: fragmentsToMap(hit.Fragments),
		}
	}

	return &SearchResponse{
		Results: results,
		Total:   int(searchResult.Total),
		TookMs:  time.Since(start).Milliseconds(),
	}, nil
}

func (e *BleveEngine) Close() error {
	return e.index.Close()
}

func entityTypeFromHit(hit *bleveSearch.DocumentMatch) string {
	if hit.Fields != nil {
		if et, ok := hit.Fields["entity_type"].(string); ok {
			return et
		}
	}
	return ""
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
