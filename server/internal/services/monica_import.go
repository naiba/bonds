package services

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/search"
	"gorm.io/gorm"
)

const MonicaExportVersion = "1.0-preview.1"

var (
	ErrMonicaInvalidJSON    = errors.New("invalid Monica export JSON")
	ErrMonicaInvalidVersion = errors.New("unsupported Monica export version")
)

// ParseMonicaExport 解析 Monica 4.x JSON 导出数据
func ParseMonicaExport(data []byte) (*MonicaExport, error) {
	var export MonicaExport
	if err := json.Unmarshal(data, &export); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMonicaInvalidJSON, err)
	}
	if export.Version != MonicaExportVersion {
		return nil, fmt.Errorf("%w: %s (expected %s)", ErrMonicaInvalidVersion, export.Version, MonicaExportVersion)
	}
	return &export, nil
}

// getCollectionByType 从 []MonicaCollection 中按 type 获取 values。
// Monica 4.x 的 CountResourceCollection 通过 Str::of(class)->afterLast('\\')->kebab()
// 生成单数形式的 type 名 (e.g. "contact", "note")，而 Bonds 代码库历史上使用
// 复数形式 (e.g. "contacts", "notes")。为兼容真实 Monica 导出 (#83)，同时接受两种。
func getCollectionByType(collections []MonicaCollection, entityType string) []json.RawMessage {
	singular := strings.TrimSuffix(entityType, "s")
	if entityType == "addresses" {
		singular = "address"
	} else if entityType == "activities" {
		singular = "activity"
	}
	for _, c := range collections {
		if c.Type == entityType || c.Type == singular {
			return c.Values
		}
	}
	return nil
}

type MonicaImportService struct {
	DB           *gorm.DB
	UploadDir    string // Storage.UploadDir — 为空时跳过文件导入
	feedRecorder *FeedRecorder
	searchEngine search.Engine
}

func NewMonicaImportService(db *gorm.DB, uploadDir string) *MonicaImportService {
	return &MonicaImportService{DB: db, UploadDir: uploadDir}
}

func (s *MonicaImportService) SetFeedRecorder(fr *FeedRecorder) {
	s.feedRecorder = fr
}

func (s *MonicaImportService) SetSearchEngine(se search.Engine) {
	s.searchEngine = se
}

func (s *MonicaImportService) Import(vaultID, userID string, data []byte) (*dto.MonicaImportResponse, error) {
	export, err := ParseMonicaExport(data)
	if err != nil {
		return nil, err
	}

	resp := &dto.MonicaImportResponse{Errors: []string{}}

	var vault models.Vault
	if err := s.DB.First(&vault, "id = ?", vaultID).Error; err != nil {
		return nil, fmt.Errorf("vault not found: %w", err)
	}
	accountID := vault.AccountID

	genderByUUID := buildGenderMap(export.Account.Instance.Genders)
	fieldTypeByUUID := buildFieldTypeMap(export.Account.Instance.ContactFieldTypes)
	lifeEventTypeByUUID := buildLifeEventTypeMap(export.Account.Instance.LifeEventTypes)
	activityTypeByUUID := buildActivityTypeMap(export.Account.Instance.ActivityTypes)

	contactUUIDMap := make(map[string]string)

	contactRaws := getCollectionByType(export.Account.Data, "contacts")
	for _, raw := range contactRaws {
		var mc MonicaContact
		if err := json.Unmarshal(raw, &mc); err != nil {
			resp.Errors = append(resp.Errors, fmt.Sprintf("failed to parse contact: %v", err))
			resp.SkippedCount++
			continue
		}

		contactID, imported, err := s.importContact(s.DB, &mc, vaultID, accountID, userID, genderByUUID, resp)
		if err != nil {
			resp.Errors = append(resp.Errors, fmt.Sprintf("contact %s: %v", mc.UUID, err))
			resp.SkippedCount++
			continue
		}

		contactUUIDMap[mc.UUID] = contactID
		if imported {
			resp.ImportedContacts++
		}
	}

	// 获取影子联系人的 ID (用于 Loan 中的 loaner/loanee)
	userContactID := ""
	var uv models.UserVault
	if err := s.DB.Where("user_id = ? AND vault_id = ?", userID, vaultID).First(&uv).Error; err == nil {
		userContactID = uv.ContactID
	}

	activityByUUID := buildActivityContentMap(export.Account.Data, activityTypeByUUID)

	// Phase 2: 导入子资源 (需要重新遍历 contactRaws)
	for _, raw := range contactRaws {
		var mc MonicaContact
		if err := json.Unmarshal(raw, &mc); err != nil {
			continue
		}
		contactID, ok := contactUUIDMap[mc.UUID]
		if !ok {
			continue
		}
		s.importContactSubResources(
			s.DB, &mc, contactID, vaultID, accountID, userID, userContactID,
			fieldTypeByUUID, lifeEventTypeByUUID, activityByUUID, resp,
		)
	}

	// Phase 3: 导入 Relationships (account-level, 需要 contactUUIDMap 完成)
	relRaws := getCollectionByType(export.Account.Data, "relationships")
	for _, raw := range relRaws {
		var mr MonicaRelationship
		if err := json.Unmarshal(raw, &mr); err != nil {
			continue
		}

		contactIsID, ok1 := contactUUIDMap[mr.Properties.ContactIs]
		ofContactID, ok2 := contactUUIDMap[mr.Properties.OfContact]
		if !ok1 || !ok2 {
			resp.Errors = append(resp.Errors, fmt.Sprintf("relationship: unresolved contacts %s/%s", mr.Properties.ContactIs, mr.Properties.OfContact))
			continue
		}

		// 按名称匹配 RelationshipType（先 Name 再 NameReverseRelationship，case-insensitive）
		// Monica 4.x 用 kebab-case 的类型名（如 "sibling", "significant-other"），需要
		// 先映射到 Bonds seed 翻译键再回到本地化名称。
		var relType models.RelationshipType
		typeName := mr.Properties.Type
		candidates := monicaRelationshipNameCandidates(typeName)
		found := false
		for _, candidate := range candidates {
			if err := s.DB.Where("relationship_group_type_id IN (SELECT id FROM relationship_group_types WHERE account_id = ?) AND (LOWER(name) = LOWER(?) OR LOWER(name_translation_key) = LOWER(?))", accountID, candidate, candidate).First(&relType).Error; err == nil {
				found = true
				break
			}
			if err := s.DB.Where("relationship_group_type_id IN (SELECT id FROM relationship_group_types WHERE account_id = ?) AND (LOWER(name_reverse_relationship) = LOWER(?) OR LOWER(name_reverse_relationship_translation_key) = LOWER(?))", accountID, candidate, candidate).First(&relType).Error; err == nil {
				found = true
				break
			}
		}
		if !found {
			resp.Errors = append(resp.Errors, fmt.Sprintf("relationship type not found: %s", typeName))
			continue
		}

		// 检查重复
		var existing models.Relationship
		if err := s.DB.Where("contact_id = ? AND related_contact_id = ? AND relationship_type_id = ?", contactIsID, ofContactID, relType.ID).First(&existing).Error; err != nil {
			rel := models.Relationship{
				ContactID:          contactIsID,
				RelatedContactID:   ofContactID,
				RelationshipTypeID: relType.ID,
			}
			if err := s.DB.Create(&rel).Error; err == nil {
				resp.ImportedRelationships++
			}
		}
	}

	// Phase 4: 导入文件 (photos + documents)
	if s.UploadDir != "" {
		s.importPhotos(export.Account.Data, contactUUIDMap, vaultID, resp)
		s.importDocuments(export.Account.Data, contactUUIDMap, vaultID, resp)
	} else {
		photoCount := len(getCollectionByType(export.Account.Data, "photos"))
		docCount := len(getCollectionByType(export.Account.Data, "documents"))
		if photoCount+docCount > 0 {
			resp.Errors = append(resp.Errors, "photo/document import skipped: no upload directory configured")
		}
	}

	return resp, nil
}

func (s *MonicaImportService) importContact(
	tx *gorm.DB, mc *MonicaContact, vaultID, accountID, userID string,
	genderByUUID map[string]string, resp *dto.MonicaImportResponse,
) (string, bool, error) {
	var existingContact models.Contact
	if err := tx.Where("vault_id = ? AND distant_uuid = ?", vaultID, mc.UUID).First(&existingContact).Error; err == nil {
		resp.SkippedCount++
		return existingContact.ID, false, nil
	}

	var genderID *uint
	if mc.Properties.Gender != "" {
		if genderName, ok := genderByUUID[mc.Properties.Gender]; ok {
			var gender models.Gender
			if err := tx.Where("account_id = ? AND name = ?", accountID, genderName).First(&gender).Error; err == nil {
				genderID = &gender.ID
			}
		}
	}

	listed := !mc.Properties.IsPartial
	contact := models.Contact{
		VaultID:     vaultID,
		FirstName:   strPtrOrNil(mc.Properties.FirstName),
		MiddleName:  strPtrOrNil(mc.Properties.MiddleName),
		LastName:    strPtrOrNil(mc.Properties.LastName),
		Nickname:    strPtrOrNil(mc.Properties.Nickname),
		JobPosition: strPtrOrNil(mc.Properties.Job),
		GenderID:    genderID,
		DistantUUID: strPtrOrNil(mc.UUID),
		Listed:      listed,
	}
	if t, ok := parseMonicaTimestamp(mc.CreatedAt); ok {
		contact.CreatedAt = t
		contact.UpdatedAt = t
	}
	if t, ok := parseMonicaTimestamp(mc.UpdatedAt); ok {
		contact.UpdatedAt = t
	}
	if err := tx.Create(&contact).Error; err != nil {
		return "", false, fmt.Errorf("create contact: %w", err)
	}
	// GORM zero-value bool 陷阱：Listed default:true，要设为 false 必须先 Create 再 Update
	if !listed {
		tx.Model(&contact).Update("listed", false)
	}

	cvu := models.ContactVaultUser{
		ContactID:  contact.ID,
		VaultID:    vaultID,
		UserID:     userID,
		IsFavorite: mc.Properties.IsStarred,
	}
	if err := tx.Create(&cvu).Error; err != nil {
		return "", false, fmt.Errorf("create contact_vault_user: %w", err)
	}

	for _, tagName := range mc.Properties.Tags {
		if tagName == "" {
			continue
		}
		label, err := s.findOrCreateLabel(tx, vaultID, tagName)
		if err != nil {
			resp.Errors = append(resp.Errors, fmt.Sprintf("tag %q: %v", tagName, err))
			continue
		}
		cl := models.ContactLabel{
			ContactID: contact.ID,
			LabelID:   label.ID,
		}
		tx.Create(&cl) // 忽略错误（已存在则跳过）
	}

	if mc.Properties.Birthdate != nil {
		s.importSpecialDate(tx, contact.ID, vaultID, mc.Properties.Birthdate, "birthdate", "Birthdate", resp)
	}
	if mc.Properties.DeceasedDate != nil {
		s.importSpecialDate(tx, contact.ID, vaultID, mc.Properties.DeceasedDate, "deceased_date", "Deceased date", resp)
	}

	if s.feedRecorder != nil {
		s.feedRecorder.Record(contact.ID, userID, ActionContactCreated, "", nil, nil)
	}

	if s.searchEngine != nil {
		s.searchEngine.IndexContact(contact.ID, vaultID, ptrToStr(contact.FirstName), ptrToStr(contact.LastName), ptrToStr(contact.Nickname), ptrToStr(contact.JobPosition))
	}

	return contact.ID, true, nil
}

func (s *MonicaImportService) findOrCreateLabel(tx *gorm.DB, vaultID, name string) (*models.Label, error) {
	slug := slugify(name)
	var label models.Label
	err := tx.Where("vault_id = ? AND slug = ?", vaultID, slug).First(&label).Error
	if err == nil {
		return &label, nil
	}
	label = models.Label{
		VaultID: vaultID,
		Name:    name,
		Slug:    slug,
	}
	if err := tx.Create(&label).Error; err != nil {
		return nil, err
	}
	return &label, nil
}

func (s *MonicaImportService) importSpecialDate(
	tx *gorm.DB, contactID, vaultID string,
	sd *MonicaSpecialDate, internalType, label string,
	resp *dto.MonicaImportResponse,
) {
	var dateType models.ContactImportantDateType
	if err := tx.Where("vault_id = ? AND internal_type = ?", vaultID, internalType).First(&dateType).Error; err != nil {
		return
	}

	var year, month, day *int
	if sd.Date != "" && !sd.IsAgeBased {
		// Monica 日期格式可能是 "2006-01-02" 或 "2006-01-02T15:04:05Z"(ISO 8601)
		dateStr := sd.Date
		var t time.Time
		var parseErr error
		if t, parseErr = time.Parse(time.RFC3339, dateStr); parseErr != nil {
			t, parseErr = time.Parse("2006-01-02", dateStr)
		}
		if parseErr == nil {
			if !sd.IsYearUnknown {
				y := t.Year()
				year = &y
			}
			m := int(t.Month())
			d := t.Day()
			month = &m
			day = &d
		}
	}

	dtID := dateType.ID
	cid := models.ContactImportantDate{
		ContactID:                  contactID,
		ContactImportantDateTypeID: &dtID,
		Label:                      label,
		Year:                       year,
		Month:                      month,
		Day:                        day,
	}
	if err := tx.Create(&cid).Error; err != nil {
		resp.Errors = append(resp.Errors, fmt.Sprintf("importantdate %s: %v", internalType, err))
	}
}

func (s *MonicaImportService) importContactSubResources(
	tx *gorm.DB,
	mc *MonicaContact,
	contactID, vaultID, accountID, userID, userContactID string,
	fieldTypeByUUID map[string]MonicaContactFieldTypeRef,
	lifeEventTypeByUUID map[string]string,
	activityByUUID map[string]string,
	resp *dto.MonicaImportResponse,
) {
	s.importNotes(tx, mc, contactID, vaultID, userID, resp)
	s.importCalls(tx, mc, contactID, userID, resp)
	s.importTasks(tx, mc, contactID, userID, resp)
	s.importReminders(tx, mc, contactID, resp)
	s.importAddresses(tx, mc, contactID, vaultID, accountID, resp)
	s.importContactFields(tx, mc, contactID, accountID, fieldTypeByUUID, resp)
	s.importPets(tx, mc, contactID, accountID, resp)
	s.importGifts(tx, mc, contactID, resp)
	s.importDebtsAsLoans(tx, mc, contactID, vaultID, userContactID, resp)
	s.importLifeEvents(tx, mc, contactID, vaultID, lifeEventTypeByUUID, resp)
	s.importActivitiesAsNotes(tx, mc, contactID, vaultID, userID, activityByUUID, resp)
	s.importConversationsAsNotes(tx, mc, contactID, vaultID, userID, resp)
}

func (s *MonicaImportService) importNotes(
	tx *gorm.DB, mc *MonicaContact, contactID, vaultID, userID string,
	resp *dto.MonicaImportResponse,
) {
	for _, raw := range getCollectionByType(mc.Data, "notes") {
		var mn MonicaNote
		if err := json.Unmarshal(raw, &mn); err != nil {
			continue
		}
		note := models.Note{
			ContactID: contactID,
			VaultID:   vaultID,
			Body:      mn.Properties.Body,
			AuthorID:  &userID,
		}
		if t, ok := parseMonicaTimestamp(mn.CreatedAt); ok {
			note.CreatedAt = t
			note.UpdatedAt = t
		}
		if t, ok := parseMonicaTimestamp(mn.UpdatedAt); ok {
			note.UpdatedAt = t
		}
		if err := tx.Create(&note).Error; err == nil {
			resp.ImportedNotes++
		}
	}
}

func parseMonicaTimestamp(s string) (time.Time, bool) {
	if s == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{time.RFC3339, time.RFC3339Nano, "2006-01-02T15:04:05.000000Z", "2006-01-02 15:04:05", "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func (s *MonicaImportService) importCalls(
	tx *gorm.DB, mc *MonicaContact, contactID, userID string,
	resp *dto.MonicaImportResponse,
) {
	for _, raw := range getCollectionByType(mc.Data, "calls") {
		var mcall MonicaCall
		if err := json.Unmarshal(raw, &mcall); err != nil {
			continue
		}
		calledAt := time.Now()
		if mcall.Properties.CalledAt != "" {
			if t, err := time.Parse(time.RFC3339, mcall.Properties.CalledAt); err == nil {
				calledAt = t
			} else if t, err := time.Parse("2006-01-02", mcall.Properties.CalledAt); err == nil {
				calledAt = t
			}
		}
		whoInitiated := "user"
		if mcall.Properties.ContactCalled {
			whoInitiated = "contact"
		}
		call := models.Call{
			ContactID:    contactID,
			AuthorID:     &userID,
			AuthorName:   "Monica Import",
			CalledAt:     calledAt,
			Type:         "phone",
			WhoInitiated: whoInitiated,
			Answered:     true,
			Description:  strPtrOrNil(mcall.Properties.Content),
		}
		if err := tx.Create(&call).Error; err == nil {
			resp.ImportedCalls++
		}
	}
}

func (s *MonicaImportService) importTasks(
	tx *gorm.DB, mc *MonicaContact, contactID, userID string,
	resp *dto.MonicaImportResponse,
) {
	for _, raw := range getCollectionByType(mc.Data, "tasks") {
		var mt MonicaTask
		if err := json.Unmarshal(raw, &mt); err != nil {
			continue
		}
		task := models.ContactTask{
			ContactID:  contactID,
			Label:      mt.Properties.Title,
			AuthorID:   &userID,
			AuthorName: "Monica Import",
			Completed:  mt.Properties.Completed,
		}
		if mt.Properties.Description != "" {
			task.Description = strPtrOrNil(mt.Properties.Description)
		}
		if mt.Properties.CompletedAt != "" {
			if t, err := time.Parse(time.RFC3339, mt.Properties.CompletedAt); err == nil {
				task.CompletedAt = &t
			}
		}
		if err := tx.Create(&task).Error; err == nil {
			resp.ImportedTasks++
		}
	}
}

func (s *MonicaImportService) importReminders(
	tx *gorm.DB, mc *MonicaContact, contactID string,
	resp *dto.MonicaImportResponse,
) {
	freqMap := map[string]string{
		"one_time": "one_time",
		"week":     "recurring_week",
		"month":    "recurring_month",
		"year":     "recurring_year",
	}
	for _, raw := range getCollectionByType(mc.Data, "reminders") {
		var mr MonicaReminder
		if err := json.Unmarshal(raw, &mr); err != nil {
			continue
		}
		rType := "one_time"
		if t, ok := freqMap[mr.Properties.FrequencyType]; ok {
			rType = t
		}
		reminder := models.ContactReminder{
			ContactID:       contactID,
			Label:           mr.Properties.Title,
			Type:            rType,
			FrequencyNumber: &mr.Properties.FrequencyNumber,
		}
		if mr.Properties.InitialDate != "" {
			for _, layout := range []string{"2006-01-02", time.RFC3339} {
				if t, err := time.Parse(layout, mr.Properties.InitialDate); err == nil {
					d := t.Day()
					m := int(t.Month())
					y := t.Year()
					reminder.Day = &d
					reminder.Month = &m
					reminder.Year = &y
					break
				}
			}
		}
		if err := tx.Create(&reminder).Error; err == nil {
			resp.ImportedReminders++
		}
	}
}

func (s *MonicaImportService) importAddresses(
	tx *gorm.DB, mc *MonicaContact, contactID, vaultID, accountID string,
	resp *dto.MonicaImportResponse,
) {
	for _, raw := range getCollectionByType(mc.Data, "addresses") {
		var ma MonicaAddress
		if err := json.Unmarshal(raw, &ma); err != nil {
			continue
		}
		var addrType models.AddressType
		if err := tx.Where("account_id = ? AND LOWER(name) = LOWER(?)", accountID, ma.Properties.Name).First(&addrType).Error; err != nil {
			tx.Where("account_id = ?", accountID).First(&addrType)
		}
		addr := models.Address{
			VaultID:       vaultID,
			AddressTypeID: &addrType.ID,
			Line1:         strPtrOrNil(ma.Properties.Street),
			City:          strPtrOrNil(ma.Properties.City),
			Province:      strPtrOrNil(ma.Properties.Province),
			PostalCode:    strPtrOrNil(ma.Properties.PostalCode),
			Country:       strPtrOrNil(ma.Properties.Country),
		}
		if ma.Properties.Latitude != 0 {
			addr.Latitude = &ma.Properties.Latitude
		}
		if ma.Properties.Longitude != 0 {
			addr.Longitude = &ma.Properties.Longitude
		}
		if err := tx.Create(&addr).Error; err != nil {
			continue
		}
		ca := models.ContactAddress{ContactID: contactID, AddressID: addr.ID}
		if err := tx.Create(&ca).Error; err == nil {
			resp.ImportedAddresses++
		}
	}
}

func (s *MonicaImportService) importContactFields(
	tx *gorm.DB, mc *MonicaContact, contactID, accountID string,
	fieldTypeByUUID map[string]MonicaContactFieldTypeRef,
	resp *dto.MonicaImportResponse,
) {
	for _, raw := range getCollectionByType(mc.Data, "contact_fields") {
		var mcf MonicaContactField
		if err := json.Unmarshal(raw, &mcf); err != nil {
			continue
		}
		fieldTypeRef, ok := fieldTypeByUUID[mcf.Properties.Type]
		if !ok {
			resp.Errors = append(resp.Errors, fmt.Sprintf("unknown contact_field type UUID: %s", mcf.Properties.Type))
			continue
		}
		var ciType models.ContactInformationType
		found := false
		if fieldTypeRef.Properties.Protocol != "" {
			if err := tx.Where("account_id = ? AND protocol = ?", accountID, fieldTypeRef.Properties.Protocol).First(&ciType).Error; err == nil {
				found = true
			}
		}
		if !found && fieldTypeRef.Properties.Type != "" {
			if err := tx.Where("account_id = ? AND type = ?", accountID, fieldTypeRef.Properties.Type).First(&ciType).Error; err == nil {
				found = true
			}
		}
		if !found {
			resp.Errors = append(resp.Errors, fmt.Sprintf("unmatched contact_field type: %s", fieldTypeRef.Properties.Name))
			continue
		}
		ci := models.ContactInformation{
			ContactID: contactID,
			TypeID:    ciType.ID,
			Data:      mcf.Properties.Data,
		}
		tx.Create(&ci)
	}
}

func (s *MonicaImportService) importPets(
	tx *gorm.DB, mc *MonicaContact, contactID, accountID string,
	resp *dto.MonicaImportResponse,
) {
	for _, raw := range getCollectionByType(mc.Data, "pets") {
		var mp MonicaPet
		if err := json.Unmarshal(raw, &mp); err != nil {
			continue
		}
		var petCat models.PetCategory
		if err := tx.Where("account_id = ? AND LOWER(name) = LOWER(?)", accountID, mp.Properties.Category).First(&petCat).Error; err != nil {
			tx.Where("account_id = ?", accountID).First(&petCat)
		}
		name := strPtrOrNil(mp.Properties.Name)
		pet := models.Pet{
			ContactID:     contactID,
			Name:          name,
			PetCategoryID: petCat.ID,
		}
		tx.Create(&pet)
	}
}

func (s *MonicaImportService) importGifts(
	tx *gorm.DB, mc *MonicaContact, contactID string,
	resp *dto.MonicaImportResponse,
) {
	for _, raw := range getCollectionByType(mc.Data, "gifts") {
		var mg MonicaGift
		if err := json.Unmarshal(raw, &mg); err != nil {
			continue
		}
		giftType := "given"
		if mg.Properties.Status == "received" {
			giftType = "received"
		}
		gift := models.Gift{
			ContactID:   contactID,
			Name:        mg.Properties.Name,
			Type:        giftType,
			Description: strPtrOrNil(mg.Properties.Comment),
		}
		if mg.Properties.Amount > 0 {
			amt := int(mg.Properties.Amount * 100)
			gift.EstimatedPrice = &amt
		}
		if mg.Properties.Date != "" {
			if t, err := time.Parse("2006-01-02", mg.Properties.Date); err == nil {
				if giftType == "received" {
					gift.ReceivedAt = &t
				} else {
					gift.GivenAt = &t
				}
			}
		}
		tx.Create(&gift)
	}
}

func (s *MonicaImportService) importDebtsAsLoans(
	tx *gorm.DB, mc *MonicaContact, contactID, vaultID, userContactID string,
	resp *dto.MonicaImportResponse,
) {
	for _, raw := range getCollectionByType(mc.Data, "debts") {
		var md MonicaDebt
		if err := json.Unmarshal(raw, &md); err != nil {
			continue
		}
		if userContactID == "" {
			resp.Errors = append(resp.Errors, "debt: no user shadow contact")
			continue
		}
		loanType := "lent_to"
		loaner, loanee := userContactID, contactID
		if md.Properties.InDebt {
			loanType = "borrowed_from"
			loaner, loanee = contactID, userContactID
		}
		amt := int(md.Properties.Amount * 100)
		loan := models.Loan{
			VaultID:    vaultID,
			Name:       "Monica Import",
			Type:       loanType,
			AmountLent: &amt,
		}
		if md.Properties.Currency != "" {
			var cur models.Currency
			if err := tx.Where("code = ?", strings.ToUpper(md.Properties.Currency)).First(&cur).Error; err == nil {
				loan.CurrencyID = &cur.ID
			}
		}
		if err := tx.Create(&loan).Error; err != nil {
			continue
		}
		// GORM zero-value bool u9677u9631uff1aSettled default:falseuff0cu5df2u7ecfu7b26u5408u521du59cbu503cuff0cu5b8cu6210u65f6u518d Update
		if md.Properties.Status == "complete" {
			tx.Model(&loan).Update("settled", true)
		}
		cl := models.ContactLoan{LoanID: loan.ID, LoanerID: loaner, LoaneeID: loanee}
		tx.Create(&cl)
	}
}

func (s *MonicaImportService) importLifeEvents(
	tx *gorm.DB, mc *MonicaContact, contactID, vaultID string,
	lifeEventTypeByUUID map[string]string,
	resp *dto.MonicaImportResponse,
) {
	lifeRaws := getCollectionByType(mc.Data, "life_events")
	if len(lifeRaws) == 0 {
		return
	}
	now := time.Now()
	teLabel := "Monica Import"
	te := models.TimelineEvent{VaultID: vaultID, Label: &teLabel, StartedAt: now}
	if err := tx.Create(&te).Error; err != nil {
		return
	}

	for _, raw := range lifeRaws {
		var ml MonicaLifeEvent
		if err := json.Unmarshal(raw, &ml); err != nil {
			continue
		}
		happenedAt := now
		if ml.Properties.HappenedAt != "" {
			if t, err := time.Parse(time.RFC3339, ml.Properties.HappenedAt); err == nil {
				happenedAt = t
			} else if t, err := time.Parse("2006-01-02", ml.Properties.HappenedAt); err == nil {
				happenedAt = t
			}
		}
		var let models.LifeEventType
		if err := tx.Joins("JOIN life_event_categories ON life_event_types.life_event_category_id = life_event_categories.id").
			Where("life_event_categories.vault_id = ?", vaultID).
			First(&let).Error; err != nil {
			continue
		}
		le := models.LifeEvent{
			TimelineEventID: te.ID,
			LifeEventTypeID: let.ID,
			HappenedAt:      happenedAt,
			Summary:         strPtrOrNil(ml.Properties.Name),
			Description:     strPtrOrNil(ml.Properties.Note),
		}
		if err := tx.Create(&le).Error; err != nil {
			continue
		}
		lep := models.LifeEventParticipant{ContactID: contactID, LifeEventID: le.ID}
		tx.Create(&lep)
		resp.ImportedLifeEvents++
	}
}

func (s *MonicaImportService) importActivitiesAsNotes(
	tx *gorm.DB, mc *MonicaContact, contactID, vaultID, userID string,
	activityByUUID map[string]string,
	resp *dto.MonicaImportResponse,
) {
	// contact.data[type="activities"] values u662f UUID string u6570u7ec4uff0cu975e MonicaActivity u5bf9u8c61
	for _, raw := range getCollectionByType(mc.Data, "activities") {
		var uuidStr string
		if err := json.Unmarshal(raw, &uuidStr); err != nil {
			continue
		}
		activityContent, ok := activityByUUID[uuidStr]
		if !ok {
			continue
		}
		note := models.Note{
			ContactID: contactID,
			VaultID:   vaultID,
			Body:      activityContent,
			AuthorID:  &userID,
		}
		if err := tx.Create(&note).Error; err == nil {
			resp.ImportedNotes++
		}
	}
}

func (s *MonicaImportService) importConversationsAsNotes(
	tx *gorm.DB, mc *MonicaContact, contactID, vaultID, userID string,
	resp *dto.MonicaImportResponse,
) {
	for _, raw := range getCollectionByType(mc.Data, "conversations") {
		var mconv MonicaConversation
		if err := json.Unmarshal(raw, &mconv); err != nil {
			continue
		}

		var lines []string
		for _, msg := range mconv.Properties.Messages {
			sender := "Contact"
			if msg.Properties.WrittenByMe {
				sender = "Me"
			}
			lines = append(lines, fmt.Sprintf("[%s] %s: %s", msg.Properties.WrittenAt, sender, msg.Properties.Content))
		}
		body := strings.Join(lines, "\n")
		title := fmt.Sprintf("Conversation (%s)", mconv.Properties.HappenedAt)

		note := models.Note{
			ContactID: contactID,
			VaultID:   vaultID,
			Title:     strPtrOrNil(title),
			Body:      body,
			AuthorID:  &userID,
		}
		if err := tx.Create(&note).Error; err == nil {
			resp.ImportedNotes++
		}
	}
}

func buildActivityContentMap(accountData []MonicaCollection, activityTypeByUUID map[string]string) map[string]string {
	result := make(map[string]string)
	for _, raw := range getCollectionByType(accountData, "activities") {
		var ma MonicaActivity
		if err := json.Unmarshal(raw, &ma); err != nil {
			continue
		}
		typeName := activityTypeByUUID[ma.Properties.Type]
		var body string
		if typeName != "" {
			body = fmt.Sprintf("[Activity: %s] %s", typeName, ma.Properties.Summary)
		} else {
			body = ma.Properties.Summary
		}
		if ma.Properties.Description != "" {
			body += "\n" + ma.Properties.Description
		}
		result[ma.UUID] = body
	}
	return result
}

func slugify(name string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(name), " ", "-"))
}

// monicaRelationshipTypeAliases maps Monica 4.x relationship type strings
// (kebab/snake form, see Monica's RelationshipType seeder) to the Bonds
// seed translation keys defined in seed_account.go. Anything not listed
// here falls back to a direct localized-name match.
var monicaRelationshipTypeAliases = map[string]string{
	"sibling":           "seed.relationship_types.brother_sister",
	"siblings":          "seed.relationship_types.brother_sister",
	"brother":           "seed.relationship_types.brother_sister",
	"sister":            "seed.relationship_types.brother_sister",
	"partner":           "seed.relationship_types.significant_other",
	"significant_other": "seed.relationship_types.significant_other",
	"significant-other": "seed.relationship_types.significant_other",
	"parent":            "seed.relationship_types.parent",
	"child":             "seed.relationship_types.child",
	"grandparent":       "seed.relationship_types.grand_parent",
	"grand_parent":      "seed.relationship_types.grand_parent",
	"grandchild":        "seed.relationship_types.grand_child",
	"grand_child":       "seed.relationship_types.grand_child",
	"uncle":             "seed.relationship_types.uncle_aunt",
	"aunt":              "seed.relationship_types.uncle_aunt",
	"uncle_aunt":        "seed.relationship_types.uncle_aunt",
	"nephew":            "seed.relationship_types.nephew_niece",
	"niece":             "seed.relationship_types.nephew_niece",
	"nephew_niece":      "seed.relationship_types.nephew_niece",
	"cousin":            "seed.relationship_types.cousin",
	"godparent":         "seed.relationship_types.godparent",
	"godchild":          "seed.relationship_types.godchild",
	"friend":            "seed.relationship_types.friend",
	"best_friend":       "seed.relationship_types.best_friend",
	"colleague":         "seed.relationship_types.colleague",
	"boss":              "seed.relationship_types.boss",
	"subordinate":       "seed.relationship_types.subordinate",
	"mentor":            "seed.relationship_types.mentor",
	"protege":           "seed.relationship_types.protege",
	"spouse":            "seed.relationship_types.spouse",
	"date":              "seed.relationship_types.date",
	"lover":             "seed.relationship_types.lover",
	"in_love_with":      "seed.relationship_types.in_love_with",
	"loved_by":          "seed.relationship_types.loved_by",
	"ex":                "seed.relationship_types.ex_boyfriend",
	"ex_boyfriend":      "seed.relationship_types.ex_boyfriend",
	"ex_girlfriend":     "seed.relationship_types.ex_boyfriend",
}

func monicaRelationshipNameCandidates(monicaType string) []string {
	normalized := strings.ToLower(strings.TrimSpace(monicaType))
	candidates := []string{normalized}
	if alias, ok := monicaRelationshipTypeAliases[normalized]; ok {
		candidates = append(candidates, alias)
	}
	if dashed := strings.ReplaceAll(normalized, "_", "-"); dashed != normalized {
		candidates = append(candidates, dashed)
		if alias, ok := monicaRelationshipTypeAliases[dashed]; ok {
			candidates = append(candidates, alias)
		}
	}
	if underscored := strings.ReplaceAll(normalized, "-", "_"); underscored != normalized {
		candidates = append(candidates, underscored)
		if alias, ok := monicaRelationshipTypeAliases[underscored]; ok {
			candidates = append(candidates, alias)
		}
	}
	return candidates
}

func buildGenderMap(refs []MonicaGenderRef) map[string]string {
	m := make(map[string]string, len(refs))
	for _, r := range refs {
		m[r.UUID] = r.Properties.Name
	}
	return m
}

func buildFieldTypeMap(refs []MonicaContactFieldTypeRef) map[string]MonicaContactFieldTypeRef {
	m := make(map[string]MonicaContactFieldTypeRef, len(refs))
	for _, r := range refs {
		m[r.UUID] = r
	}
	return m
}

func buildLifeEventTypeMap(refs []MonicaLifeEventTypeRef) map[string]string {
	m := make(map[string]string, len(refs))
	for _, r := range refs {
		m[r.UUID] = r.Properties.Name
	}
	return m
}

func buildActivityTypeMap(refs []MonicaActivityTypeRef) map[string]string {
	m := make(map[string]string, len(refs))
	for _, r := range refs {
		m[r.UUID] = r.Properties.Name
	}
	return m
}

// MIME 白名单 — 仅允许安全的文件类型导入
var monicaImportAllowedMimeTypes = map[string]string{
	"image/jpeg":      ".jpg",
	"image/png":       ".png",
	"image/gif":       ".gif",
	"image/webp":      ".webp",
	"application/pdf": ".pdf",
}

func (s *MonicaImportService) importPhotos(
	accountData []MonicaCollection,
	contactUUIDMap map[string]string,
	vaultID string,
	resp *dto.MonicaImportResponse,
) {
	photoFileMap := make(map[string]uint)

	for _, raw := range getCollectionByType(accountData, "photos") {
		var mp MonicaPhoto
		if err := json.Unmarshal(raw, &mp); err != nil {
			continue
		}

		fileID, err := s.saveBase64File(
			mp.Properties.DataURL, mp.Properties.OriginalFilename,
			mp.Properties.MimeType, mp.Properties.Filesize, vaultID, "photo",
		)
		if err != nil {
			resp.Errors = append(resp.Errors, fmt.Sprintf("photo %s: %v", mp.UUID, err))
			continue
		}
		photoFileMap[mp.UUID] = fileID
		resp.ImportedPhotos++
	}

	for _, raw := range getCollectionByType(accountData, "contacts") {
		var mc MonicaContact
		if err := json.Unmarshal(raw, &mc); err != nil {
			continue
		}
		contactID, ok := contactUUIDMap[mc.UUID]
		if !ok {
			continue
		}

		// contact.data[type="photos"] values 是 UUID string 数组
		photoUUIDs := getCollectionByType(mc.Data, "photos")
		for i, puRaw := range photoUUIDs {
			var photoUUID string
			if err := json.Unmarshal(puRaw, &photoUUID); err != nil {
				continue
			}

			fileID, ok := photoFileMap[photoUUID]
			if !ok {
				continue
			}

			s.DB.Model(&models.File{}).Where("id = ?", fileID).Updates(map[string]interface{}{
				"ufileable_id": contactID,
			})

			// 第一张照片（或 avatar 指定的照片）设为联系人头像
			if i == 0 {
				s.DB.Model(&models.Contact{}).Where("id = ?", contactID).Update("file_id", fileID)
			}
		}
	}
}

func (s *MonicaImportService) importDocuments(
	accountData []MonicaCollection,
	contactUUIDMap map[string]string,
	vaultID string,
	resp *dto.MonicaImportResponse,
) {
	docFileMap := make(map[string]uint)

	for _, raw := range getCollectionByType(accountData, "documents") {
		var md MonicaDocument
		if err := json.Unmarshal(raw, &md); err != nil {
			continue
		}

		fileID, err := s.saveBase64File(
			md.Properties.DataURL, md.Properties.OriginalFilename,
			md.Properties.MimeType, md.Properties.Filesize, vaultID, "document",
		)
		if err != nil {
			resp.Errors = append(resp.Errors, fmt.Sprintf("document %s: %v", md.UUID, err))
			continue
		}
		docFileMap[md.UUID] = fileID
		resp.ImportedDocuments++
	}

	for _, raw := range getCollectionByType(accountData, "contacts") {
		var mc MonicaContact
		if err := json.Unmarshal(raw, &mc); err != nil {
			continue
		}
		contactID, ok := contactUUIDMap[mc.UUID]
		if !ok {
			continue
		}

		for _, duRaw := range getCollectionByType(mc.Data, "documents") {
			var docUUID string
			if err := json.Unmarshal(duRaw, &docUUID); err != nil {
				continue
			}

			fileID, ok := docFileMap[docUUID]
			if !ok {
				continue
			}

			s.DB.Model(&models.File{}).Where("id = ?", fileID).Updates(map[string]interface{}{
				"ufileable_id": contactID,
			})
		}
	}
}

func (s *MonicaImportService) saveBase64File(
	dataURL, originalFilename, mimeType string, filesize int, vaultID, fileType string,
) (uint, error) {
	if dataURL == "" {
		return 0, fmt.Errorf("empty dataUrl")
	}

	// 解析 data:{mime};base64,{payload}
	parts := strings.SplitN(dataURL, ",", 2)
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid dataUrl format")
	}

	decoded, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return 0, fmt.Errorf("base64 decode failed: %w", err)
	}

	ext, ok := monicaImportAllowedMimeTypes[mimeType]
	if !ok {
		return 0, fmt.Errorf("unsupported mime type: %s", mimeType)
	}

	// 存储路径: {uploadDir}/{yyyy/MM/dd}/{uuid}{ext}
	now := time.Now()
	dir := filepath.Join(s.UploadDir, now.Format("2006/01/02"))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return 0, fmt.Errorf("create dir: %w", err)
	}

	fileUUID := uuid.New().String()
	fileName := fileUUID + ext
	filePath := filepath.Join(dir, fileName)
	if err := os.WriteFile(filePath, decoded, 0644); err != nil {
		return 0, fmt.Errorf("write file: %w", err)
	}

	name := originalFilename
	if name == "" {
		name = fileName
	}
	size := filesize
	if size == 0 {
		size = len(decoded)
	}

	file := models.File{
		VaultID:     vaultID,
		UUID:        fileUUID,
		Name:        name,
		MimeType:    mimeType,
		Size:        size,
		Type:        fileType,
		OriginalURL: &filePath,
	}
	if err := s.DB.Create(&file).Error; err != nil {
		os.Remove(filePath)
		return 0, fmt.Errorf("create file record: %w", err)
	}
	return file.ID, nil
}

// ==== Top-level structs ====

type MonicaExport struct {
	Version    string        `json:"version"`
	AppVersion string        `json:"app_version"`
	ExportDate string        `json:"export_date"`
	URL        string        `json:"url"`
	ExportedBy string        `json:"exported_by"`
	Account    MonicaAccount `json:"account"`
}

type MonicaAccount struct {
	UUID       string                  `json:"uuid"`
	CreatedAt  string                  `json:"created_at"`
	UpdatedAt  string                  `json:"updated_at"`
	Data       []MonicaCollection      `json:"data"`
	Properties MonicaAccountProperties `json:"properties"`
	Instance   MonicaInstance          `json:"instance"`
}

type MonicaCollection struct {
	Count  int               `json:"count"`
	Type   string            `json:"type"`
	Values []json.RawMessage `json:"values"`
}

type MonicaAccountProperties struct {
	DefaultGender string `json:"default_gender"`
	// journal_entries, modules, reminder_rules, audit_logs 收到忽略
}

type MonicaInstance struct {
	Genders                []MonicaGenderRef            `json:"genders"`
	ContactFieldTypes      []MonicaContactFieldTypeRef  `json:"contact_field_types"`
	ActivityTypes          []MonicaActivityTypeRef      `json:"activity_types"`
	ActivityTypeCategories []MonicaActivityTypeRef      `json:"activity_type_categories"`
	LifeEventTypes         []MonicaLifeEventTypeRef     `json:"life_event_types"`
	LifeEventCategories    []MonicaLifeEventCategoryRef `json:"life_event_categories"`
}

// ==== Instance reference structs ====

type MonicaGenderRef struct {
	UUID       string `json:"uuid"`
	Properties struct {
		Name string `json:"name"`
	} `json:"properties"`
}

type MonicaContactFieldTypeRef struct {
	UUID       string `json:"uuid"`
	Properties struct {
		Name     string `json:"name"`
		Protocol string `json:"protocol"`
		Type     string `json:"type"`
	} `json:"properties"`
}

type MonicaActivityTypeRef struct {
	UUID       string `json:"uuid"`
	Properties struct {
		Name           string `json:"name"`
		TranslationKey string `json:"translation_key"`
		Category       string `json:"category"`
	} `json:"properties"`
}

type MonicaLifeEventTypeRef struct {
	UUID       string `json:"uuid"`
	Properties struct {
		Name     string `json:"name"`
		Category string `json:"category"`
	} `json:"properties"`
}

type MonicaLifeEventCategoryRef struct {
	UUID       string `json:"uuid"`
	Properties struct {
		Name           string `json:"name"`
		TranslationKey string `json:"translation_key"`
	} `json:"properties"`
}

// ==== Contact entity ====

type MonicaContact struct {
	UUID       string             `json:"uuid"`
	CreatedAt  string             `json:"created_at"`
	UpdatedAt  string             `json:"updated_at"`
	Properties MonicaContactProps `json:"properties"`
	Data       []MonicaCollection `json:"data"`
}

type MonicaContactProps struct {
	FirstName       string             `json:"first_name"`
	MiddleName      string             `json:"middle_name"`
	LastName        string             `json:"last_name"`
	Nickname        string             `json:"nickname"`
	Description     string             `json:"description"`
	IsStarred       bool               `json:"is_starred"`
	IsPartial       bool               `json:"is_partial"`
	IsActive        bool               `json:"is_active"`
	IsDead          bool               `json:"is_dead"`
	Job             string             `json:"job"`
	Company         string             `json:"company"`
	FoodPreferences string             `json:"food_preferences"`
	LastTalkedTo    string             `json:"last_talked_to"`
	Gender          string             `json:"gender"`
	Tags            []string           `json:"tags"`
	Birthdate       *MonicaSpecialDate `json:"birthdate"`
	DeceasedDate    *MonicaSpecialDate `json:"deceased_date"`
	FirstMetDate    *MonicaSpecialDate `json:"first_met_date"`
	FirstMetThrough string             `json:"first_met_through"`
	Avatar          *MonicaAvatar      `json:"avatar"`
}

type MonicaSpecialDate struct {
	UUID          string `json:"uuid"`
	IsAgeBased    bool   `json:"is_age_based"`
	IsYearUnknown bool   `json:"is_year_unknown"`
	Date          string `json:"date"`
}

type MonicaAvatar struct {
	AvatarSource    string `json:"avatar_source"`
	AvatarPhotoUUID string `json:"avatar_photo"`
	HasAvatar       bool   `json:"has_avatar"`
}

// ==== Sub-resource entity structs ====

type MonicaNote struct {
	UUID       string `json:"uuid"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
	Properties struct {
		Body       string `json:"body"`
		IsFavorite bool   `json:"is_favorite"`
	} `json:"properties"`
}

type MonicaCall struct {
	UUID       string `json:"uuid"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
	Properties struct {
		CalledAt      string   `json:"called_at"`
		Content       string   `json:"content"`
		ContactCalled bool     `json:"contact_called"`
		Emotions      []string `json:"emotions"`
	} `json:"properties"`
}

type MonicaTask struct {
	UUID       string `json:"uuid"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
	Properties struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Completed   bool   `json:"completed"`
		CompletedAt string `json:"completed_at"`
	} `json:"properties"`
}

type MonicaReminder struct {
	UUID       string `json:"uuid"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
	Properties struct {
		InitialDate     string `json:"initial_date"`
		Title           string `json:"title"`
		Description     string `json:"description"`
		FrequencyType   string `json:"frequency_type"`
		FrequencyNumber int    `json:"frequency_number"`
		Inactive        bool   `json:"inactive"`
	} `json:"properties"`
}

type MonicaAddress struct {
	UUID       string `json:"uuid"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
	Properties struct {
		Name       string  `json:"name"`
		Street     string  `json:"street"`
		City       string  `json:"city"`
		Province   string  `json:"province"`
		PostalCode string  `json:"postal_code"`
		Latitude   float64 `json:"latitude"`
		Longitude  float64 `json:"longitude"`
		Country    string  `json:"country"`
	} `json:"properties"`
}

type MonicaGift struct {
	UUID       string `json:"uuid"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
	Properties struct {
		Name      string   `json:"name"`
		Comment   string   `json:"comment"`
		URL       string   `json:"url"`
		Amount    float64  `json:"amount"`
		Status    string   `json:"status"`
		Date      string   `json:"date"`
		Recipient string   `json:"recipient"`
		Photos    []string `json:"photos"`
	} `json:"properties"`
}

type MonicaDebt struct {
	UUID       string `json:"uuid"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
	Properties struct {
		Amount   float64 `json:"amount"`
		Currency string  `json:"currency"`
		Status   string  `json:"status"`
		InDebt   bool    `json:"in_debt"`
	} `json:"properties"`
}

type MonicaContactField struct {
	UUID       string `json:"uuid"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
	Properties struct {
		Data string `json:"data"`
		Type string `json:"type"` // Monica ContactFieldType UUID
	} `json:"properties"`
}

type MonicaPet struct {
	UUID       string `json:"uuid"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
	Properties struct {
		Name     string `json:"name"`
		Category string `json:"category"` // category name string, NOT UUID
	} `json:"properties"`
}

type MonicaLifeEvent struct {
	UUID       string `json:"uuid"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
	Properties struct {
		Name       string `json:"name"`
		Note       string `json:"note"`
		HappenedAt string `json:"happened_at"`
		Type       string `json:"type"` // Monica LifeEventType UUID
	} `json:"properties"`
}

type MonicaConversation struct {
	UUID       string `json:"uuid"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
	Properties struct {
		HappenedAt string          `json:"happened_at"`
		Messages   []MonicaMessage `json:"messages"`
	} `json:"properties"`
}

type MonicaMessage struct {
	UUID       string `json:"uuid"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
	Properties struct {
		Content     string `json:"content"`
		WrittenAt   string `json:"written_at"`
		WrittenByMe bool   `json:"written_by_me"`
	} `json:"properties"`
}

type MonicaActivity struct {
	UUID       string `json:"uuid"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
	Properties struct {
		Summary     string `json:"summary"`
		Description string `json:"description"`
		HappenedAt  string `json:"happened_at"`
		Type        string `json:"type"` // Monica ActivityType UUID
	} `json:"properties"`
}

type MonicaRelationship struct {
	UUID       string `json:"uuid"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
	Properties struct {
		Type      string `json:"type"`       // relationship type name e.g. "partner"
		ContactIs string `json:"contact_is"` // contact UUID
		OfContact string `json:"of_contact"` // contact UUID
	} `json:"properties"`
}

type MonicaPhoto struct {
	UUID       string `json:"uuid"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
	Properties struct {
		OriginalFilename string `json:"original_filename"`
		Filesize         int    `json:"filesize"`
		MimeType         string `json:"mime_type"`
		DataURL          string `json:"dataUrl"`
	} `json:"properties"`
}

type MonicaDocument struct {
	UUID       string `json:"uuid"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
	Properties struct {
		OriginalFilename string `json:"original_filename"`
		Filesize         int    `json:"filesize"`
		Type             string `json:"type"`
		MimeType         string `json:"mime_type"`
		DataURL          string `json:"dataUrl"`
	} `json:"properties"`
}
