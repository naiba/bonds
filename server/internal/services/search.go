package services

import (
	"fmt"

	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/search"
	"gorm.io/gorm"
)

type SearchService struct {
	engine search.Engine
}

func NewSearchService(engine search.Engine) *SearchService {
	return &SearchService{engine: engine}
}

func (s *SearchService) Search(vaultID, query string, page, perPage int) (*search.SearchResponse, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	offset := (page - 1) * perPage
	return s.engine.Search(vaultID, query, perPage, offset)
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
