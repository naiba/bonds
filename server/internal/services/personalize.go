package services

import (
	"errors"
	"fmt"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/i18n"
	"gorm.io/gorm"
)

var ErrPersonalizeEntityNotFound = errors.New("entity not found")
var ErrUnknownEntityType = errors.New("unknown entity type")

type PersonalizeService struct {
	db *gorm.DB
}

func NewPersonalizeService(db *gorm.DB) *PersonalizeService {
	return &PersonalizeService{db: db}
}

type entityConfig struct {
	table    string
	hasLabel bool
	hasName  bool
}

var entityConfigs = map[string]entityConfig{
	"genders":            {table: "genders", hasName: true},
	"pronouns":           {table: "pronouns", hasName: true},
	"address-types":      {table: "address_types", hasName: true},
	"pet-categories":     {table: "pet_categories", hasName: true},
	"contact-info-types": {table: "contact_information_types", hasName: true},
	"call-reasons":       {table: "call_reason_types", hasLabel: true},
	"religions":          {table: "religions", hasName: true},
	"gift-occasions":     {table: "gift_occasions", hasLabel: true},
	"gift-states":        {table: "gift_states", hasLabel: true},
	"group-types":        {table: "group_types", hasLabel: true},
	"post-templates":     {table: "post_templates", hasLabel: true},
	"relationship-types": {table: "relationship_group_types", hasName: true},
	"templates":          {table: "templates", hasName: true},
	"modules":            {table: "modules", hasName: true},
	"currencies":         {table: "currencies"},
	"emotions":           {table: "emotions"},
}

func (s *PersonalizeService) List(accountID, entity string) ([]dto.PersonalizeEntityResponse, error) {
	cfg, ok := entityConfigs[entity]
	if !ok {
		return nil, ErrUnknownEntityType
	}

	var results []dto.PersonalizeEntityResponse
	labelCol := s.getLabelCol(cfg)

	query := fmt.Sprintf(
		"SELECT id, COALESCE(%s, '') as label, COALESCE(%s, '') as name, created_at, updated_at FROM %s WHERE account_id = ? ORDER BY id ASC",
		labelCol, s.getNameCol(cfg), cfg.table,
	)

	if entity == "currencies" {
		query = fmt.Sprintf(
			"SELECT c.id, c.code as label, c.code as name, c.created_at, c.updated_at FROM %s c JOIN account_currency ac ON ac.currency_id = c.id WHERE ac.account_id = ? ORDER BY c.code ASC",
			cfg.table,
		)
	}

	if entity == "emotions" {
		query = fmt.Sprintf(
			"SELECT id, COALESCE(name, '') as label, type as name, created_at, updated_at FROM %s WHERE account_id = ? ORDER BY id ASC",
			cfg.table,
		)
	}

	if err := s.db.Raw(query, accountID).Scan(&results).Error; err != nil {
		return nil, err
	}
	if results == nil {
		results = []dto.PersonalizeEntityResponse{}
	}
	return results, nil
}

func (s *PersonalizeService) Create(accountID, entity string, req dto.PersonalizeEntityRequest) (*dto.PersonalizeEntityResponse, error) {
	cfg, ok := entityConfigs[entity]
	if !ok {
		return nil, ErrUnknownEntityType
	}

	labelCol := s.getLabelCol(cfg)
	nameCol := s.getNameCol(cfg)
	val := req.Label
	if val == "" {
		val = req.Name
	}

	result := s.db.Exec(
		fmt.Sprintf("INSERT INTO %s (account_id, %s, %s, created_at, updated_at) VALUES (?, ?, ?, NOW(), NOW())", cfg.table, labelCol, nameCol),
		accountID, val, val,
	)
	if result.Error != nil {
		return nil, result.Error
	}

	var resp dto.PersonalizeEntityResponse
	s.db.Raw(fmt.Sprintf("SELECT id, COALESCE(%s, '') as label, COALESCE(%s, '') as name, created_at, updated_at FROM %s WHERE account_id = ? ORDER BY id DESC LIMIT 1", labelCol, nameCol, cfg.table), accountID).Scan(&resp)
	return &resp, nil
}

func (s *PersonalizeService) Update(accountID, entity string, id uint, req dto.PersonalizeEntityRequest) (*dto.PersonalizeEntityResponse, error) {
	cfg, ok := entityConfigs[entity]
	if !ok {
		return nil, ErrUnknownEntityType
	}

	labelCol := s.getLabelCol(cfg)
	nameCol := s.getNameCol(cfg)
	val := req.Label
	if val == "" {
		val = req.Name
	}

	result := s.db.Exec(
		fmt.Sprintf("UPDATE %s SET %s = ?, %s = ?, updated_at = NOW() WHERE id = ? AND account_id = ?", cfg.table, labelCol, nameCol),
		val, val, id, accountID,
	)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, ErrPersonalizeEntityNotFound
	}

	var resp dto.PersonalizeEntityResponse
	s.db.Raw(fmt.Sprintf("SELECT id, COALESCE(%s, '') as label, COALESCE(%s, '') as name, created_at, updated_at FROM %s WHERE id = ?", labelCol, nameCol, cfg.table), id).Scan(&resp)
	return &resp, nil
}

func (s *PersonalizeService) Delete(accountID, entity string, id uint) error {
	cfg, ok := entityConfigs[entity]
	if !ok {
		return ErrUnknownEntityType
	}

	result := s.db.Exec(
		fmt.Sprintf("DELETE FROM %s WHERE id = ? AND account_id = ?", cfg.table),
		id, accountID,
	)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrPersonalizeEntityNotFound
	}
	return nil
}

func (s *PersonalizeService) getLabelCol(cfg entityConfig) string {
	if cfg.hasLabel {
		return "label"
	}
	return "name"
}

func (s *PersonalizeService) getNameCol(cfg entityConfig) string {
	if cfg.hasName {
		return "name"
	}
	return "label"
}

func (s *PersonalizeService) ListTemplates(accountID string) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	if err := s.db.Table("templates").Where("account_id = ?", accountID).Order("id ASC").Find(&results).Error; err != nil {
		return nil, err
	}
	return results, nil
}

type syncableEntity struct {
	table         string
	displayCol    string
	keyCol        string
	ownerCol      string
	parentTable   string
	parentJoinCol string
}

var accountSyncEntities = []syncableEntity{
	{table: "genders", displayCol: "name", keyCol: "name_translation_key", ownerCol: "account_id"},
	{table: "pronouns", displayCol: "name", keyCol: "name_translation_key", ownerCol: "account_id"},
	{table: "address_types", displayCol: "name", keyCol: "name_translation_key", ownerCol: "account_id"},
	{table: "pet_categories", displayCol: "name", keyCol: "name_translation_key", ownerCol: "account_id"},
	{table: "contact_information_types", displayCol: "name", keyCol: "name_translation_key", ownerCol: "account_id"},
	{table: "relationship_group_types", displayCol: "name", keyCol: "name_translation_key", ownerCol: "account_id"},
	{table: "relationship_types", displayCol: "name", keyCol: "name_translation_key", ownerCol: "account_id", parentTable: "relationship_group_types", parentJoinCol: "relationship_group_type_id"},
	{table: "relationship_types", displayCol: "name_reverse_relationship", keyCol: "name_reverse_relationship_translation_key", ownerCol: "account_id", parentTable: "relationship_group_types", parentJoinCol: "relationship_group_type_id"},
	{table: "call_reason_types", displayCol: "label", keyCol: "label_translation_key", ownerCol: "account_id"},
	{table: "call_reasons", displayCol: "label", keyCol: "label_translation_key", ownerCol: "account_id", parentTable: "call_reason_types", parentJoinCol: "call_reason_type_id"},
	{table: "religions", displayCol: "name", keyCol: "translation_key", ownerCol: "account_id"},
	{table: "group_types", displayCol: "label", keyCol: "label_translation_key", ownerCol: "account_id"},
	{table: "group_type_roles", displayCol: "label", keyCol: "label_translation_key", ownerCol: "account_id", parentTable: "group_types", parentJoinCol: "group_type_id"},
	{table: "emotions", displayCol: "name", keyCol: "name_translation_key", ownerCol: "account_id"},
	{table: "gift_occasions", displayCol: "label", keyCol: "label_translation_key", ownerCol: "account_id"},
	{table: "gift_states", displayCol: "label", keyCol: "label_translation_key", ownerCol: "account_id"},
	{table: "post_templates", displayCol: "label", keyCol: "label_translation_key", ownerCol: "account_id"},
	{table: "templates", displayCol: "name", keyCol: "name_translation_key", ownerCol: "account_id"},
	{table: "template_pages", displayCol: "name", keyCol: "name_translation_key", ownerCol: "account_id", parentTable: "templates", parentJoinCol: "template_id"},
	{table: "modules", displayCol: "name", keyCol: "name_translation_key", ownerCol: "account_id"},
}

var vaultSyncEntities = []syncableEntity{
	{table: "mood_tracking_parameters", displayCol: "label", keyCol: "label_translation_key", ownerCol: "vault_id"},
	{table: "life_event_categories", displayCol: "label", keyCol: "label_translation_key", ownerCol: "vault_id"},
	{table: "life_event_types", displayCol: "label", keyCol: "label_translation_key", ownerCol: "vault_id", parentTable: "life_event_categories", parentJoinCol: "life_event_category_id"},
	{table: "vault_quick_facts_templates", displayCol: "label", keyCol: "label_translation_key", ownerCol: "vault_id"},
}

func (s *PersonalizeService) SyncAllTranslations(accountID, locale string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := syncEntities(tx, accountSyncEntities, accountID, locale); err != nil {
			return err
		}

		var vaultIDs []string
		if err := tx.Table("vaults").Where("account_id = ?", accountID).Pluck("id", &vaultIDs).Error; err != nil {
			return err
		}
		for _, vaultID := range vaultIDs {
			if err := syncEntities(tx, vaultSyncEntities, vaultID, locale); err != nil {
				return err
			}
		}
		return nil
	})
}

func syncEntities(tx *gorm.DB, entities []syncableEntity, ownerID, locale string) error {
	for _, e := range entities {
		var rows []struct {
			ID  uint   `gorm:"column:id"`
			Key string `gorm:"column:key"`
		}

		query := tx.Table(e.table).Select("id, " + e.keyCol + " AS key").
			Where(e.keyCol + " IS NOT NULL AND " + e.keyCol + " != ''")

		if e.parentTable != "" {
			query = query.Where(
				fmt.Sprintf("%s IN (SELECT id FROM %s WHERE %s = ?)", e.parentJoinCol, e.parentTable, e.ownerCol),
				ownerID,
			)
		} else {
			query = query.Where(e.ownerCol+" = ?", ownerID)
		}

		if err := query.Find(&rows).Error; err != nil {
			return err
		}

		for _, row := range rows {
			translated := i18n.T(locale, row.Key)
			if err := tx.Table(e.table).Where("id = ?", row.ID).
				Update(e.displayCol, translated).Error; err != nil {
				return err
			}
		}
	}
	return nil
}
