package services

import (
	"errors"
	"math"
	"strings"
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
		MiddleName:    strPtrOrNil(req.MiddleName),
		Nickname:      strPtrOrNil(req.Nickname),
		MaidenName:    strPtrOrNil(req.MaidenName),
		Prefix:        strPtrOrNil(req.Prefix),
		Suffix:        strPtrOrNil(req.Suffix),
		GenderID:      req.GenderID,
		PronounID:     req.PronounID,
		TemplateID:    req.TemplateID,
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

	// Handle GORM SQLite zero-value bool quirk: Listed defaults to true via gorm tag,
	// so we must explicitly update to false after Create if requested.
	if req.Listed != nil && !*req.Listed {
		s.db.Model(&contact).Update("listed", false)
		contact.Listed = false
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

func (s *ContactService) GetContact(contactID, userID, vaultID string) (*dto.ContactResponse, error) {
	var contact models.Contact
	if err := s.db.Where("id = ? AND vault_id = ?", contactID, vaultID).First(&contact).Error; err != nil {
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

func (s *ContactService) UpdateContact(contactID, vaultID string, req dto.UpdateContactRequest) (*dto.ContactResponse, error) {
	var contact models.Contact
	if err := s.db.Where("id = ? AND vault_id = ?", contactID, vaultID).First(&contact).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrContactNotFound
		}
		return nil, err
	}

	now := time.Now()
	contact.FirstName = &req.FirstName
	contact.LastName = strPtrOrNil(req.LastName)
	contact.MiddleName = strPtrOrNil(req.MiddleName)
	contact.Nickname = strPtrOrNil(req.Nickname)
	contact.MaidenName = strPtrOrNil(req.MaidenName)
	contact.Prefix = strPtrOrNil(req.Prefix)
	contact.Suffix = strPtrOrNil(req.Suffix)
	contact.GenderID = req.GenderID
	contact.PronounID = req.PronounID
	contact.TemplateID = req.TemplateID
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

func (s *ContactService) DeleteContact(contactID, vaultID string) error {
	var contact models.Contact
	if err := s.db.Where("id = ? AND vault_id = ?", contactID, vaultID).First(&contact).Error; err != nil {
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

func (s *ContactService) ToggleArchive(contactID, vaultID string) (*dto.ContactResponse, error) {
	var contact models.Contact
	if err := s.db.Where("id = ? AND vault_id = ?", contactID, vaultID).First(&contact).Error; err != nil {
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
	if err := s.db.Where("id = ? AND vault_id = ?", contactID, vaultID).First(&contact).Error; err != nil {
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

func (s *ContactService) ListContactsByLabel(vaultID, userID string, labelID uint, page, perPage int) ([]dto.ContactResponse, response.Meta, error) {
	query := s.db.Where("vault_id = ? AND id IN (SELECT contact_id FROM contact_label WHERE label_id = ?)", vaultID, labelID)

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

func (s *ContactService) QuickSearch(vaultID, term string) ([]dto.ContactSearchItem, error) {
	if term == "" {
		return []dto.ContactSearchItem{}, nil
	}

	likeTerm := "%" + term + "%"
	var contacts []models.Contact
	if err := s.db.Where("vault_id = ?", vaultID).
		Where(
			s.db.Where("first_name LIKE ?", likeTerm).
				Or("last_name LIKE ?", likeTerm).
				Or("nickname LIKE ?", likeTerm).
				Or("maiden_name LIKE ?", likeTerm).
				Or("middle_name LIKE ?", likeTerm),
		).
		Order("first_name ASC, last_name ASC").
		Limit(5).
		Find(&contacts).Error; err != nil {
		return nil, err
	}

	result := make([]dto.ContactSearchItem, len(contacts))
	for i, c := range contacts {
		result[i] = dto.ContactSearchItem{
			ID:   c.ID,
			Name: buildContactDisplayName(&c),
		}
	}
	return result, nil
}

func buildContactDisplayName(c *models.Contact) string {
	parts := make([]string, 0, 5)
	if c.FirstName != nil && *c.FirstName != "" {
		parts = append(parts, *c.FirstName)
	}
	if c.LastName != nil && *c.LastName != "" {
		parts = append(parts, *c.LastName)
	}
	if c.Nickname != nil && *c.Nickname != "" {
		parts = append(parts, *c.Nickname)
	}
	if c.MaidenName != nil && *c.MaidenName != "" {
		parts = append(parts, *c.MaidenName)
	}
	if c.MiddleName != nil && *c.MiddleName != "" {
		parts = append(parts, *c.MiddleName)
	}
	return strings.Join(parts, " ")
}

// validateContactBelongsToVault checks that a contact exists and belongs to the given vault.
// Returns ErrContactNotFound if the contact does not exist or belongs to a different vault.
func validateContactBelongsToVault(db *gorm.DB, contactID, vaultID string) error {
	var contact models.Contact
	if err := db.Where("id = ? AND vault_id = ?", contactID, vaultID).First(&contact).Error; err != nil {
		return ErrContactNotFound
	}
	return nil
}

func strPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func toContactResponse(c *models.Contact, isFavorite bool) dto.ContactResponse {
	return dto.ContactResponse{
		ID:             c.ID,
		VaultID:        c.VaultID,
		FirstName:      ptrToStr(c.FirstName),
		LastName:       ptrToStr(c.LastName),
		MiddleName:     ptrToStr(c.MiddleName),
		Nickname:       ptrToStr(c.Nickname),
		MaidenName:     ptrToStr(c.MaidenName),
		Prefix:         ptrToStr(c.Prefix),
		Suffix:         ptrToStr(c.Suffix),
		GenderID:       c.GenderID,
		PronounID:      c.PronounID,
		TemplateID:     c.TemplateID,
		CompanyID:      c.CompanyID,
		ReligionID:     c.ReligionID,
		FileID:         c.FileID,
		JobPosition:    ptrToStr(c.JobPosition),
		Listed:         c.Listed,
		ShowQuickFacts: c.ShowQuickFacts,
		IsArchived:     !c.Listed,
		IsFavorite:     isFavorite,
		CreatedAt:      c.CreatedAt,
		UpdatedAt:      c.UpdatedAt,
	}
}
