package models

import (
	"time"

	"github.com/naiba/bonds/internal/i18n"
	"gorm.io/gorm"
)

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

func SeedAccountDefaults(tx *gorm.DB, accountID, userID, userEmail, locale string) error {
	seeders := []func(*gorm.DB, string, string) error{
		seedGenders,
		seedPronouns,
		seedAddressTypes,
		seedPetCategories,
		seedContactInfoTypes,
		seedRelationshipGroupTypes,
		seedCallReasonTypes,
		seedReligions,
		seedGroupTypes,
		seedEmotions,
		seedGiftOccasions,
		seedGiftStates,
		seedPostTemplates,
		seedDefaultTemplate,
		seedDefaultModules,
		seedAccountCurrencies,
	}
	for _, fn := range seeders {
		if err := fn(tx, accountID, locale); err != nil {
			return err
		}
	}
	return seedNotificationChannel(tx, userID, userEmail, locale)
}

func seedGenders(tx *gorm.DB, accountID, locale string) error {
	keys := []string{
		"seed.genders.male",
		"seed.genders.female",
		"seed.genders.other",
	}
	items := make([]Gender, len(keys))
	for idx, k := range keys {
		items[idx] = Gender{
			AccountID:          accountID,
			Name:               strPtr(i18n.T(locale, k)),
			NameTranslationKey: strPtr(k),
		}
	}
	return tx.Create(&items).Error
}

func seedPronouns(tx *gorm.DB, accountID, locale string) error {
	keys := []string{
		"seed.pronouns.he_him",
		"seed.pronouns.she_her",
		"seed.pronouns.they_them",
		"seed.pronouns.per_per",
		"seed.pronouns.ve_ver",
		"seed.pronouns.xe_xem",
		"seed.pronouns.ze_hir",
	}
	items := make([]Pronoun, len(keys))
	for idx, k := range keys {
		items[idx] = Pronoun{
			AccountID:          accountID,
			Name:               strPtr(i18n.T(locale, k)),
			NameTranslationKey: strPtr(k),
		}
	}
	return tx.Create(&items).Error
}

func seedAddressTypes(tx *gorm.DB, accountID, locale string) error {
	type addrDef struct {
		key string
		typ string
	}
	defs := []addrDef{
		{"seed.address_types.home", "home"},
		{"seed.address_types.secondary_residence", "secondary"},
		{"seed.address_types.work", "work"},
		{"seed.address_types.chalet", "chalet"},
		{"seed.address_types.other", "other"},
	}
	items := make([]AddressType, len(defs))
	for idx, d := range defs {
		items[idx] = AddressType{
			AccountID:          accountID,
			Name:               strPtr(i18n.T(locale, d.key)),
			Type:               strPtr(d.typ),
			NameTranslationKey: strPtr(d.key),
		}
	}
	return tx.Create(&items).Error
}

func seedPetCategories(tx *gorm.DB, accountID, locale string) error {
	keys := []string{
		"seed.pet_categories.dog",
		"seed.pet_categories.cat",
		"seed.pet_categories.bird",
		"seed.pet_categories.fish",
		"seed.pet_categories.small_animal",
		"seed.pet_categories.hamster",
		"seed.pet_categories.horse",
		"seed.pet_categories.rabbit",
		"seed.pet_categories.rat",
		"seed.pet_categories.reptile",
	}
	items := make([]PetCategory, len(keys))
	for idx, k := range keys {
		items[idx] = PetCategory{
			AccountID:          accountID,
			Name:               strPtr(i18n.T(locale, k)),
			NameTranslationKey: strPtr(k),
		}
	}
	return tx.Create(&items).Error
}

func seedContactInfoTypes(tx *gorm.DB, accountID, locale string) error {
	type infoDef struct {
		key      string
		typ      string
		protocol string
	}
	defs := []infoDef{
		{"seed.contact_info_types.email_address", "email", "mailto:"},
		{"seed.contact_info_types.phone", "phone", "tel:"},
		{"seed.contact_info_types.facebook", "social", ""},
		{"seed.contact_info_types.whatsapp", "social", ""},
		{"seed.contact_info_types.telegram", "social", ""},
		{"seed.contact_info_types.linkedin", "social", ""},
		{"seed.contact_info_types.instagram", "social", ""},
		{"seed.contact_info_types.twitter", "social", ""},
		{"seed.contact_info_types.mastodon", "social", ""},
		{"seed.contact_info_types.bluesky", "social", ""},
		{"seed.contact_info_types.threads", "social", ""},
		{"seed.contact_info_types.github", "social", ""},
	}
	items := make([]ContactInformationType, len(defs))
	for idx, d := range defs {
		items[idx] = ContactInformationType{
			AccountID:          accountID,
			Name:               strPtr(i18n.T(locale, d.key)),
			Type:               strPtr(d.typ),
			NameTranslationKey: strPtr(d.key),
		}
		if d.protocol != "" {
			items[idx].Protocol = strPtr(d.protocol)
		}
	}
	if err := tx.Create(&items).Error; err != nil {
		return err
	}
	return tx.Model(&ContactInformationType{}).
		Where("account_id = ? AND type IN ?", accountID, []string{"email", "phone"}).
		Update("can_be_deleted", false).Error
}

func seedRelationshipGroupTypes(tx *gorm.DB, accountID, locale string) error {
	type relType struct {
		nameKey, reverseKey, typ string
		canBeDeleted             bool
		degree                   *int
	}
	type relGroup struct {
		nameKey      string
		canBeDeleted bool
		types        []relType
	}

	groups := []relGroup{
		{
			nameKey: "seed.relationship_groups.love", canBeDeleted: false,
			types: []relType{
				{"seed.relationship_types.significant_other", "seed.relationship_types.significant_other", "love", false, nil},
				{"seed.relationship_types.spouse", "seed.relationship_types.spouse", "love", false, nil},
				{"seed.relationship_types.date", "seed.relationship_types.date", "", true, nil},
				{"seed.relationship_types.lover", "seed.relationship_types.lover", "", true, nil},
				{"seed.relationship_types.in_love_with", "seed.relationship_types.loved_by", "", true, nil},
				{"seed.relationship_types.ex_boyfriend", "seed.relationship_types.ex_boyfriend", "", true, nil},
			},
		},
		{
			nameKey: "seed.relationship_groups.family", canBeDeleted: false,
			types: []relType{
				{"seed.relationship_types.parent", "seed.relationship_types.child", "child", false, intPtr(1)},
				{"seed.relationship_types.brother_sister", "seed.relationship_types.brother_sister", "", true, intPtr(2)},
				{"seed.relationship_types.grand_parent", "seed.relationship_types.grand_child", "", true, intPtr(2)},
				{"seed.relationship_types.uncle_aunt", "seed.relationship_types.nephew_niece", "", true, intPtr(3)},
				{"seed.relationship_types.cousin", "seed.relationship_types.cousin", "", true, intPtr(4)},
				{"seed.relationship_types.godparent", "seed.relationship_types.godchild", "", true, nil},
			},
		},
		{
			nameKey: "seed.relationship_groups.friend", canBeDeleted: true,
			types: []relType{
				{"seed.relationship_types.friend", "seed.relationship_types.friend", "", true, nil},
				{"seed.relationship_types.best_friend", "seed.relationship_types.best_friend", "", true, nil},
			},
		},
		{
			nameKey: "seed.relationship_groups.work", canBeDeleted: true,
			types: []relType{
				{"seed.relationship_types.colleague", "seed.relationship_types.colleague", "", true, nil},
				{"seed.relationship_types.subordinate", "seed.relationship_types.boss", "", true, intPtr(1)},
				{"seed.relationship_types.mentor", "seed.relationship_types.protege", "", true, intPtr(1)},
			},
		},
	}

	var undeletableGroupIDs []uint
	var undeletableTypeIDs []uint

	for _, g := range groups {
		group := RelationshipGroupType{
			AccountID:          accountID,
			Name:               strPtr(i18n.T(locale, g.nameKey)),
			NameTranslationKey: strPtr(g.nameKey),
		}
		if err := tx.Create(&group).Error; err != nil {
			return err
		}
		if !g.canBeDeleted {
			undeletableGroupIDs = append(undeletableGroupIDs, group.ID)
		}
		for _, t := range g.types {
			rt := RelationshipType{
				RelationshipGroupTypeID:               group.ID,
				Name:                                  strPtr(i18n.T(locale, t.nameKey)),
				NameTranslationKey:                    strPtr(t.nameKey),
				NameReverseRelationship:               strPtr(i18n.T(locale, t.reverseKey)),
				NameReverseRelationshipTranslationKey: strPtr(t.reverseKey),
				Degree:                                t.degree,
			}
			if t.typ != "" {
				rt.Type = strPtr(t.typ)
			}
			if err := tx.Create(&rt).Error; err != nil {
				return err
			}
			if !t.canBeDeleted {
				undeletableTypeIDs = append(undeletableTypeIDs, rt.ID)
			}
		}
	}

	if len(undeletableGroupIDs) > 0 {
		if err := tx.Model(&RelationshipGroupType{}).Where("id IN ?", undeletableGroupIDs).Update("can_be_deleted", false).Error; err != nil {
			return err
		}
	}
	if len(undeletableTypeIDs) > 0 {
		if err := tx.Model(&RelationshipType{}).Where("id IN ?", undeletableTypeIDs).Update("can_be_deleted", false).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedCallReasonTypes(tx *gorm.DB, accountID, locale string) error {
	personalKey := "seed.call_reason_types.personal"
	personal := CallReasonType{
		AccountID:           accountID,
		Label:               strPtr(i18n.T(locale, personalKey)),
		LabelTranslationKey: strPtr(personalKey),
	}
	if err := tx.Create(&personal).Error; err != nil {
		return err
	}
	personalReasonKeys := []string{
		"seed.call_reasons.for_advice",
		"seed.call_reasons.just_to_say_hello",
		"seed.call_reasons.to_see_if_they_need_anything",
		"seed.call_reasons.out_of_respect_and_appreciation",
		"seed.call_reasons.to_hear_their_story",
	}
	personalReasons := make([]CallReason, len(personalReasonKeys))
	for idx, k := range personalReasonKeys {
		personalReasons[idx] = CallReason{
			CallReasonTypeID:    personal.ID,
			Label:               strPtr(i18n.T(locale, k)),
			LabelTranslationKey: strPtr(k),
		}
	}
	if err := tx.Create(&personalReasons).Error; err != nil {
		return err
	}

	businessKey := "seed.call_reason_types.business"
	business := CallReasonType{
		AccountID:           accountID,
		Label:               strPtr(i18n.T(locale, businessKey)),
		LabelTranslationKey: strPtr(businessKey),
	}
	if err := tx.Create(&business).Error; err != nil {
		return err
	}
	businessReasonKeys := []string{
		"seed.call_reasons.discuss_recent_purchases",
		"seed.call_reasons.discuss_partnership",
	}
	businessReasons := make([]CallReason, len(businessReasonKeys))
	for idx, k := range businessReasonKeys {
		businessReasons[idx] = CallReason{
			CallReasonTypeID:    business.ID,
			Label:               strPtr(i18n.T(locale, k)),
			LabelTranslationKey: strPtr(k),
		}
	}
	return tx.Create(&businessReasons).Error
}

func seedReligions(tx *gorm.DB, accountID, locale string) error {
	keys := []string{
		"seed.religions.christian",
		"seed.religions.muslim",
		"seed.religions.hinduist",
		"seed.religions.buddhist",
		"seed.religions.shintoist",
		"seed.religions.taoist",
		"seed.religions.sikh",
		"seed.religions.jew",
		"seed.religions.atheist",
	}
	items := make([]Religion, len(keys))
	for idx, k := range keys {
		pos := idx + 1
		items[idx] = Religion{
			AccountID:      accountID,
			Name:           strPtr(i18n.T(locale, k)),
			TranslationKey: strPtr(k),
			Position:       &pos,
		}
	}
	return tx.Create(&items).Error
}

func seedGroupTypes(tx *gorm.DB, accountID, locale string) error {
	type groupDef struct {
		key      string
		position int
		roles    []string
	}

	groups := []groupDef{
		{"seed.group_types.family", 1, []string{"seed.group_type_roles.parent", "seed.group_type_roles.child"}},
		{"seed.group_types.couple", 2, []string{"seed.group_type_roles.partner"}},
		{"seed.group_types.club", 3, nil},
		{"seed.group_types.association", 4, nil},
		{"seed.group_types.roommates", 5, nil},
	}

	for _, g := range groups {
		gt := GroupType{
			AccountID:           accountID,
			Label:               strPtr(i18n.T(locale, g.key)),
			LabelTranslationKey: strPtr(g.key),
			Position:            g.position,
		}
		if err := tx.Create(&gt).Error; err != nil {
			return err
		}
		for idx, rk := range g.roles {
			pos := idx + 1
			role := GroupTypeRole{
				GroupTypeID:         gt.ID,
				Label:               strPtr(i18n.T(locale, rk)),
				LabelTranslationKey: strPtr(rk),
				Position:            &pos,
			}
			if err := tx.Create(&role).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func seedEmotions(tx *gorm.DB, accountID, locale string) error {
	type emotionDef struct {
		key string
		typ string
	}
	defs := []emotionDef{
		{"seed.emotions.negative", "negative"},
		{"seed.emotions.neutral", "neutral"},
		{"seed.emotions.positive", "positive"},
	}
	items := make([]Emotion, len(defs))
	for idx, d := range defs {
		items[idx] = Emotion{
			AccountID:          accountID,
			Name:               strPtr(i18n.T(locale, d.key)),
			NameTranslationKey: strPtr(d.key),
			Type:               d.typ,
		}
	}
	return tx.Create(&items).Error
}

func seedGiftOccasions(tx *gorm.DB, accountID, locale string) error {
	keys := []string{
		"seed.gift_occasions.birthday",
		"seed.gift_occasions.anniversary",
		"seed.gift_occasions.christmas",
		"seed.gift_occasions.just_because",
		"seed.gift_occasions.wedding",
	}
	items := make([]GiftOccasion, len(keys))
	for idx, k := range keys {
		pos := idx + 1
		items[idx] = GiftOccasion{
			AccountID:           accountID,
			Label:               strPtr(i18n.T(locale, k)),
			LabelTranslationKey: strPtr(k),
			Position:            &pos,
		}
	}
	return tx.Create(&items).Error
}

func seedGiftStates(tx *gorm.DB, accountID, locale string) error {
	keys := []string{
		"seed.gift_states.idea",
		"seed.gift_states.searched",
		"seed.gift_states.found",
		"seed.gift_states.bought",
		"seed.gift_states.offered",
	}
	items := make([]GiftState, len(keys))
	for idx, k := range keys {
		pos := idx + 1
		items[idx] = GiftState{
			AccountID:           accountID,
			Label:               strPtr(i18n.T(locale, k)),
			LabelTranslationKey: strPtr(k),
			Position:            &pos,
		}
	}
	return tx.Create(&items).Error
}

func seedPostTemplates(tx *gorm.DB, accountID, locale string) error {
	regularKey := "seed.post_templates.regular_post"
	regular := PostTemplate{
		AccountID:           accountID,
		Label:               strPtr(i18n.T(locale, regularKey)),
		LabelTranslationKey: strPtr(regularKey),
		Position:            1,
	}
	if err := tx.Create(&regular).Error; err != nil {
		return err
	}
	if err := tx.Model(&regular).Update("can_be_deleted", false).Error; err != nil {
		return err
	}
	inspirationalKey := "seed.post_templates.inspirational_post"
	inspirational := PostTemplate{
		AccountID:           accountID,
		Label:               strPtr(i18n.T(locale, inspirationalKey)),
		LabelTranslationKey: strPtr(inspirationalKey),
		Position:            2,
	}
	return tx.Create(&inspirational).Error
}

func seedDefaultTemplate(tx *gorm.DB, accountID, locale string) error {
	tmplKey := "seed.templates.default_template"
	tmpl := Template{
		AccountID:          accountID,
		Name:               strPtr(i18n.T(locale, tmplKey)),
		NameTranslationKey: strPtr(tmplKey),
	}
	if err := tx.Create(&tmpl).Error; err != nil {
		return err
	}
	if err := tx.Model(&tmpl).Update("can_be_deleted", false).Error; err != nil {
		return err
	}

	type pageDef struct {
		key      string
		slug     string
		position int
		typ      *string
		undel    bool
	}
	pages := []pageDef{
		{"seed.template_pages.contact_information", "contact", 1, strPtr("contact"), true},
		{"seed.template_pages.feed", "feed", 2, nil, false},
		{"seed.template_pages.social", "social", 3, nil, false},
		{"seed.template_pages.life_and_goals", "life-goals", 4, nil, false},
		{"seed.template_pages.information", "information", 5, nil, false},
	}

	for _, p := range pages {
		pos := p.position
		page := TemplatePage{
			TemplateID:         tmpl.ID,
			Name:               strPtr(i18n.T(locale, p.key)),
			NameTranslationKey: strPtr(p.key),
			Slug:               p.slug,
			Position:           &pos,
			Type:               p.typ,
		}
		if err := tx.Create(&page).Error; err != nil {
			return err
		}
		if p.undel {
			if err := tx.Model(&page).Update("can_be_deleted", false).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func seedDefaultModules(tx *gorm.DB, accountID, locale string) error {
	var tmpl Template
	if err := tx.Where("account_id = ? AND can_be_deleted = ?", accountID, false).First(&tmpl).Error; err != nil {
		return err
	}

	var pages []TemplatePage
	if err := tx.Where("template_id = ?", tmpl.ID).Order("position ASC").Find(&pages).Error; err != nil {
		return err
	}

	pageBySlug := make(map[string]TemplatePage)
	for _, p := range pages {
		pageBySlug[p.Slug] = p
	}

	type moduleDef struct {
		key                          string
		typ                          string
		reservedToContactInformation bool
	}

	pageModules := map[string][]moduleDef{
		"contact": {
			{"seed.modules.avatar", "avatar", true},
			{"seed.modules.contact_name", "contact_names", true},
			{"seed.modules.family_summary", "family_summary", true},
			{"seed.modules.important_dates", "important_dates", true},
			{"seed.modules.gender_and_pronoun", "gender_pronoun", true},
			{"seed.modules.labels", "labels", true},
			{"seed.modules.job_information", "company", true},
			{"seed.modules.religions", "religions", true},
			{"seed.modules.addresses", "addresses", false},
			{"seed.modules.contact_information", "contact_information", false},
		},
		"feed": {
			{"seed.modules.contact_feed", "feed", false},
		},
		"social": {
			{"seed.modules.relationships", "relationships", false},
			{"seed.modules.pets", "pets", false},
			{"seed.modules.groups", "groups", false},
		},
		"life-goals": {
			{"seed.modules.life_events", "life_events", false},
			{"seed.modules.goals", "goals", false},
		},
		"information": {
			{"seed.modules.documents", "documents", false},
			{"seed.modules.photos", "photos", false},
			{"seed.modules.notes", "notes", false},
			{"seed.modules.reminders", "reminders", false},
			{"seed.modules.loans", "loans", false},
			{"seed.modules.tasks", "tasks", false},
			{"seed.modules.calls", "calls", false},
			{"seed.modules.posts", "posts", false},
		},
	}

	var undeletableModuleIDs []uint

	for _, slug := range []string{"contact", "feed", "social", "life-goals", "information"} {
		page, ok := pageBySlug[slug]
		if !ok {
			continue
		}
		defs := pageModules[slug]
		for idx, def := range defs {
			mod := Module{
				AccountID:                    accountID,
				Name:                         strPtr(i18n.T(locale, def.key)),
				NameTranslationKey:           strPtr(def.key),
				Type:                         strPtr(def.typ),
				ReservedToContactInformation: def.reservedToContactInformation,
			}
			if err := tx.Create(&mod).Error; err != nil {
				return err
			}
			undeletableModuleIDs = append(undeletableModuleIDs, mod.ID)

			pos := idx + 1
			pivot := ModuleTemplatePage{
				TemplatePageID: page.ID,
				ModuleID:       mod.ID,
				Position:       intPtr(pos),
			}
			if err := tx.Create(&pivot).Error; err != nil {
				return err
			}
		}
	}

	if len(undeletableModuleIDs) > 0 {
		if err := tx.Model(&Module{}).Where("id IN ?", undeletableModuleIDs).Update("can_be_deleted", false).Error; err != nil {
			return err
		}
	}

	return nil
}

func seedNotificationChannel(tx *gorm.DB, userID, userEmail, locale string) error {
	key := "seed.notification_channel.email_address"
	now := time.Now()
	channel := UserNotificationChannel{
		UserID:        &userID,
		Type:          "email",
		Label:         strPtr(i18n.T(locale, key)),
		Content:       userEmail,
		PreferredTime: strPtr("09:00"),
		VerifiedAt:    &now,
	}
	if err := tx.Create(&channel).Error; err != nil {
		return err
	}
	return tx.Model(&channel).Update("active", true).Error
}

func seedAccountCurrencies(tx *gorm.DB, accountID, _ string) error {
	var currencies []Currency
	if err := tx.Find(&currencies).Error; err != nil {
		return err
	}
	if len(currencies) == 0 {
		return nil
	}
	items := make([]AccountCurrency, len(currencies))
	for i, c := range currencies {
		items[i] = AccountCurrency{CurrencyID: c.ID, AccountID: accountID}
	}
	return tx.CreateInBatches(&items, 50).Error
}
