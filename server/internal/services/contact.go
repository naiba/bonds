package services

import (
	"errors"
	"fmt"
	"math"
	"sort"
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
	db             *gorm.DB
	feedRecorder   *FeedRecorder
	searchService  *SearchService
	davPushService *DavPushService
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

func (s *ContactService) SetDavPushService(ps *DavPushService) {
	s.davPushService = ps
}

func (s *ContactService) ListContacts(vaultID, userID string, page, perPage int, search, sort, filter string) ([]dto.ContactResponse, response.Meta, error) {
	// Exclude UserVault shadow contacts (can_be_deleted=false AND listed=false)
	query := s.db.Where("vault_id = ? AND NOT (can_be_deleted = ? AND listed = ?)", vaultID, false, false)
	switch filter {
	case "archived":
		query = query.Where("listed = ?", false)
	case "all":
		// no additional filter
	case "favorites":
		query = query.Where("listed = ?", true)
		query = query.Where("id IN (SELECT contact_id FROM contact_vault_user WHERE user_id = ? AND is_favorite = ?)", userID, true)
	case "needs_verification":
		// Spans active and archived: a contact flagged for verification should
		// remain discoverable even if the user archived it before reviewing.
		query = query.Where("needs_verification = ?", true)
	default: // "active" or empty
		query = query.Where("listed = ?", true)
	}
	if search != "" {
		like := "%" + strings.ToLower(search) + "%"
		query = query.Where(
			s.db.Where("LOWER(first_name) LIKE ?", like).
				Or("LOWER(last_name) LIKE ?", like).
				Or("LOWER(nickname) LIKE ?", like),
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
	orderClause := contactSortOrder(sort)
	finalOrder := favoriteOrderClause(userID) + ", " + orderClause
	var contacts []models.Contact
	if err := query.Offset(offset).Limit(perPage).Order(finalOrder).Find(&contacts).Error; err != nil {
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

	birthdayMap, groupMap := s.fetchBirthdayAndGroupMaps(contactIDs)
	result := make([]dto.ContactResponse, len(contacts))
	for i, c := range contacts {
		result[i] = toContactResponse(&c, favoriteMap[c.ID])
	}

	enrichContactsWithBirthdayAndGroups(result, contacts, birthdayMap, groupMap)
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
		VaultID:                  vaultID,
		FirstName:                &req.FirstName,
		LastName:                 strPtrOrNil(req.LastName),
		MiddleName:               strPtrOrNil(req.MiddleName),
		Nickname:                 strPtrOrNil(req.Nickname),
		MaidenName:               strPtrOrNil(req.MaidenName),
		Prefix:                   strPtrOrNil(req.Prefix),
		Suffix:                   strPtrOrNil(req.Suffix),
		GenderID:                 req.GenderID,
		PronounID:                req.PronounID,
		TemplateID:               req.TemplateID,
		LastTalkedTo:             req.LastTalkedTo,
		StayInTouchFrequencyDays: req.StayInTouchFrequencyDays,
		StayInTouchTriggerDate:   calculateStayInTouchTriggerDate(req.LastTalkedTo, req.StayInTouchFrequencyDays),
		LastUpdatedAt:            &now,
	}
	if req.NeedsVerification != nil {
		contact.NeedsVerification = *req.NeedsVerification
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

	if s.davPushService != nil {
		go s.davPushService.PushContactChange(contact.ID, vaultID)
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
	contact.LastTalkedTo = req.LastTalkedTo
	contact.StayInTouchFrequencyDays = req.StayInTouchFrequencyDays
	contact.StayInTouchTriggerDate = calculateStayInTouchTriggerDate(req.LastTalkedTo, req.StayInTouchFrequencyDays)
	contact.LastUpdatedAt = &now
	if req.NeedsVerification != nil {
		contact.NeedsVerification = *req.NeedsVerification
	}

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

	if s.davPushService != nil {
		go s.davPushService.PushContactChange(contactID, vaultID)
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

	if s.davPushService != nil {
		go s.davPushService.PushContactDelete(contactID, vaultID)
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

func (s *ContactService) ListCatchUpPrompts(vaultID string) ([]dto.CatchUpPromptResponse, error) {
	now := time.Now()
	var contacts []models.Contact
	if err := s.db.Where("vault_id = ?", vaultID).
		Where("listed = ?", true).
		Where("NOT (can_be_deleted = ? AND listed = ?)", false, false).
		Where("last_talked_to IS NOT NULL").
		Where("stay_in_touch_frequency_days IS NOT NULL AND stay_in_touch_frequency_days > ?", 0).
		Find(&contacts).Error; err != nil {
		return nil, err
	}

	prompts := make([]dto.CatchUpPromptResponse, 0, len(contacts))
	for _, contact := range contacts {
		triggerDate := resolveStayInTouchTriggerDate(&contact)
		if triggerDate == nil || triggerDate.After(now) {
			continue
		}
		daysSinceLastContact := daysBetween(*contact.LastTalkedTo, now)
		daysOverdue := daysBetween(*triggerDate, now)
		frequencyDays := *contact.StayInTouchFrequencyDays
		prompts = append(prompts, dto.CatchUpPromptResponse{
			ContactID:                contact.ID,
			FirstName:                ptrToStr(contact.FirstName),
			LastName:                 ptrToStr(contact.LastName),
			LastTalkedTo:             *contact.LastTalkedTo,
			StayInTouchFrequencyDays: frequencyDays,
			StayInTouchTriggerDate:   *triggerDate,
			DaysSinceLastContact:     daysSinceLastContact,
			DaysOverdue:              daysOverdue,
			PriorityScore:            float64(daysOverdue) / float64(frequencyDays),
		})
	}

	sort.SliceStable(prompts, func(i, j int) bool {
		return prompts[i].PriorityScore > prompts[j].PriorityScore
	})
	return prompts, nil
}

func (s *ContactService) MarkCaughtUp(contactID, vaultID string) (*dto.ContactResponse, error) {
	var contact models.Contact
	if err := s.db.Where("id = ? AND vault_id = ?", contactID, vaultID).First(&contact).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrContactNotFound
		}
		return nil, err
	}

	now := time.Now()
	contact.LastTalkedTo = &now
	contact.StayInTouchTriggerDate = calculateStayInTouchTriggerDate(contact.LastTalkedTo, contact.StayInTouchFrequencyDays)
	contact.LastUpdatedAt = &now
	if err := s.db.Save(&contact).Error; err != nil {
		return nil, err
	}

	resp := toContactResponse(&contact, false)
	return &resp, nil
}

func (s *ContactService) ListContactsByLabel(vaultID, userID string, labelID uint, page, perPage int, sort, filter string) ([]dto.ContactResponse, response.Meta, error) {
	query := s.db.Where("vault_id = ? AND id IN (SELECT contact_id FROM contact_label WHERE label_id = ?)", vaultID, labelID)
	// Apply filter
	switch filter {
	case "archived":
		query = query.Where("listed = ?", false)
	case "all":
		// no filter
	case "favorites":
		query = query.Where("listed = ?", true)
		query = query.Where("id IN (SELECT contact_id FROM contact_vault_user WHERE user_id = ? AND is_favorite = ?)", userID, true)
	case "needs_verification":
		// Spans active and archived: a contact flagged for verification should
		// remain discoverable even if the user archived it before reviewing.
		query = query.Where("needs_verification = ?", true)
	default: // "active" or empty
		query = query.Where("listed = ?", true)
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
	orderClause := contactSortOrder(sort)
	finalOrder := favoriteOrderClause(userID) + ", " + orderClause
	var contacts []models.Contact
	if err := query.Offset(offset).Limit(perPage).Order(finalOrder).Find(&contacts).Error; err != nil {
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

	birthdayMap, groupMap := s.fetchBirthdayAndGroupMaps(contactIDs)
	result := make([]dto.ContactResponse, len(contacts))
	for i, c := range contacts {
		result[i] = toContactResponse(&c, favoriteMap[c.ID])
	}

	enrichContactsWithBirthdayAndGroups(result, contacts, birthdayMap, groupMap)
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

	likeTerm := "%" + strings.ToLower(term) + "%"
	var contacts []models.Contact
	if err := s.db.Where("vault_id = ?", vaultID).
		Where(
			s.db.Where("LOWER(first_name) LIKE ?", likeTerm).
				Or("LOWER(last_name) LIKE ?", likeTerm).
				Or("LOWER(nickname) LIKE ?", likeTerm).
				Or("LOWER(maiden_name) LIKE ?", likeTerm).
				Or("LOWER(middle_name) LIKE ?", likeTerm),
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

func favoriteOrderClause(userID string) string {
	return "CASE WHEN id IN (SELECT contact_id FROM contact_vault_user WHERE user_id = '" + userID + "' AND is_favorite = true) THEN 0 ELSE 1 END"
}

func contactSortOrder(sort string) string {
	switch sort {
	case "first_name":
		return "first_name ASC, last_name ASC"
	case "last_name":
		return "last_name ASC, first_name ASC"
	case "created_at":
		return "created_at DESC"
	default:
		return "updated_at DESC"
	}
}

func toContactResponse(c *models.Contact, isFavorite bool) dto.ContactResponse {
	return dto.ContactResponse{
		ID:                       c.ID,
		VaultID:                  c.VaultID,
		FirstName:                ptrToStr(c.FirstName),
		LastName:                 ptrToStr(c.LastName),
		MiddleName:               ptrToStr(c.MiddleName),
		Nickname:                 ptrToStr(c.Nickname),
		MaidenName:               ptrToStr(c.MaidenName),
		Prefix:                   ptrToStr(c.Prefix),
		Suffix:                   ptrToStr(c.Suffix),
		GenderID:                 c.GenderID,
		PronounID:                c.PronounID,
		TemplateID:               c.TemplateID,
		CompanyID:                c.CompanyID,
		ReligionID:               c.ReligionID,
		FileID:                   c.FileID,
		JobPosition:              ptrToStr(c.JobPosition),
		LastTalkedTo:             c.LastTalkedTo,
		StayInTouchFrequencyDays: c.StayInTouchFrequencyDays,
		StayInTouchTriggerDate:   c.StayInTouchTriggerDate,
		Listed:                   c.Listed,
		ShowQuickFacts:           c.ShowQuickFacts,
		IsArchived:               !c.Listed,
		IsFavorite:               isFavorite,
		NeedsVerification:        c.NeedsVerification,
		CreatedAt:                c.CreatedAt,
		UpdatedAt:                c.UpdatedAt,
	}
}

func calculateStayInTouchTriggerDate(lastTalkedTo *time.Time, frequencyDays *int) *time.Time {
	if lastTalkedTo == nil || frequencyDays == nil {
		return nil
	}
	triggerDate := lastTalkedTo.AddDate(0, 0, *frequencyDays)
	return &triggerDate
}

func resolveStayInTouchTriggerDate(contact *models.Contact) *time.Time {
	if contact.StayInTouchTriggerDate != nil {
		return contact.StayInTouchTriggerDate
	}
	return calculateStayInTouchTriggerDate(contact.LastTalkedTo, contact.StayInTouchFrequencyDays)
}

func daysBetween(from, to time.Time) int {
	if to.Before(from) {
		return 0
	}
	return int(to.Sub(from).Hours() / 24)
}

func (s *ContactService) fetchBirthdayAndGroupMaps(contactIDs []string) (map[string]*models.ContactImportantDate, map[string][]dto.ContactGroupBrief) {
	birthdayMap := make(map[string]*models.ContactImportantDate)
	if len(contactIDs) > 0 {
		var dates []models.ContactImportantDate
		s.db.Joins("JOIN contact_important_date_types ON contact_important_date_types.id = contact_important_dates.contact_important_date_type_id").
			Where("contact_important_dates.contact_id IN ? AND contact_important_date_types.internal_type = ?", contactIDs, "birthdate").
			Where("contact_important_dates.deleted_at IS NULL").
			Find(&dates)
		for i := range dates {
			birthdayMap[dates[i].ContactID] = &dates[i]
		}
	}

	groupMap := make(map[string][]dto.ContactGroupBrief)
	if len(contactIDs) > 0 {
		type groupRow struct {
			ContactID string
			GroupID   uint
			GroupName string
		}
		var rows []groupRow
		s.db.Table("contact_group").
			Select("contact_group.contact_id, groups.id as group_id, groups.name as group_name").
			Joins("JOIN groups ON groups.id = contact_group.group_id").
			Where("contact_group.contact_id IN ?", contactIDs).
			Where("groups.deleted_at IS NULL").
			Scan(&rows)
		for _, r := range rows {
			groupMap[r.ContactID] = append(groupMap[r.ContactID], dto.ContactGroupBrief{ID: r.GroupID, Name: r.GroupName})
		}
	}

	return birthdayMap, groupMap
}

func enrichContactsWithBirthdayAndGroups(result []dto.ContactResponse, contacts []models.Contact, birthdayMap map[string]*models.ContactImportantDate, groupMap map[string][]dto.ContactGroupBrief) {
	for i, c := range contacts {
		if bd, ok := birthdayMap[c.ID]; ok {
			result[i].Birthday = formatBirthdayStr(bd)
			result[i].Age = calculateAgeFromDate(bd)
		}
		if groups, ok := groupMap[c.ID]; ok {
			result[i].Groups = groups
		}
	}
}

func formatBirthdayStr(d *models.ContactImportantDate) *string {
	if d.Month == nil || d.Day == nil {
		return nil
	}
	var s string
	if d.Year != nil {
		s = fmt.Sprintf("%04d-%02d-%02d", *d.Year, *d.Month, *d.Day)
	} else {
		s = fmt.Sprintf("--%02d-%02d", *d.Month, *d.Day)
	}
	return &s
}

func calculateAgeFromDate(d *models.ContactImportantDate) *int {
	if d.Year == nil || d.Month == nil || d.Day == nil {
		return nil
	}
	now := time.Now()
	age := now.Year() - *d.Year
	if int(now.Month()) < *d.Month || (int(now.Month()) == *d.Month && now.Day() < *d.Day) {
		age--
	}
	if age < 0 {
		return nil
	}
	return &age
}
