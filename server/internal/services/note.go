package services

import (
	"errors"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var ErrNoteNotFound = errors.New("note not found")

type NoteService struct {
	db            *gorm.DB
	feedRecorder  *FeedRecorder
	searchService *SearchService
}

func NewNoteService(db *gorm.DB) *NoteService {
	return &NoteService{db: db}
}

func (s *NoteService) SetFeedRecorder(fr *FeedRecorder) {
	s.feedRecorder = fr
}

func (s *NoteService) SetSearchService(ss *SearchService) {
	s.searchService = ss
}

func (s *NoteService) List(contactID, vaultID string) ([]dto.NoteResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
	var notes []models.Note
	if err := s.db.Where("contact_id = ?", contactID).Order("created_at DESC").Find(&notes).Error; err != nil {
		return nil, err
	}
	result := make([]dto.NoteResponse, len(notes))
	for i, n := range notes {
		result[i] = toNoteResponse(&n)
	}
	return result, nil
}

func (s *NoteService) Create(contactID, vaultID, authorID string, req dto.CreateNoteRequest) (*dto.NoteResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
	note := models.Note{
		ContactID: contactID,
		VaultID:   vaultID,
		AuthorID:  strPtrOrNil(authorID),
		Title:     strPtrOrNil(req.Title),
		Body:      req.Body,
	}
	if err := s.db.Create(&note).Error; err != nil {
		return nil, err
	}

	if s.feedRecorder != nil {
		entityType := "Note"
		s.feedRecorder.Record(contactID, authorID, ActionNoteCreated, "Created a note", &note.ID, &entityType)
	}

	if s.searchService != nil {
		s.searchService.IndexNote(&note)
	}

	resp := toNoteResponse(&note)
	return &resp, nil
}

func (s *NoteService) Update(id uint, contactID, vaultID string, req dto.UpdateNoteRequest) (*dto.NoteResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
	var note models.Note
	if err := s.db.Where("id = ? AND contact_id = ?", id, contactID).First(&note).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNoteNotFound
		}
		return nil, err
	}
	note.Title = strPtrOrNil(req.Title)
	note.Body = req.Body
	if err := s.db.Save(&note).Error; err != nil {
		return nil, err
	}

	if s.feedRecorder != nil {
		entityType := "Note"
		s.feedRecorder.Record(contactID, "", ActionNoteUpdated, "Updated a note", &note.ID, &entityType)
	}

	if s.searchService != nil {
		s.searchService.IndexNote(&note)
	}

	resp := toNoteResponse(&note)
	return &resp, nil
}

func (s *NoteService) Delete(id uint, contactID, vaultID string) error {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return err
	}
	result := s.db.Where("id = ? AND contact_id = ?", id, contactID).Delete(&models.Note{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNoteNotFound
	}

	if s.feedRecorder != nil {
		entityType := "Note"
		s.feedRecorder.Record(contactID, "", ActionNoteDeleted, "Deleted a note", &id, &entityType)
	}

	if s.searchService != nil {
		s.searchService.DeleteNote(id)
	}

	return nil
}

func toNoteResponse(n *models.Note) dto.NoteResponse {
	return dto.NoteResponse{
		ID:        n.ID,
		ContactID: n.ContactID,
		VaultID:   n.VaultID,
		AuthorID:  ptrToStr(n.AuthorID),
		Title:     ptrToStr(n.Title),
		Body:      n.Body,
		CreatedAt: n.CreatedAt,
		UpdatedAt: n.UpdatedAt,
	}
}
