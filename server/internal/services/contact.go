package services

import (
	"errors"
	"math"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/pkg/response"
	"gorm.io/gorm"
)

var (
	ErrContactNotFound = errors.New("contact not found")
)

type ContactService struct {
	db            *gorm.DB
	feedRecorder  *FeedRecorder
	searchService *SearchService
}

func NewContactService(db *gorm.DB) *ContactService {
	return &ContactService{db: db}
}

func (s *ContactService) SetFeedRecorder(fr *FeedRecorder) {
	s.feedRecorder = fr
}

func (s *ContactService) SetSearchService(ss *SearchService) {
	s.searchService = ss
}

func (s *ContactService) ListContacts(vaultID, userID string, page, perPage int, search string) ([]dto.ContactResponse, response.Meta, error) {
	query := s.db.Where("vault_id = ?", vaultID)

	if search != "" {
		query = query.Where(
			s.db.Where("first_name LIKE ?", "%"+search+"%").
				Or("last_name LIKE ?", "%"+search+"%").
				Or("nickname LIKE ?", "%"+search+"%"),
		)
	}

	var total int64
	if err := query.Model(&models.Contact{}).Count(&total).Error; err != nil {
		return nil, response.Meta{}, err
	}

	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 15
	}
	offset := (page - 1) * perPage

	var contacts []models.Contact
	if err := query.Offset(offset).Limit(perPage).Order("created_at DESC").Find(&contacts).Error; err != nil {
		return nil, response.Meta{}, err
	}

	contactIDs := make([]string, len(contacts))
	for i, c := range contacts {
		contactIDs[i] = c.ID
	}

	favoriteMap := make(map[string]bool)
	if len(contactIDs) > 0 {
		var cvus []models.ContactVaultUser
		s.db.Where("contact_id IN ? AND user_id = ?", contactIDs, userID).Find(&cvus)
		for _, cvu := range cvus {
			favoriteMap[cvu.ContactID] = cvu.IsFavorite
		}
	}

	result := make([]dto.ContactResponse, len(contacts))
	for i, c := range contacts {
		result[i] = toContactResponse(&c, favoriteMap[c.ID])
	}

	meta := response.Meta{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: int(math.Ceil(float64(total) / float64(perPage))),
	}

	return result, meta, nil
}

func (s *ContactService) CreateContact(vaultID, userID string, req dto.CreateContactRequest) (*dto.ContactResponse, error) {
	now := time.Now()
	contact := models.Contact{
		VaultID:       vaultID,
		FirstName:     &req.FirstName,
		LastName:      strPtrOrNil(req.LastName),
		Nickname:      strPtrOrNil(req.Nickname),
		LastUpdatedAt: &now,
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&contact).Error; err != nil {
			return err
		}
		cvu := models.ContactVaultUser{
			ContactID: contact.ID,
			UserID:    userID,
			VaultID:   vaultID,
		}
		return tx.Create(&cvu).Error
	})
	if err != nil {
		return nil, err
	}

	if s.feedRecorder != nil {
		desc := "Created contact " + req.FirstName
		s.feedRecorder.Record(contact.ID, userID, ActionContactCreated, desc, nil, nil)
	}

	if s.searchService != nil {
		s.searchService.IndexContact(&contact)
	}

	resp := toContactResponse(&contact, false)
	return &resp, nil
}

func (s *ContactService) GetContact(contactID, userID string) (*dto.ContactResponse, error) {
	var contact models.Contact
	if err := s.db.First(&contact, "id = ?", contactID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrContactNotFound
		}
		return nil, err
	}

	var cvu models.ContactVaultUser
	isFav := false
	if err := s.db.Where("contact_id = ? AND user_id = ?", contactID, userID).First(&cvu).Error; err == nil {
		isFav = cvu.IsFavorite
		s.db.Model(&cvu).Update("number_of_views", cvu.NumberOfViews+1)
	}

	resp := toContactResponse(&contact, isFav)
	return &resp, nil
}

func (s *ContactService) UpdateContact(contactID string, req dto.UpdateContactRequest) (*dto.ContactResponse, error) {
	var contact models.Contact
	if err := s.db.First(&contact, "id = ?", contactID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrContactNotFound
		}
		return nil, err
	}

	now := time.Now()
	contact.FirstName = &req.FirstName
	contact.LastName = strPtrOrNil(req.LastName)
	contact.Nickname = strPtrOrNil(req.Nickname)
	contact.LastUpdatedAt = &now

	if err := s.db.Save(&contact).Error; err != nil {
		return nil, err
	}

	if s.feedRecorder != nil {
		desc := "Updated contact " + req.FirstName
		s.feedRecorder.Record(contact.ID, "", ActionContactUpdated, desc, nil, nil)
	}

	if s.searchService != nil {
		s.searchService.IndexContact(&contact)
	}

	resp := toContactResponse(&contact, false)
	return &resp, nil
}

func (s *ContactService) DeleteContact(contactID string) error {
	var contact models.Contact
	if err := s.db.First(&contact, "id = ?", contactID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrContactNotFound
		}
		return err
	}
	if err := s.db.Delete(&contact).Error; err != nil {
		return err
	}

	if s.feedRecorder != nil {
		s.feedRecorder.Record(contactID, "", ActionContactDeleted, "Deleted contact", nil, nil)
	}

	if s.searchService != nil {
		s.searchService.DeleteContact(contactID)
	}

	return nil
}

func (s *ContactService) ToggleArchive(contactID string) (*dto.ContactResponse, error) {
	var contact models.Contact
	if err := s.db.First(&contact, "id = ?", contactID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrContactNotFound
		}
		return nil, err
	}

	contact.Listed = !contact.Listed
	if err := s.db.Save(&contact).Error; err != nil {
		return nil, err
	}

	resp := toContactResponse(&contact, false)
	return &resp, nil
}

func (s *ContactService) ToggleFavorite(contactID, userID, vaultID string) (*dto.ContactResponse, error) {
	var contact models.Contact
	if err := s.db.First(&contact, "id = ?", contactID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrContactNotFound
		}
		return nil, err
	}

	var cvu models.ContactVaultUser
	err := s.db.Where("contact_id = ? AND user_id = ?", contactID, userID).First(&cvu).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		cvu = models.ContactVaultUser{
			ContactID:  contactID,
			UserID:     userID,
			VaultID:    vaultID,
			IsFavorite: true,
		}
		if err := s.db.Create(&cvu).Error; err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	} else {
		cvu.IsFavorite = !cvu.IsFavorite
		if err := s.db.Save(&cvu).Error; err != nil {
			return nil, err
		}
	}

	resp := toContactResponse(&contact, cvu.IsFavorite)
	return &resp, nil
}

func strPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func toContactResponse(c *models.Contact, isFavorite bool) dto.ContactResponse {
	return dto.ContactResponse{
		ID:         c.ID,
		VaultID:    c.VaultID,
		FirstName:  ptrToStr(c.FirstName),
		LastName:   ptrToStr(c.LastName),
		Nickname:   ptrToStr(c.Nickname),
		IsArchived: !c.Listed,
		IsFavorite: isFavorite,
		CreatedAt:  c.CreatedAt,
		UpdatedAt:  c.UpdatedAt,
	}
}
