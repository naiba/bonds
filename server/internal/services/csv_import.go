package services

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strings"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// birthdayLayouts are tried in order when parsing a birthday column.
var birthdayLayouts = []string{
	"2006-01-02",
	"02/01/2006",
	"01/02/2006",
	"2/1/2006",
	"1/2/2006",
	"02-01-2006",
	"01-02-2006",
	"2 January 2006",
	"January 2, 2006",
	"2 Jan 2006",
	"Jan 2, 2006",
}

// MaxCSVFileSize is the maximum accepted file size for CSV uploads (10 MB).
const MaxCSVFileSize = 10 * 1024 * 1024

type CSVImportService struct {
	db             *gorm.DB
	feedRecorder   *FeedRecorder
	searchService  *SearchService
	davPushService *DavPushService
}

func NewCSVImportService(db *gorm.DB) *CSVImportService {
	return &CSVImportService{db: db}
}

func (s *CSVImportService) SetFeedRecorder(fr *FeedRecorder) {
	s.feedRecorder = fr
}

func (s *CSVImportService) SetSearchService(ss *SearchService) {
	s.searchService = ss
}

func (s *CSVImportService) SetDavPushService(ps *DavPushService) {
	s.davPushService = ps
}

func (s *CSVImportService) Import(vaultID, userID string, data []byte, mapping dto.CSVColumnMapping) (*dto.CSVImportResponse, error) {
	var vault models.Vault
	if err := s.db.First(&vault, "id = ?", vaultID).Error; err != nil {
		return nil, fmt.Errorf("vault not found: %w", err)
	}
	accountID := vault.AccountID

	data = bytes.TrimPrefix(data, []byte("\xEF\xBB\xBF"))
	r := csv.NewReader(strings.NewReader(string(data)))
	r.LazyQuotes = true
	r.TrimLeadingSpace = true

	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}
	if len(records) < 2 {
		return &dto.CSVImportResponse{Errors: []string{}}, nil
	}

	headers := records[0]
	colIndex := buildColIndex(headers)

	resp := &dto.CSVImportResponse{Errors: []string{}}

	for rowNum, row := range records[1:] {
		if err := s.importRow(row, rowNum+2, colIndex, mapping, vaultID, accountID, userID, resp); err != nil {
			resp.Errors = append(resp.Errors, fmt.Sprintf("row %d: %v", rowNum+2, err))
			resp.SkippedCount++
		}
	}
	return resp, nil
}

// buildColIndex maps lowercased header name to its column index.
func buildColIndex(headers []string) map[string]int {
	m := make(map[string]int, len(headers))
	for i, h := range headers {
		m[strings.ToLower(strings.TrimSpace(h))] = i
	}
	return m
}

// col returns the trimmed value for a mapped field, or "" if not mapped / out of range.
func col(row []string, colIndex map[string]int, header string) string {
	if header == "" {
		return ""
	}
	idx, ok := colIndex[strings.ToLower(strings.TrimSpace(header))]
	if !ok || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

func (s *CSVImportService) importRow(
	row []string, rowNum int,
	colIndex map[string]int,
	m dto.CSVColumnMapping,
	vaultID, accountID, userID string,
	resp *dto.CSVImportResponse,
) error {
	firstName := col(row, colIndex, m.FirstName)
	if firstName == "" {
		resp.SkippedCount++
		return nil
	}

	now := time.Now()

	// Gender lookup: name match first, then translation key fallback.
	var genderID *uint
	if genderVal := col(row, colIndex, m.Gender); genderVal != "" {
		var gender models.Gender
		if s.db.Where("account_id = ? AND name = ?", accountID, genderVal).First(&gender).Error == nil {
			genderID = &gender.ID
		} else {
			key := genderNameToTranslationKey(genderVal)
			if key != "" {
				if s.db.Where("account_id = ? AND name_translation_key = ?", accountID, key).First(&gender).Error == nil {
					genderID = &gender.ID
				}
			}
		}
	}

	contact := models.Contact{
		VaultID:       vaultID,
		FirstName:     strPtrOrNil(firstName),
		LastName:      strPtrOrNil(col(row, colIndex, m.LastName)),
		MiddleName:    strPtrOrNil(col(row, colIndex, m.MiddleName)),
		Nickname:      strPtrOrNil(col(row, colIndex, m.Nickname)),
		Prefix:        strPtrOrNil(col(row, colIndex, m.Prefix)),
		Suffix:        strPtrOrNil(col(row, colIndex, m.Suffix)),
		JobPosition:   strPtrOrNil(col(row, colIndex, m.JobTitle)),
		GenderID:      genderID,
		Listed:        true,
		LastUpdatedAt: &now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	// Contact + ContactVaultUser in one atomic transaction.
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&contact).Error; err != nil {
			return fmt.Errorf("failed to create contact: %w", err)
		}
		cvu := models.ContactVaultUser{
			ContactID: contact.ID,
			VaultID:   vaultID,
			UserID:    userID,
		}
		if err := tx.Create(&cvu).Error; err != nil {
			return fmt.Errorf("vault user link: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	resp.ImportedContacts++

	// Email.
	if emailVal := col(row, colIndex, m.Email); emailVal != "" {
		s.createContactInfo(accountID, contact.ID, emailVal, "seed.contact_info_types.email_address", resp, rowNum)
	}

	// Phone.
	if phoneVal := col(row, colIndex, m.Phone); phoneVal != "" {
		s.createContactInfo(accountID, contact.ID, phoneVal, "seed.contact_info_types.phone", resp, rowNum)
	}

	// Birthday.
	if bdVal := col(row, colIndex, m.Birthday); bdVal != "" {
		s.createBirthday(vaultID, contact.ID, bdVal, resp, rowNum)
	}

	// Tags (comma-separated, already unquoted by encoding/csv).
	if tagsVal := col(row, colIndex, m.Tags); tagsVal != "" {
		for _, tag := range splitCSVList(tagsVal) {
			label, err := s.findOrCreateLabel(s.db, vaultID, tag)
			if err != nil {
				resp.Errors = append(resp.Errors, fmt.Sprintf("row %d: tag %q: %v", rowNum, tag, err))
				continue
			}
			if err := s.db.Create(&models.ContactLabel{ContactID: contact.ID, LabelID: label.ID}).Error; err != nil {
				resp.Errors = append(resp.Errors, fmt.Sprintf("row %d: tag %q link: %v", rowNum, tag, err))
			}
		}
	}

	// Groups (comma-separated).
	if groupsVal := col(row, colIndex, m.Groups); groupsVal != "" {
		for _, groupName := range splitCSVList(groupsVal) {
			var group models.Group
			if s.db.Where("vault_id = ? AND name = ?", vaultID, groupName).First(&group).Error == nil {
				if err := s.db.Create(&models.ContactGroup{GroupID: group.ID, ContactID: contact.ID}).Error; err != nil {
					resp.Errors = append(resp.Errors, fmt.Sprintf("row %d: group %q link: %v", rowNum, groupName, err))
				}
			} else {
				resp.Errors = append(resp.Errors, fmt.Sprintf("row %d: group %q not found (create it first)", rowNum, groupName))
			}
		}
	}

	// Notes — also prepend company name if provided (company is not a plain
	// string field on Contact; full company linking is out of scope for CSV import).
	var createdNote *models.Note
	notesVal := col(row, colIndex, m.Notes)
	if companyVal := col(row, colIndex, m.Company); companyVal != "" {
		if notesVal != "" {
			notesVal = "Company: " + companyVal + "\n" + notesVal
		} else {
			notesVal = "Company: " + companyVal
		}
	}
	if notesVal != "" {
		note := models.Note{
			ContactID: contact.ID,
			VaultID:   vaultID,
			AuthorID:  &userID,
			Body:      notesVal,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := s.db.Create(&note).Error; err != nil {
			resp.Errors = append(resp.Errors, fmt.Sprintf("row %d: note: %v", rowNum, err))
		} else {
			createdNote = &note
		}
	}

	// Address (only created if at least one field is non-empty).
	s.createAddress(accountID, vaultID, contact.ID, row, colIndex, m, resp, rowNum)

	// Side effects: feed, search index, DAV push — mirror ContactService.CreateContact.
	if s.feedRecorder != nil {
		s.feedRecorder.Record(contact.ID, userID, ActionContactCreated, "Imported contact "+firstName, nil, nil)
	}
	if s.searchService != nil {
		s.searchService.IndexContact(&contact)
	}
	if s.davPushService != nil {
		go s.davPushService.PushContactChange(contact.ID, vaultID)
	}

	// Note side effects: mirror NoteService.Create.
	if createdNote != nil {
		if s.feedRecorder != nil {
			entityType := "Note"
			s.feedRecorder.Record(contact.ID, userID, ActionNoteCreated, "Created a note", &createdNote.ID, &entityType)
		}
		if s.searchService != nil {
			s.searchService.IndexNote(createdNote)
		}
	}

	return nil
}

func (s *CSVImportService) createContactInfo(accountID, contactID, value, translationKey string, resp *dto.CSVImportResponse, rowNum int) {
	var ciType models.ContactInformationType
	if err := s.db.Where("account_id = ? AND name_translation_key = ?", accountID, translationKey).First(&ciType).Error; err != nil {
		resp.Errors = append(resp.Errors, fmt.Sprintf("row %d: contact info type %q not found", rowNum, translationKey))
		return
	}
	ci := models.ContactInformation{
		ContactID: contactID,
		TypeID:    ciType.ID,
		Data:      value,
	}
	if err := s.db.Create(&ci).Error; err != nil {
		resp.Errors = append(resp.Errors, fmt.Sprintf("row %d: contact info: %v", rowNum, err))
	}
}

func (s *CSVImportService) createBirthday(vaultID, contactID, value string, resp *dto.CSVImportResponse, rowNum int) {
	var dateType models.ContactImportantDateType
	if err := s.db.Where("vault_id = ? AND internal_type = ?", vaultID, "birthdate").First(&dateType).Error; err != nil {
		return
	}
	t, ok := parseBirthday(value)
	if !ok {
		resp.Errors = append(resp.Errors, fmt.Sprintf("row %d: unrecognised birthday format: %q", rowNum, value))
		return
	}
	day, month, year := t.Day(), int(t.Month()), t.Year()
	date := models.ContactImportantDate{
		ContactID:                  contactID,
		ContactImportantDateTypeID: &dateType.ID,
		Label:                      dateType.Label,
		Day:                        &day,
		Month:                      &month,
		Year:                       &year,
		CalendarType:               "gregorian",
	}
	if err := s.db.Create(&date).Error; err != nil {
		resp.Errors = append(resp.Errors, fmt.Sprintf("row %d: birthday: %v", rowNum, err))
	}
}

func (s *CSVImportService) createAddress(accountID, vaultID, contactID string, row []string, colIndex map[string]int, m dto.CSVColumnMapping, resp *dto.CSVImportResponse, rowNum int) {
	street := col(row, colIndex, m.AddressStreet)
	city := col(row, colIndex, m.AddressCity)
	state := col(row, colIndex, m.AddressState)
	postal := col(row, colIndex, m.AddressPostalCode)
	country := col(row, colIndex, m.AddressCountry)

	if street == "" && city == "" && state == "" && postal == "" && country == "" {
		return
	}

	// Use "home" address type; fall back to any available type.
	var addrType models.AddressType
	if s.db.Where("account_id = ? AND name_translation_key = ?", accountID, "seed.address_types.home").First(&addrType).Error != nil {
		if s.db.Where("account_id = ?", accountID).First(&addrType).Error != nil {
			resp.Errors = append(resp.Errors, fmt.Sprintf("row %d: no address type found", rowNum))
			return
		}
	}

	addr := models.Address{
		VaultID:       vaultID,
		AddressTypeID: &addrType.ID,
		Line1:         strPtrOrNil(street),
		City:          strPtrOrNil(city),
		Province:      strPtrOrNil(state),
		PostalCode:    strPtrOrNil(postal),
		Country:       strPtrOrNil(country),
	}
	if err := s.db.Create(&addr).Error; err != nil {
		resp.Errors = append(resp.Errors, fmt.Sprintf("row %d: address: %v", rowNum, err))
		return
	}

	if err := s.db.Create(&models.ContactAddress{ContactID: contactID, AddressID: addr.ID}).Error; err != nil {
		resp.Errors = append(resp.Errors, fmt.Sprintf("row %d: address link: %v", rowNum, err))
	}
}

func (s *CSVImportService) findOrCreateLabel(tx *gorm.DB, vaultID, name string) (*models.Label, error) {
	slug := slugify(name)
	var label models.Label
	if err := tx.Where("vault_id = ? AND slug = ?", vaultID, slug).First(&label).Error; err == nil {
		return &label, nil
	}
	// Race-safe insert: another concurrent import could create the same
	// (vault_id, slug) between our SELECT above and CREATE below. The
	// uniqueIndex on Label (vault_id, slug) makes the conflict deterministic;
	// DoNothing lets us re-SELECT the winning row instead of bubbling up an
	// error. Without this, duplicate labels accumulate under concurrent imports.
	label = models.Label{VaultID: vaultID, Name: name, Slug: slug}
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&label).Error; err != nil {
		return nil, err
	}
	if label.ID == 0 {
		if err := tx.Where("vault_id = ? AND slug = ?", vaultID, slug).First(&label).Error; err != nil {
			return nil, err
		}
	}
	return &label, nil
}

// genderNameToTranslationKey maps common English gender names to seed keys.
func genderNameToTranslationKey(name string) string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "male", "m", "man":
		return "seed.genders.male"
	case "female", "f", "woman":
		return "seed.genders.female"
	case "other", "o", "non-binary", "nonbinary":
		return "seed.genders.other"
	}
	return ""
}

func parseBirthday(s string) (time.Time, bool) {
	for _, layout := range birthdayLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// splitCSVList splits a comma-separated list and trims each item.
func splitCSVList(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}
