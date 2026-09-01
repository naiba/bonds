package services

import (
	"errors"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/markdown"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/utils"
	"github.com/naiba/bonds/pkg/response"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrActivityNotFound = errors.New("activity not found")
var ErrInvalidActivityTime = errors.New("invalid activity time")
var ErrInvalidActivityInput = errors.New("invalid activity input")
var ErrInvalidContentFormat = errors.New("invalid content format")

var contactMentionPattern = regexp.MustCompile(`@\[(?:\\[\\\]]|[^\]\r\n])+\]\(contact:([0-9a-fA-F-]{36})\)`)

type ActivityService struct {
	db           *gorm.DB
	feedRecorder *FeedRecorder
}

func NewActivityService(db *gorm.DB) *ActivityService       { return &ActivityService{db: db} }
func (s *ActivityService) SetFeedRecorder(fr *FeedRecorder) { s.feedRecorder = fr }

func (s *ActivityService) List(vaultID, contactID string, page, perPage int) ([]dto.ActivityResponse, response.Meta, error) {
	return s.ListForUser(vaultID, "", contactID, page, perPage)
}

func (s *ActivityService) ListForUser(vaultID, userID, contactID string, page, perPage int) ([]dto.ActivityResponse, response.Meta, error) {
	if contactID != "" {
		if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
			return nil, response.Meta{}, err
		}
	}
	query := activityListQuery(s.db, vaultID, contactID)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, response.Meta{}, err
	}
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 15
	}
	var events []models.Activity
	if err := query.Preload("Participants").Preload("ActivityType.ActivityCategory").
		Order("start_date IS NULL, start_date DESC, activities.id DESC").
		Offset((page - 1) * perPage).Limit(perPage).Find(&events).Error; err != nil {
		return nil, response.Meta{}, err
	}
	items := make([]dto.ActivityResponse, len(events))
	for i := range events {
		item, err := s.toActivityResponse(&events[i], userID)
		if err != nil {
			return nil, response.Meta{}, err
		}
		items[i] = item
	}
	return items, response.Meta{Page: page, PerPage: perPage, Total: total, TotalPages: int(math.Ceil(float64(total) / float64(perPage)))}, nil
}

func activityListQuery(db *gorm.DB, vaultID, contactID string) *gorm.DB {
	query := db.Model(&models.Activity{}).Where("activities.vault_id = ?", vaultID)
	if contactID == "" {
		return query
	}
	return query.Where(
		"EXISTS (SELECT 1 FROM activity_participants WHERE activity_participants.activity_id = activities.id AND activity_participants.contact_id = ?)",
		contactID,
	)
}

// Get returns one activity scoped to its vault. Authorization is enforced by
// the vault middleware; retaining the vault predicate here prevents ID-based
// cross-vault reads at the data boundary.
func (s *ActivityService) Get(vaultID, userID string, id uint) (*dto.ActivityResponse, error) {
	return s.get(vaultID, id, userID)
}

func (s *ActivityService) Create(vaultID string, req dto.ActivityUpsertRequest) (*dto.ActivityResponse, error) {
	return s.CreateForUser(vaultID, "", req)
}

func (s *ActivityService) CreateForUser(vaultID, userID string, req dto.ActivityUpsertRequest) (*dto.ActivityResponse, error) {
	isUserSubject := strings.TrimSpace(req.PrimaryContactID) == ""
	event, contactIDs, err := s.eventFromRequest(vaultID, req, nil, isUserSubject && userID != "")
	if err != nil {
		return nil, err
	}
	if isUserSubject {
		if userID == "" {
			return nil, ErrInvalidActivityInput
		}
		subjectName, err := s.currentUserSubjectName(vaultID, userID)
		if err != nil {
			return nil, err
		}
		event.SubjectUserID = &userID
		event.SubjectUserName = &subjectName
	}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&event).Error; err != nil {
			return err
		}
		if err := syncContentFileReferences(tx, vaultID, models.ContentOwnerActivity, event.ID, req.Description, event.DescriptionFormat); err != nil {
			return err
		}
		if err := replaceActivityParticipants(tx, event.ID, contactIDs); err != nil {
			return err
		}
		return updateInteractionLastTalkedTo(tx, event.ActivityTypeID, event.StartDate, contactIDs)
	}); err != nil {
		return nil, err
	}
	if s.feedRecorder != nil && req.PrimaryContactID != "" {
		entityType := "Activity"
		s.feedRecorder.Record(req.PrimaryContactID, "", ActionActivityCreated, "Created an activity", &event.ID, &entityType)
	}
	return s.get(vaultID, event.ID, userID)
}

func (s *ActivityService) Update(vaultID string, id uint, req dto.ActivityUpsertRequest) (*dto.ActivityResponse, error) {
	return s.UpdateForUser(vaultID, "", id, req)
}

func (s *ActivityService) UpdateForUser(vaultID, userID string, id uint, req dto.ActivityUpsertRequest) (*dto.ActivityResponse, error) {
	var current models.Activity
	if err := s.db.Preload("Participants").Where("id = ? AND vault_id = ?", id, vaultID).First(&current).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrActivityNotFound
		}
		return nil, err
	}
	currentParticipantIDs := make([]string, len(current.Participants))
	for i := range current.Participants {
		currentParticipantIDs[i] = current.Participants[i].ID
	}
	replacement, contactIDs, err := s.eventFromRequest(vaultID, req, currentParticipantIDs, current.SubjectUserID != nil)
	if err != nil {
		return nil, err
	}
	replacement.ID, replacement.CreatedAt = current.ID, current.CreatedAt
	if req.DescriptionFormat == "" {
		replacement.DescriptionFormat = markdown.NormalizeFormat(current.DescriptionFormat)
	}
	replacement.SubjectUserID = current.SubjectUserID
	replacement.SubjectUserName = current.SubjectUserName
	if replacement.ParentID != nil && *replacement.ParentID == id {
		return nil, ErrInvalidActivityTime
	}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&replacement).Error; err != nil {
			return err
		}
		if err := syncContentFileReferences(tx, vaultID, models.ContentOwnerActivity, id, req.Description, replacement.DescriptionFormat); err != nil {
			return err
		}
		if err := replaceActivityParticipantsLocked(tx, id, contactIDs); err != nil {
			return err
		}
		return updateInteractionLastTalkedTo(tx, replacement.ActivityTypeID, replacement.StartDate, contactIDs)
	}); err != nil {
		return nil, err
	}
	return s.get(vaultID, id, userID)
}

func (s *ActivityService) Delete(vaultID string, id uint) error {
	var event models.Activity
	if err := s.db.Where("id = ? AND vault_id = ?", id, vaultID).First(&event).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrActivityNotFound
		}
		return err
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Activity{}).Where("parent_id = ?", id).Update("parent_id", nil).Error; err != nil {
			return err
		}
		if err := tx.Where("activity_id = ?", id).Delete(&models.ActivityParticipant{}).Error; err != nil {
			return err
		}
		if err := tx.Where("vault_id = ? AND owner_type = ? AND owner_id = ?", vaultID, models.ContentOwnerActivity, id).
			Delete(&models.ContentFileReference{}).Error; err != nil {
			return err
		}
		return tx.Delete(&event).Error
	})
}

func (s *ActivityService) get(vaultID string, id uint, userID string) (*dto.ActivityResponse, error) {
	var event models.Activity
	if err := s.db.Preload("Participants").Preload("ActivityType.ActivityCategory").Where("id = ? AND vault_id = ?", id, vaultID).First(&event).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrActivityNotFound
		}
		return nil, err
	}
	resp, err := s.toActivityResponse(&event, userID)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *ActivityService) eventFromRequest(vaultID string, req dto.ActivityUpsertRequest, existingParticipantIDs []string, allowEmptyParticipants bool) (models.Activity, []string, error) {
	if strings.TrimSpace(req.Title) == "" || req.ActivityTypeID == 0 {
		return models.Activity{}, nil, ErrInvalidActivityInput
	}
	if !markdown.IsValidFormat(req.DescriptionFormat) {
		return models.Activity{}, nil, ErrInvalidContentFormat
	}
	if !validPrecision(req.StartPrecision, true) || !validEndStatus(req.EndStatus) ||
		(normalizedEndStatus(req.EndStatus) == "known" && !validPrecision(req.EndPrecision, true)) ||
		(req.CalendarType != "" && req.CalendarType != "gregorian" && req.CalendarType != "lunar") {
		return models.Activity{}, nil, ErrInvalidActivityInput
	}
	if err := validateActivityTypeBelongsToVault(s.db, req.ActivityTypeID, vaultID); err != nil {
		return models.Activity{}, nil, err
	}
	if err := validateActivityTime(req); err != nil {
		return models.Activity{}, nil, err
	}
	if req.ParentID != nil {
		var count int64
		if err := s.db.Model(&models.Activity{}).Where("id = ? AND vault_id = ? AND parent_id IS NULL", *req.ParentID, vaultID).Count(&count).Error; err != nil {
			return models.Activity{}, nil, err
		}
		if count != 1 {
			return models.Activity{}, nil, ErrActivityNotFound
		}
	}
	var contactIDs []string
	if req.ParticipantIDs != nil {
		contactIDs = mergeContactIDs([]string{req.PrimaryContactID}, *req.ParticipantIDs)
	} else if len(existingParticipantIDs) > 0 {
		// Older API clients do not send participant_ids. Preserve associations on
		// update rather than allowing an unrelated description edit to remove them.
		contactIDs = mergeContactIDs([]string{req.PrimaryContactID}, existingParticipantIDs)
	} else {
		// Creation fallback for clients generated before participant_ids existed.
		contactIDs = mergeContactIDs([]string{req.PrimaryContactID}, contactMentionIDs(req.Description))
	}
	if len(contactIDs) == 0 && !allowEmptyParticipants {
		return models.Activity{}, nil, ErrContactNotFound
	}
	if len(contactIDs) > 0 {
		if err := validateContactsBelongToVault(s.db, contactIDs, vaultID); err != nil {
			return models.Activity{}, nil, err
		}
	}
	typeID := req.ActivityTypeID
	event := models.Activity{
		VaultID: vaultID, ParentID: req.ParentID, ActivityTypeID: &typeID, Title: strings.TrimSpace(req.Title),
		Description: strPtrOrNil(req.Description), StartDate: req.StartDate, StartPrecision: normalizedPrecision(req.StartPrecision),
		DescriptionFormat: markdown.NormalizeFormat(req.DescriptionFormat),
		EndDate:           req.EndDate, EndPrecision: normalizedPrecision(req.EndPrecision), EndStatus: normalizedEndStatus(req.EndStatus),
		CalendarType: req.CalendarType, OriginalDay: req.OriginalDay, OriginalMonth: req.OriginalMonth, OriginalYear: req.OriginalYear,
		EmotionID: req.EmotionID, Costs: req.Costs, CurrencyID: req.CurrencyID, DurationInMinutes: req.DurationInMinutes,
		Distance: req.Distance, DistanceUnit: strPtrOrNil(req.DistanceUnit), FromPlace: strPtrOrNil(req.FromPlace),
		ToPlace: strPtrOrNil(req.ToPlace), Place: strPtrOrNil(req.Place),
	}
	if event.CalendarType == "" {
		event.CalendarType = "gregorian"
	}
	return event, contactIDs, nil
}

func validateActivityTime(req dto.ActivityUpsertRequest) error {
	status := normalizedEndStatus(req.EndStatus)
	if status == "known" {
		if req.StartDate == nil || req.EndDate == nil || req.EndDate.Before(*req.StartDate) {
			return ErrInvalidActivityTime
		}
	} else if req.EndDate != nil {
		return ErrInvalidActivityTime
	}
	return nil
}

func normalizedPrecision(value string) string {
	switch value {
	case "day", "month", "year":
		return value
	default:
		return "day"
	}
}
func validPrecision(value string, allowEmpty bool) bool {
	return (allowEmpty && value == "") || value == "day" || value == "month" || value == "year"
}
func validEndStatus(value string) bool {
	return value == "" || value == "none" || value == "known" || value == "ongoing" || value == "unknown"
}
func normalizedEndStatus(value string) string {
	switch value {
	case "known", "ongoing", "unknown":
		return value
	default:
		return "none"
	}
}

func contactMentionIDs(content string) []string {
	ids := make([]string, 0)
	for _, match := range contactMentionPattern.FindAllStringSubmatch(content, -1) {
		if len(match) == 2 {
			ids = append(ids, strings.ToLower(match[1]))
		}
	}
	return dedupeContactIDs(ids)
}

func validateActivityTypeBelongsToVault(db *gorm.DB, id uint, vaultID string) error {
	var count int64
	if err := db.Model(&models.ActivityType{}).Joins("JOIN activity_categories ON activity_categories.id = activity_types.activity_category_id").
		Where("activity_types.id = ? AND activity_categories.vault_id = ?", id, vaultID).Count(&count).Error; err != nil {
		return err
	}
	if count != 1 {
		return ErrActivityNotFound
	}
	return nil
}

func (s *ActivityService) toActivityResponse(le *models.Activity, currentUserID string) (dto.ActivityResponse, error) {
	resp := dto.ActivityResponse{ID: le.ID, VaultID: le.VaultID, SubjectUserID: ptrToStr(le.SubjectUserID),
		SubjectUserName: ptrToStr(le.SubjectUserName), SubjectIsCurrentUser: le.SubjectUserID != nil && *le.SubjectUserID == currentUserID,
		ParentID: le.ParentID, ActivityTypeID: le.ActivityTypeID,
		Title: le.Title, Description: ptrToStr(le.Description), DescriptionFormat: markdown.NormalizeFormat(le.DescriptionFormat),
		RenderedDescription: markdown.Render(ptrToStr(le.Description), le.DescriptionFormat), StartDate: le.StartDate, StartPrecision: le.StartPrecision,
		EndDate: le.EndDate, EndPrecision: le.EndPrecision, EndStatus: le.EndStatus, CalendarType: le.CalendarType,
		OriginalDay: le.OriginalDay, OriginalMonth: le.OriginalMonth, OriginalYear: le.OriginalYear, EmotionID: le.EmotionID,
		Costs: le.Costs, CurrencyID: le.CurrencyID, DurationInMinutes: le.DurationInMinutes, Distance: le.Distance,
		DistanceUnit: ptrToStr(le.DistanceUnit), FromPlace: ptrToStr(le.FromPlace), ToPlace: ptrToStr(le.ToPlace),
		Place: ptrToStr(le.Place), Participants: contactRefs(le.Participants), CreatedAt: le.CreatedAt, UpdatedAt: le.UpdatedAt}
	if le.ActivityType != nil {
		resp.ActivityType = &dto.ActivityTypeResponse{ID: le.ActivityType.ID, CategoryID: le.ActivityType.ActivityCategoryID,
			Label: ptrToStr(le.ActivityType.Label), CanBeDeleted: le.ActivityType.CanBeDeleted, Position: le.ActivityType.Position,
			SystemKind: ptrToStr(le.ActivityType.SystemKind), Icon: ptrToStr(le.ActivityType.Icon), Color: ptrToStr(le.ActivityType.Color),
			CountsAsInteraction: le.ActivityType.CountsAsInteraction, CreatedAt: le.ActivityType.CreatedAt, UpdatedAt: le.ActivityType.UpdatedAt}
	}
	mentionIDs := contactMentionIDs(ptrToStr(le.Description))
	if len(mentionIDs) > 0 {
		var mentioned []models.Contact
		if err := s.db.Where("vault_id = ? AND id IN ?", le.VaultID, mentionIDs).Find(&mentioned).Error; err != nil {
			return dto.ActivityResponse{}, err
		}
		resp.MentionedContacts = contactRefs(mentioned)
	}
	for i := range le.Milestones {
		milestone, err := s.toActivityResponse(&le.Milestones[i], currentUserID)
		if err != nil {
			return dto.ActivityResponse{}, err
		}
		resp.Milestones = append(resp.Milestones, milestone)
	}
	return resp, nil
}

func (s *ActivityService) currentUserSubjectName(vaultID, userID string) (string, error) {
	var user models.User
	if err := s.db.Model(&models.User{}).
		Joins("JOIN user_vault ON user_vault.user_id = users.id").
		Where("users.id = ? AND user_vault.vault_id = ?", userID, vaultID).
		First(&user).Error; err != nil {
		return "", err
	}
	name := strings.TrimSpace(strings.Join([]string{ptrToStr(user.FirstName), ptrToStr(user.LastName)}, " "))
	if name == "" {
		name = user.Email
	}
	return name, nil
}

func updateInteractionLastTalkedTo(tx *gorm.DB, typeID *uint, happenedAt *time.Time, contactIDs []string) error {
	if typeID == nil || happenedAt == nil || len(contactIDs) == 0 {
		return nil
	}
	var eventType models.ActivityType
	if err := tx.Select("counts_as_interaction").First(&eventType, *typeID).Error; err != nil {
		return err
	}
	if !eventType.CountsAsInteraction {
		return nil
	}
	return tx.Model(&models.Contact{}).Where("id IN ? AND (last_talked_to IS NULL OR last_talked_to < ?)", contactIDs, *happenedAt).
		Update("last_talked_to", *happenedAt).Error
}

func replaceActivityParticipants(tx *gorm.DB, activityID uint, contactIDs []string) error {
	if err := tx.Where("activity_id = ?", activityID).Delete(&models.ActivityParticipant{}).Error; err != nil {
		return err
	}
	ids := dedupeContactIDs(contactIDs)
	if len(ids) == 0 {
		return nil
	}
	rows := make([]models.ActivityParticipant, 0, len(ids))
	for _, id := range ids {
		rows = append(rows, models.ActivityParticipant{ContactID: id, ActivityID: activityID})
	}
	return tx.Create(&rows).Error
}

func replaceActivityParticipantsLocked(tx *gorm.DB, activityID uint, contactIDs []string) error {
	var target models.Activity
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").Where("id = ?", activityID).First(&target).Error; err != nil {
		return err
	}
	return replaceActivityParticipants(tx, activityID, contactIDs)
}

func mergeContactIDs(required, additional []string) []string {
	return dedupeContactIDs(append(append([]string{}, required...), additional...))
}
func dedupeContactIDs(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}

func contactRefs(contacts []models.Contact) []dto.TaskContactRef {
	refs := make([]dto.TaskContactRef, 0, len(contacts))
	for i := range contacts {
		refs = append(refs, dto.TaskContactRef{
			ID:           contacts[i].ID,
			Name:         utils.FormatContactName("%first_name% %last_name%", &contacts[i], contacts[i].ID),
			FirstName:    ptrToStr(contacts[i].FirstName),
			LastName:     ptrToStr(contacts[i].LastName),
			Nickname:     ptrToStr(contacts[i].Nickname),
			JobPosition:  ptrToStr(contacts[i].JobPosition),
			LastTalkedTo: contacts[i].LastTalkedTo,
		})
	}
	sort.Slice(refs, func(i, j int) bool {
		if refs[i].Name == refs[j].Name {
			return refs[i].ID < refs[j].ID
		}
		return refs[i].Name < refs[j].Name
	})
	return refs
}
