package search

type SearchResult struct {
	ID         string              `json:"id"`
	Type       string              `json:"type"`
	Score      float64             `json:"score"`
	Highlights map[string][]string `json:"highlights,omitempty"`
}

type SearchResponse struct {
	Results []SearchResult `json:"results"`
	Total   int            `json:"total"`
	TookMs  int64          `json:"took_ms"`
}

type Engine interface {
	IndexContact(id, vaultID, firstName, lastName, nickname, jobPosition string) error
	IndexNote(id string, vaultID, contactID, title, body string) error
	DeleteDocument(id string) error
	Search(vaultID, query string, limit, offset int) (*SearchResponse, error)
	Close() error
}
