package search

type NoopEngine struct{}

func (e *NoopEngine) IndexContact(id, vaultID, firstName, lastName, nickname, jobPosition string) error {
	return nil
}

func (e *NoopEngine) IndexNote(id string, vaultID, contactID, title, body string) error {
	return nil
}

func (e *NoopEngine) DeleteDocument(id string) error {
	return nil
}

func (e *NoopEngine) Search(vaultID, query string, limit, offset int) (*SearchResponse, error) {
	return &SearchResponse{
		Contacts: []SearchResult{},
		Notes:    []SearchResult{},
		Total:    0,
		TookMs:   0,
	}, nil
}

func (e *NoopEngine) Close() error {
	return nil
}
