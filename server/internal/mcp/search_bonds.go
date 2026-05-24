package mcp

import (
	"fmt"
	"strings"
	"time"

	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/search"
	"github.com/naiba/bonds/internal/services"
	"gorm.io/gorm"
)

const (
	maxSearchQueryLength = 200
	maxSearchPerPage     = 50
)

type SearchService interface {
	Search(vaultID, query string, page, perPage int) (*search.SearchResponse, error)
}

type VaultAccessChecker interface {
	CheckUserVaultAccess(userID, vaultID string, requiredPerm int) error
}

type BondsSearcher struct {
	db            *gorm.DB
	searchService SearchService
	vaultService  VaultAccessChecker
}

type SearchBondsArgs struct {
	VaultID string `json:"vault_id"`
	Query   string `json:"query"`
	Page    int    `json:"page"`
	PerPage int    `json:"per_page"`
}

type SearchBondsResult struct {
	Query        string             `json:"query"`
	VaultID      string             `json:"vault_id"`
	Capabilities SearchCapabilities `json:"capabilities"`
	Results      []SearchItem       `json:"results"`
	Total        int                `json:"total"`
}

type SearchCapabilities struct {
	StructuredSearch     bool `json:"structured_search"`
	LexicalSearch        bool `json:"lexical_search"`
	SemanticVectorSearch bool `json:"semantic_vector_search"`
}

type SearchItem struct {
	Type        string              `json:"type"`
	ID          string              `json:"id"`
	Title       string              `json:"title"`
	ResourceURI string              `json:"resource_uri"`
	Reason      string              `json:"match_reason"`
	Highlights  map[string][]string `json:"highlights,omitempty"`
	Score       float64             `json:"score,omitempty"`
}

func NewBondsSearcher(db *gorm.DB, searchService SearchService, vaultService VaultAccessChecker) *BondsSearcher {
	return &BondsSearcher{db: db, searchService: searchService, vaultService: vaultService}
}

func (s *BondsSearcher) Search(userID string, args SearchBondsArgs) (*SearchBondsResult, error) {
	query := strings.TrimSpace(args.Query)
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}
	if len(query) > maxSearchQueryLength {
		query = query[:maxSearchQueryLength]
	}
	if args.VaultID == "" {
		return nil, fmt.Errorf("vault_id is required")
	}
	if err := s.vaultService.CheckUserVaultAccess(userID, args.VaultID, models.PermissionViewer); err != nil {
		return nil, err
	}
	page, perPage := normalizePagination(args.Page, args.PerPage)
	items := make([]SearchItem, 0)

	if s.searchService != nil {
		resp, err := s.searchService.Search(args.VaultID, query, page, perPage)
		if err != nil {
			return nil, err
		}
		bleveItems, err := s.visibleBleveItems(args.VaultID, itemsFromBleve(resp))
		if err != nil {
			return nil, err
		}
		items = append(items, bleveItems...)
	}

	sqlItems, err := s.sqlSearch(args.VaultID, query, perPage)
	if err != nil {
		return nil, err
	}
	items = mergeSearchItems(items, sqlItems)

	return &SearchBondsResult{
		Query:   query,
		VaultID: args.VaultID,
		Capabilities: SearchCapabilities{
			StructuredSearch:     true,
			LexicalSearch:        true,
			SemanticVectorSearch: false,
		},
		Results: items,
		Total:   len(items),
	}, nil
}

func normalizePagination(page, perPage int) (int, int) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > maxSearchPerPage {
		perPage = maxSearchPerPage
	}
	return page, perPage
}

func itemsFromBleve(resp *search.SearchResponse) []SearchItem {
	if resp == nil {
		return []SearchItem{}
	}
	items := make([]SearchItem, 0, len(resp.Contacts)+len(resp.Notes))
	for _, result := range resp.Contacts {
		items = append(items, SearchItem{
			Type:        "contact",
			ID:          result.ID,
			Title:       result.Name,
			ResourceURI: "bonds://contact/" + result.ID,
			Reason:      "matched full-text contact index",
			Highlights:  result.Highlights,
			Score:       result.Score,
		})
	}
	for _, result := range resp.Notes {
		items = append(items, SearchItem{
			Type:        "note",
			ID:          result.ID,
			Title:       result.Name,
			ResourceURI: "bonds://note/" + result.ID,
			Reason:      "matched full-text note index",
			Highlights:  result.Highlights,
			Score:       result.Score,
		})
	}
	return items
}

func (s *BondsSearcher) sqlSearch(vaultID, query string, limit int) ([]SearchItem, error) {
	items := make([]SearchItem, 0)
	likeTerm := "%" + strings.ToLower(query) + "%"

	contacts, err := s.searchContacts(vaultID, likeTerm, limit)
	if err != nil {
		return nil, err
	}
	items = append(items, contacts...)

	contactInfo, err := s.searchContactInformation(vaultID, likeTerm, limit)
	if err != nil {
		return nil, err
	}
	items = append(items, contactInfo...)

	notes, err := s.searchNotes(vaultID, likeTerm, limit)
	if err != nil {
		return nil, err
	}
	items = append(items, notes...)

	tasks, err := s.searchTasks(vaultID, query, likeTerm, limit)
	if err != nil {
		return nil, err
	}
	items = append(items, tasks...)

	reminders, err := s.searchReminders(vaultID, query, likeTerm, limit)
	if err != nil {
		return nil, err
	}
	items = append(items, reminders...)

	dates, err := s.searchImportantDates(vaultID, query, likeTerm, limit)
	if err != nil {
		return nil, err
	}
	items = append(items, dates...)

	return items, nil
}

func (s *BondsSearcher) searchContacts(vaultID, likeTerm string, limit int) ([]SearchItem, error) {
	var contacts []models.Contact
	err := s.db.Where("vault_id = ? AND listed = ?", vaultID, true).
		Where(s.db.Where("LOWER(first_name) LIKE ?", likeTerm).
			Or("LOWER(last_name) LIKE ?", likeTerm).
			Or("LOWER(middle_name) LIKE ?", likeTerm).
			Or("LOWER(nickname) LIKE ?", likeTerm).
			Or("LOWER(maiden_name) LIKE ?", likeTerm).
			Or("LOWER(job_position) LIKE ?", likeTerm)).
		Limit(limit).
		Find(&contacts).Error
	if err != nil {
		return nil, err
	}
	items := make([]SearchItem, 0, len(contacts))
	for _, contact := range contacts {
		items = append(items, SearchItem{
			Type:        "contact",
			ID:          contact.ID,
			Title:       contactName(&contact),
			ResourceURI: "bonds://contact/" + contact.ID,
			Reason:      "matched contact fields",
		})
	}
	return items, nil
}

func (s *BondsSearcher) searchContactInformation(vaultID, likeTerm string, limit int) ([]SearchItem, error) {
	type row struct {
		ID        uint
		Data      string
		ContactID string
		FirstName *string
		LastName  *string
	}
	var rows []row
	err := s.db.Table("contact_information").
		Select("contact_information.id, contact_information.data, contacts.id AS contact_id, contacts.first_name, contacts.last_name").
		Joins("JOIN contacts ON contacts.id = contact_information.contact_id").
		Where("contacts.vault_id = ? AND contacts.listed = ?", vaultID, true).
		Where("LOWER(contact_information.data) LIKE ?", likeTerm).
		Limit(limit).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	items := make([]SearchItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, SearchItem{
			Type:        "contact_information",
			ID:          fmt.Sprint(row.ID),
			Title:       strings.TrimSpace(ptrString(row.FirstName) + " " + ptrString(row.LastName)),
			ResourceURI: "bonds://contact/" + row.ContactID,
			Reason:      "matched contact information: " + row.Data,
		})
	}
	return items, nil
}

func (s *BondsSearcher) searchNotes(vaultID, likeTerm string, limit int) ([]SearchItem, error) {
	var notes []models.Note
	err := s.db.Model(&models.Note{}).
		Select("notes.*").
		Joins("JOIN contacts ON contacts.id = notes.contact_id").
		Where("notes.vault_id = ? AND contacts.listed = ?", vaultID, true).
		Where(s.db.Where("LOWER(notes.title) LIKE ?", likeTerm).Or("LOWER(notes.body) LIKE ?", likeTerm)).
		Order("notes.updated_at DESC").
		Limit(limit).
		Find(&notes).Error
	if err != nil {
		return nil, err
	}
	items := make([]SearchItem, 0, len(notes))
	for _, note := range notes {
		items = append(items, SearchItem{
			Type:        "note",
			ID:          fmt.Sprint(note.ID),
			Title:       ptrString(note.Title),
			ResourceURI: "bonds://note/" + fmt.Sprint(note.ID),
			Reason:      "matched note title or body",
		})
	}
	return items, nil
}

func (s *BondsSearcher) searchTasks(vaultID, query, likeTerm string, limit int) ([]SearchItem, error) {
	db := s.db.Model(&models.ContactTask{}).
		Where("contact_tasks.vault_id = ?", vaultID).
		Where(visibleTaskCondition(), vaultID, true)
	switch normalizedNaturalQuery(query) {
	case "overdue_tasks":
		db = db.Where("contact_tasks.status != ? AND contact_tasks.due_at IS NOT NULL AND contact_tasks.due_at < ?", models.TaskStatusDone, time.Now())
	case "open_tasks":
		db = db.Where("contact_tasks.status != ?", models.TaskStatusDone)
	default:
		db = db.Where(s.db.Where("LOWER(contact_tasks.label) LIKE ?", likeTerm).Or("LOWER(contact_tasks.description) LIKE ?", likeTerm))
	}
	var tasks []models.ContactTask
	if err := db.Order("contact_tasks.updated_at DESC").Limit(limit).Find(&tasks).Error; err != nil {
		return nil, err
	}
	items := make([]SearchItem, 0, len(tasks))
	for _, task := range tasks {
		items = append(items, SearchItem{
			Type:        "task",
			ID:          fmt.Sprint(task.ID),
			Title:       task.Label,
			ResourceURI: "bonds://task/" + fmt.Sprint(task.ID),
			Reason:      "matched task fields or structured task filter",
		})
	}
	return items, nil
}

func visibleTaskCondition() string {
	return `(NOT EXISTS (
		SELECT 1 FROM task_contacts
		WHERE task_contacts.contact_task_id = contact_tasks.id
	) OR EXISTS (
		SELECT 1 FROM task_contacts
		JOIN contacts ON contacts.id = task_contacts.contact_id
		WHERE task_contacts.contact_task_id = contact_tasks.id
			AND contacts.vault_id = ?
			AND contacts.listed = ?
			AND contacts.deleted_at IS NULL
	))`
}

func (s *BondsSearcher) searchReminders(vaultID, query, likeTerm string, limit int) ([]SearchItem, error) {
	db := s.db.Table("contact_reminders").
		Select("contact_reminders.id, contact_reminders.label, contact_reminders.contact_id").
		Joins("JOIN contacts ON contacts.id = contact_reminders.contact_id").
		Where("contacts.vault_id = ? AND contacts.listed = ?", vaultID, true)
	if normalizedNaturalQuery(query) == "today_reminders" {
		now := time.Now()
		db = db.Where("contact_reminders.day = ? AND contact_reminders.month = ?", now.Day(), int(now.Month()))
	} else {
		db = db.Where("LOWER(contact_reminders.label) LIKE ?", likeTerm)
	}
	type row struct {
		ID        uint
		Label     string
		ContactID string
	}
	var rows []row
	if err := db.Limit(limit).Scan(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]SearchItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, SearchItem{
			Type:        "reminder",
			ID:          fmt.Sprint(row.ID),
			Title:       row.Label,
			ResourceURI: "bonds://reminder/" + fmt.Sprint(row.ID),
			Reason:      "matched reminder fields or structured reminder filter",
		})
	}
	return items, nil
}

func (s *BondsSearcher) searchImportantDates(vaultID, query, likeTerm string, limit int) ([]SearchItem, error) {
	db := s.db.Table("contact_important_dates").
		Select("contact_important_dates.id, contact_important_dates.label, contact_important_dates.contact_id").
		Joins("JOIN contacts ON contacts.id = contact_important_dates.contact_id").
		Where("contacts.vault_id = ? AND contacts.listed = ?", vaultID, true)
	if normalizedNaturalQuery(query) == "next_month_birthdays" {
		next := time.Now().AddDate(0, 1, 0)
		db = db.Where("contact_important_dates.month = ?", int(next.Month())).
			Where("LOWER(contact_important_dates.label) LIKE ? OR contact_important_dates.contact_important_date_type_id IN (?)", "%birth%",
				s.db.Table("contact_important_date_types").Select("id").Where("vault_id = ? AND (LOWER(label) LIKE ? OR internal_type = ?)", vaultID, "%birth%", "birthdate"))
	} else {
		db = db.Where("LOWER(contact_important_dates.label) LIKE ?", likeTerm)
	}
	type row struct {
		ID        uint
		Label     string
		ContactID string
	}
	var rows []row
	if err := db.Limit(limit).Scan(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]SearchItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, SearchItem{
			Type:        "important_date",
			ID:          fmt.Sprint(row.ID),
			Title:       row.Label,
			ResourceURI: "bonds://important-date/" + fmt.Sprint(row.ID),
			Reason:      "matched important date fields or structured date filter",
		})
	}
	return items, nil
}

func (s *BondsSearcher) visibleBleveItems(vaultID string, items []SearchItem) ([]SearchItem, error) {
	visible := make([]SearchItem, 0, len(items))
	for _, item := range items {
		ok, err := s.isVisibleBleveItem(vaultID, item)
		if err != nil {
			return nil, err
		}
		if ok {
			visible = append(visible, item)
		}
	}
	return visible, nil
}

func (s *BondsSearcher) isVisibleBleveItem(vaultID string, item SearchItem) (bool, error) {
	var count int64
	switch item.Type {
	case "contact":
		err := s.db.Model(&models.Contact{}).Where("id = ? AND vault_id = ? AND listed = ?", item.ID, vaultID, true).Count(&count).Error
		return count > 0, err
	case "note":
		err := s.db.Model(&models.Note{}).
			Joins("JOIN contacts ON contacts.id = notes.contact_id").
			Where("notes.id = ? AND notes.vault_id = ? AND contacts.listed = ?", item.ID, vaultID, true).
			Count(&count).Error
		return count > 0, err
	default:
		return true, nil
	}
}

func normalizedNaturalQuery(query string) string {
	lower := strings.ToLower(query)
	switch {
	case strings.Contains(lower, "下个月") && (strings.Contains(lower, "生日") || strings.Contains(lower, "birth")):
		return "next_month_birthdays"
	case strings.Contains(lower, "逾期") || strings.Contains(lower, "overdue"):
		return "overdue_tasks"
	case strings.Contains(lower, "未完成") || strings.Contains(lower, "open task") || strings.Contains(lower, "todo"):
		return "open_tasks"
	case strings.Contains(lower, "今天") && (strings.Contains(lower, "提醒") || strings.Contains(lower, "reminder")):
		return "today_reminders"
	}
	return ""
}

func mergeSearchItems(primary, secondary []SearchItem) []SearchItem {
	seen := make(map[string]struct{}, len(primary)+len(secondary))
	result := make([]SearchItem, 0, len(primary)+len(secondary))
	for _, item := range append(primary, secondary...) {
		key := item.Type + ":" + item.ID
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, item)
	}
	return result
}

func contactName(contact *models.Contact) string {
	parts := []string{ptrString(contact.FirstName), ptrString(contact.MiddleName), ptrString(contact.LastName)}
	name := strings.TrimSpace(strings.Join(parts, " "))
	if name == "" {
		name = ptrString(contact.Nickname)
	}
	return name
}

func ptrString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

var _ VaultAccessChecker = (*services.VaultService)(nil)
