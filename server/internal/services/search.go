package services

import (
	"fmt"

	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/search"
	"gorm.io/gorm"
)

type SearchService struct {
	db     *gorm.DB
	engine search.Engine
}

func NewSearchService(engine search.Engine) *SearchService {
	return &SearchService{engine: engine}
}

func NewSearchServiceWithDB(db *gorm.DB, engine search.Engine) *SearchService {
	return &SearchService{db: db, engine: engine}
}

func (s *SearchService) Search(vaultID, query string, page, perPage int) (*search.SearchResponse, error) {
	page, perPage = normalizeSearchPagination(page, perPage)
	offset := (page - 1) * perPage
	return s.engine.Search(vaultID, query, perPage, offset)
}

func (s *SearchService) SearchForUser(vaultID, userID, query string, page, perPage int) (*search.SearchResponse, error) {
	if userID == "" {
		return nil, ErrUserNotFound
	}
	if s.db == nil {
		return nil, fmt.Errorf("search service requires database for user-scoped contact name formatting")
	}
	page, perPage = normalizeSearchPagination(page, perPage)
	offset := (page - 1) * perPage
	resp, err := s.engine.Search(vaultID, query, perPage, offset)
	if err != nil || resp == nil || len(resp.Contacts) == 0 {
		return resp, err
	}
	return s.hydrateContactResultNames(resp, vaultID, userID)
}

func normalizeSearchPagination(page, perPage int) (int, int) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	return page, perPage
}

func (s *SearchService) hydrateContactResultNames(resp *search.SearchResponse, vaultID, userID string) (*search.SearchResponse, error) {
	contactIDs := make([]string, 0, len(resp.Contacts))
	for _, result := range resp.Contacts {
		contactIDs = append(contactIDs, result.ID)
	}
	var contacts []models.Contact
	if err := s.db.Where("id IN ? AND vault_id = ? AND listed = ?", contactIDs, vaultID, true).Find(&contacts).Error; err != nil {
		return nil, err
	}
	formatter, err := newContactNameFormatter(s.db, userID)
	if err != nil {
		return nil, err
	}
	nameByID := make(map[string]string, len(contacts))
	for i := range contacts {
		name, err := formatter.format(&contacts[i], "")
		if err != nil {
			return nil, err
		}
		nameByID[contacts[i].ID] = name
	}
	for i := range resp.Contacts {
		if name, ok := nameByID[resp.Contacts[i].ID]; ok {
			resp.Contacts[i].Name = name
		}
	}
	filteredContacts := resp.Contacts[:0]
	removedContactCount := 0
	for _, result := range resp.Contacts {
		if _, ok := nameByID[result.ID]; ok {
			filteredContacts = append(filteredContacts, result)
		} else {
			removedContactCount++
		}
	}
	resp.Contacts = filteredContacts
	if removedContactCount > 0 {
		resp.Total -= removedContactCount
		if resp.Total < 0 {
			resp.Total = 0
		}
	}
	return resp, nil
}

func (s *SearchService) IndexContact(contact *models.Contact) error {
	firstName := ptrToStr(contact.FirstName)
	lastName := ptrToStr(contact.LastName)
	nickname := ptrToStr(contact.Nickname)
	jobPosition := ptrToStr(contact.JobPosition)
	return s.engine.IndexContact(contact.ID, contact.VaultID, firstName, lastName, nickname, jobPosition)
}

func (s *SearchService) IndexNote(note *models.Note) error {
	title := ptrToStr(note.Title)
	return s.engine.IndexNote(fmt.Sprintf("%d", note.ID), note.VaultID, note.ContactID, title, note.Body)
}

func (s *SearchService) DeleteContact(id string) error {
	return s.engine.DeleteDocument("contact:" + id)
}

func (s *SearchService) DeleteNote(id uint) error {
	return s.engine.DeleteDocument(fmt.Sprintf("note:%d", id))
}

// RebuildIndex clears the search index and re-indexes all contacts and notes.
func (s *SearchService) RebuildIndex(db *gorm.DB) (int, int, error) {
	if err := s.engine.Rebuild(); err != nil {
		return 0, 0, fmt.Errorf("failed to rebuild index: %w", err)
	}

	var contacts []models.Contact
	if err := db.Find(&contacts).Error; err != nil {
		return 0, 0, fmt.Errorf("failed to load contacts: %w", err)
	}
	contactCount := 0
	for i := range contacts {
		if err := s.IndexContact(&contacts[i]); err != nil {
			return contactCount, 0, fmt.Errorf("failed to index contact %s: %w", contacts[i].ID, err)
		}
		contactCount++
	}

	var notes []models.Note
	if err := db.Find(&notes).Error; err != nil {
		return contactCount, 0, fmt.Errorf("failed to load notes: %w", err)
	}
	noteCount := 0
	for i := range notes {
		if err := s.IndexNote(&notes[i]); err != nil {
			return contactCount, noteCount, fmt.Errorf("failed to index note %d: %w", notes[i].ID, err)
		}
		noteCount++
	}

	return contactCount, noteCount, nil
}
