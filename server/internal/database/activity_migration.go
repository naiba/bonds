package database

import (
	"fmt"
	"strings"
	"time"

	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

type legacyTimelineRow struct {
	ID        uint
	VaultID   string
	StartedAt time.Time
	Label     *string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type legacyActivityRow struct {
	ID                uint
	TimelineEventID   uint
	ActivityTypeID    uint `gorm:"column:life_event_type_id"`
	EmotionID         *uint
	HappenedAt        time.Time
	CalendarType      string
	OriginalDay       *int
	OriginalMonth     *int
	OriginalYear      *int
	Summary           *string
	Description       *string
	Costs             *int
	CurrencyID        *uint
	PaidByContactID   *string
	DurationInMinutes *int
	Distance          *int
	DistanceUnit      *string
	FromPlace         *string
	ToPlace           *string
	Place             *string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type legacyParticipantRow struct {
	ContactID string
	EntityID  uint
}

// migrateActivitySchema upgrades every historical activity schema to the
// current Activity domain. It supports both the oldest TimelineEvent-backed
// schema and the standalone legacy schema shipped immediately before this rename.
// Renames preserve IDs and timestamps, so API links, feed references and
// imported source identities remain stable.
func migrateActivitySchema(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := renameActivityTaxonomyTables(tx); err != nil {
			return err
		}
		if tx.Migrator().HasTable("life_events") && hasColumn(tx, "life_events", "timeline_event_id") {
			if err := migrateLegacyTimelineActivities(tx); err != nil {
				return err
			}
		} else if err := renameActivityRecordTables(tx); err != nil {
			return err
		}
		if err := renameDomainColumn(tx, "vaults", "default_activity_tab", "default_dashboard_tab"); err != nil {
			return err
		}
		if err := cleanupLegacyActivityIndexes(tx); err != nil {
			return err
		}
		return migrateActivityMetadata(tx)
	})
}

func cleanupLegacyActivityIndexes(tx *gorm.DB) error {
	for _, tableName := range []string{"activity_categories", "activity_types", "activities", "activity_participants"} {
		if !tx.Migrator().HasTable(tableName) {
			continue
		}
		indexes, err := tx.Migrator().GetIndexes(tableName)
		if err != nil {
			return fmt.Errorf("migrate activities: inspect indexes on %s: %w", tableName, err)
		}
		for _, index := range indexes {
			if !strings.Contains(index.Name(), "life_event") {
				continue
			}
			if err := dropLegacyActivityIndex(tx, tableName, index.Name()); err != nil {
				return fmt.Errorf("migrate activities: drop legacy index %s: %w", index.Name(), err)
			}
		}
	}
	return nil
}

// dropLegacyActivityIndex avoids postgres.Migrator.DropIndex when the table
// does not have an explicitly qualified schema. postgres v1.6.2 renders the
// fallback schema expression as an identifier prefix, producing invalid SQL:
//
//	DROP INDEX CURRENT_SCHEMA()."index_name"
//
// Resolve the schema first and quote both catalog-provided identifiers. Keep
// the regular migrator path for SQLite and any future supported dialects.
func dropLegacyActivityIndex(tx *gorm.DB, tableName, indexName string) error {
	if tx.Dialector.Name() != "postgres" {
		return tx.Migrator().DropIndex(tableName, indexName)
	}

	var schemaName string
	if err := tx.Raw("SELECT current_schema()").Scan(&schemaName).Error; err != nil {
		return fmt.Errorf("resolve current schema: %w", err)
	}
	if schemaName == "" {
		return fmt.Errorf("resolve current schema: empty schema name")
	}
	return tx.Exec("DROP INDEX IF EXISTS " + quoteIdentifier(schemaName) + "." + quoteIdentifier(indexName)).Error
}

func migrateActivityMetadata(tx *gorm.DB) error {
	updates := []struct {
		table, column, oldValue, newValue string
	}{
		{"modules", "type", "life_events", "activities"},
		{"modules", "name_translation_key", "seed.modules.life_events", "seed.modules.activities"},
		{"template_pages", "name_translation_key", "seed.template_pages.life_and_goals", "seed.template_pages.activities"},
		{"contact_feed_items", "action", "life_event_created", "activity_created"},
		{"contact_feed_items", "feedable_type", "LifeEvent", "Activity"},
		{"contact_feed_items", "description", "Created a life event", "Created an activity"},
	}
	for _, update := range updates {
		if !tx.Migrator().HasTable(update.table) || !hasColumn(tx, update.table, update.column) {
			continue
		}
		if err := tx.Table(update.table).Where(update.column+" = ?", update.oldValue).Update(update.column, update.newValue).Error; err != nil {
			return fmt.Errorf("migrate activities: update %s.%s: %w", update.table, update.column, err)
		}
	}

	translationTables := []struct {
		table, column, oldPrefix, newPrefix string
	}{
		{"activity_categories", "label_translation_key", "seed.life_event_categories.", "seed.activity_categories."},
		{"activity_types", "label_translation_key", "seed.life_event_types.", "seed.activity_types."},
	}
	for _, update := range translationTables {
		if !tx.Migrator().HasTable(update.table) || !hasColumn(tx, update.table, update.column) {
			continue
		}
		if err := tx.Exec(
			"UPDATE "+quoteIdentifier(update.table)+" SET "+quoteIdentifier(update.column)+" = ? || SUBSTR("+quoteIdentifier(update.column)+", ?) WHERE "+quoteIdentifier(update.column)+" LIKE ?",
			update.newPrefix, len(update.oldPrefix)+1, update.oldPrefix+"%",
		).Error; err != nil {
			return fmt.Errorf("migrate activities: update %s translation keys: %w", update.table, err)
		}
	}

	if tx.Migrator().HasTable("vaults") && hasColumn(tx, "vaults", "default_dashboard_tab") {
		if err := tx.Table("vaults").Where("default_dashboard_tab = ?", "activity").Update("default_dashboard_tab", "feed").Error; err != nil {
			return fmt.Errorf("migrate activities: update dashboard feed tab: %w", err)
		}
		if err := tx.Table("vaults").Where("default_dashboard_tab = ?", "life_events").Update("default_dashboard_tab", "activities").Error; err != nil {
			return fmt.Errorf("migrate activities: update dashboard activities tab: %w", err)
		}
		if err := tx.Table("vaults").
			Where("default_dashboard_tab IS NULL OR default_dashboard_tab NOT IN ?", []string{"feed", "activities", "life_metrics"}).
			Update("default_dashboard_tab", "feed").Error; err != nil {
			return fmt.Errorf("migrate activities: normalize dashboard tab: %w", err)
		}
	}
	return nil
}

func renameActivityTaxonomyTables(tx *gorm.DB) error {
	if err := renameDomainTable(tx, "life_event_categories", "activity_categories"); err != nil {
		return err
	}
	if err := renameDomainTable(tx, "life_event_types", "activity_types"); err != nil {
		return err
	}
	return renameDomainColumn(tx, "activity_types", "life_event_category_id", "activity_category_id")
}

func renameActivityRecordTables(tx *gorm.DB) error {
	if err := renameDomainTable(tx, "life_events", "activities"); err != nil {
		return err
	}
	if err := renameDomainColumn(tx, "activities", "life_event_type_id", "activity_type_id"); err != nil {
		return err
	}
	if err := renameDomainTable(tx, "life_event_participants", "activity_participants"); err != nil {
		return err
	}
	return renameDomainColumn(tx, "activity_participants", "life_event_id", "activity_id")
}

func renameDomainTable(tx *gorm.DB, oldName, newName string) error {
	oldExists := tx.Migrator().HasTable(oldName)
	newExists := tx.Migrator().HasTable(newName)
	if oldExists && newExists {
		return fmt.Errorf("migrate activities: both %s and %s exist", oldName, newName)
	}
	if !oldExists {
		return nil
	}
	if err := tx.Migrator().RenameTable(oldName, newName); err != nil {
		return fmt.Errorf("migrate activities: rename %s to %s: %w", oldName, newName, err)
	}
	return nil
}

func renameDomainColumn(tx *gorm.DB, tableName, oldName, newName string) error {
	if !tx.Migrator().HasTable(tableName) || !hasColumn(tx, tableName, oldName) {
		return nil
	}
	if hasColumn(tx, tableName, newName) {
		return fmt.Errorf("migrate activities: both %s.%s and %s.%s exist", tableName, oldName, tableName, newName)
	}
	if err := tx.Migrator().RenameColumn(tableName, oldName, newName); err != nil {
		return fmt.Errorf("migrate activities: rename %s.%s to %s: %w", tableName, oldName, newName, err)
	}
	return nil
}

// migrateLegacyTimelineActivities flattens the original TimelineEvent
// container into the new activities table. The caller has already renamed the
// category/type taxonomy and owns the surrounding transaction.
func migrateLegacyTimelineActivities(tx *gorm.DB) error {
	if !tx.Migrator().HasTable("life_events") || !hasColumn(tx, "life_events", "timeline_event_id") {
		return nil
	}
	if tx.Migrator().HasTable("activities") {
		return fmt.Errorf("migrate legacy activities: activities table already exists")
	}
	const oldEvents = "legacy_activities_v1"
	const oldParticipants = "legacy_life_event_participants_v1"
	if tx.Migrator().HasTable(oldEvents) || tx.Migrator().HasTable(oldParticipants) {
		return fmt.Errorf("migrate legacy activities: stale migration tables exist")
	}
	var timelines []legacyTimelineRow
	if err := tx.Table("timeline_events").Find(&timelines).Error; err != nil {
		return fmt.Errorf("read timelines: %w", err)
	}
	var oldRows []legacyActivityRow
	if err := tx.Table("life_events").Find(&oldRows).Error; err != nil {
		return fmt.Errorf("read activities: %w", err)
	}
	var oldLifeParticipants []legacyParticipantRow
	if tx.Migrator().HasTable("life_event_participants") {
		if err := tx.Table("life_event_participants").Select("contact_id, life_event_id AS entity_id").Scan(&oldLifeParticipants).Error; err != nil {
			return err
		}
	}
	var oldTimelineParticipants []legacyParticipantRow
	if tx.Migrator().HasTable("timeline_event_participants") {
		if err := tx.Table("timeline_event_participants").Select("contact_id, timeline_event_id AS entity_id").Scan(&oldTimelineParticipants).Error; err != nil {
			return err
		}
	}

	if err := tx.Migrator().RenameTable("life_events", oldEvents); err != nil {
		return fmt.Errorf("rename activities: %w", err)
	}
	if tx.Migrator().HasTable("life_event_participants") {
		if err := tx.Migrator().RenameTable("life_event_participants", oldParticipants); err != nil {
			return fmt.Errorf("rename participants: %w", err)
		}
	}
	if err := tx.Migrator().CreateTable(&models.Activity{}); err != nil {
		return fmt.Errorf("create activities: %w", err)
	}
	if err := tx.Migrator().CreateTable(&models.ActivityParticipant{}); err != nil {
		return fmt.Errorf("create participants: %w", err)
	}

	timelineByID := make(map[uint]legacyTimelineRow, len(timelines))
	childrenByTimeline := make(map[uint][]uint)
	var nextEventID uint = 1
	for _, timeline := range timelines {
		timelineByID[timeline.ID] = timeline
	}
	for _, row := range oldRows {
		childrenByTimeline[row.TimelineEventID] = append(childrenByTimeline[row.TimelineEventID], row.ID)
		if row.ID >= nextEventID {
			nextEventID = row.ID + 1
		}
	}
	parentByTimeline := make(map[uint]uint)

	for _, row := range oldRows {
		timeline, ok := timelineByID[row.TimelineEventID]
		if !ok {
			return fmt.Errorf("activity %d references missing timeline %d", row.ID, row.TimelineEventID)
		}
		typeID := row.ActivityTypeID
		title := strings.TrimSpace(stringValue(row.Summary))
		if title == "" {
			title = strings.TrimSpace(stringValue(timeline.Label))
		}
		if title == "" {
			title = "Activity"
		}
		calendarType := row.CalendarType
		if calendarType == "" {
			calendarType = "gregorian"
		}
		start := row.HappenedAt
		event := models.Activity{ID: row.ID, VaultID: timeline.VaultID, ActivityTypeID: &typeID, EmotionID: row.EmotionID,
			StartDate: &start, StartPrecision: "day", EndStatus: "none", CalendarType: calendarType,
			OriginalDay: row.OriginalDay, OriginalMonth: row.OriginalMonth, OriginalYear: row.OriginalYear,
			Title: title, Description: row.Description, Costs: row.Costs, CurrencyID: row.CurrencyID,
			PaidByContactID: row.PaidByContactID, DurationInMinutes: row.DurationInMinutes, Distance: row.Distance,
			DistanceUnit: row.DistanceUnit, FromPlace: row.FromPlace, ToPlace: row.ToPlace, Place: row.Place,
			CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
		if err := tx.Create(&event).Error; err != nil {
			return fmt.Errorf("copy activity %d: %w", row.ID, err)
		}
	}

	for _, timeline := range timelines {
		children := childrenByTimeline[timeline.ID]
		if len(children) == 1 {
			continue
		}
		title := strings.TrimSpace(stringValue(timeline.Label))
		if title == "" {
			title = "Activities"
		}
		start := timeline.StartedAt
		parent := models.Activity{ID: nextEventID, VaultID: timeline.VaultID, Title: title, StartDate: &start, StartPrecision: "day", EndStatus: "unknown", CalendarType: "gregorian", CreatedAt: timeline.CreatedAt, UpdatedAt: timeline.UpdatedAt}
		if err := tx.Create(&parent).Error; err != nil {
			return fmt.Errorf("create parent for timeline %d: %w", timeline.ID, err)
		}
		nextEventID++
		parentByTimeline[timeline.ID] = parent.ID
		if len(children) > 0 {
			if err := tx.Model(&models.Activity{}).Where("id IN ?", children).Update("parent_id", parent.ID).Error; err != nil {
				return err
			}
		}
	}

	lifeContacts := make(map[uint][]string)
	for _, row := range oldLifeParticipants {
		lifeContacts[row.EntityID] = append(lifeContacts[row.EntityID], row.ContactID)
	}
	timelineContacts := make(map[uint][]string)
	for _, row := range oldTimelineParticipants {
		timelineContacts[row.EntityID] = append(timelineContacts[row.EntityID], row.ContactID)
	}
	for _, row := range oldRows {
		ids := uniqueStrings(append(lifeContacts[row.ID], timelineContacts[row.TimelineEventID]...))
		if err := createMigratedParticipants(tx, row.ID, ids); err != nil {
			return err
		}
	}
	for timelineID, parentID := range parentByTimeline {
		if err := createMigratedParticipants(tx, parentID, uniqueStrings(timelineContacts[timelineID])); err != nil {
			return err
		}
	}
	if tx.Dialector.Name() == "postgres" {
		if err := tx.Exec("SELECT setval(pg_get_serial_sequence('activities', 'id'), COALESCE((SELECT MAX(id) FROM activities), 1), true)").Error; err != nil {
			return fmt.Errorf("reset activity sequence: %w", err)
		}
	}

	for _, table := range []string{oldParticipants, oldEvents, "timeline_event_participants", "timeline_events"} {
		if tx.Migrator().HasTable(table) {
			if err := tx.Migrator().DropTable(table); err != nil {
				return fmt.Errorf("drop %s: %w", table, err)
			}
		}
	}
	return nil
}

func createMigratedParticipants(tx *gorm.DB, eventID uint, ids []string) error {
	for _, id := range ids {
		if err := tx.Create(&models.ActivityParticipant{ActivityID: eventID, ContactID: id}).Error; err != nil {
			return fmt.Errorf("copy participant for event %d: %w", eventID, err)
		}
	}
	return nil
}
func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
