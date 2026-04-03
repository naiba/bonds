package services

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

const MonicaExportVersion = "1.0-preview.1"

// ParseMonicaExport 解析 Monica 4.x JSON 导出数据
func ParseMonicaExport(data []byte) (*MonicaExport, error) {
	var export MonicaExport
	if err := json.Unmarshal(data, &export); err != nil {
		return nil, fmt.Errorf("failed to parse Monica export: %w", err)
	}
	if export.Version != MonicaExportVersion {
		return nil, fmt.Errorf("unsupported Monica export version: %s (expected %s)", export.Version, MonicaExportVersion)
	}
	return &export, nil
}

// getCollectionByType 从 []MonicaCollection 中按 type 获取 values
func getCollectionByType(collections []MonicaCollection, entityType string) []json.RawMessage {
	for _, c := range collections {
		if c.Type == entityType {
			return c.Values
		}
	}
	return nil
}

type MonicaImportService struct {
	DB *gorm.DB
}

func NewMonicaImportService(db *gorm.DB) *MonicaImportService {
	return &MonicaImportService{DB: db}
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

	// 其余 Phase (sub-resources, relationships, files) 留给后续 Task 4/5/6
	_ = fieldTypeByUUID
	_ = lifeEventTypeByUUID
	_ = activityTypeByUUID
	_ = contactUUIDMap

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

func slugify(name string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(name), " ", "-"))
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
