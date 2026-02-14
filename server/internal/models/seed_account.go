package models

import (
	"time"

	"gorm.io/gorm"
)

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

func SeedAccountDefaults(tx *gorm.DB, accountID, userID, userEmail string) error {
	seeders := []func(*gorm.DB, string) error{
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
		seedAccountCurrencies,
	}
	for _, fn := range seeders {
		if err := fn(tx, accountID); err != nil {
			return err
		}
	}
	return seedNotificationChannel(tx, userID, userEmail)
}

func seedGenders(tx *gorm.DB, accountID string) error {
	items := []Gender{
		{AccountID: accountID, Name: strPtr("Male")},
		{AccountID: accountID, Name: strPtr("Female")},
		{AccountID: accountID, Name: strPtr("Other")},
	}
	return tx.Create(&items).Error
}

func seedPronouns(tx *gorm.DB, accountID string) error {
	items := []Pronoun{
		{AccountID: accountID, Name: strPtr("he/him")},
		{AccountID: accountID, Name: strPtr("she/her")},
		{AccountID: accountID, Name: strPtr("they/them")},
		{AccountID: accountID, Name: strPtr("per/per")},
		{AccountID: accountID, Name: strPtr("ve/ver")},
		{AccountID: accountID, Name: strPtr("xe/xem")},
		{AccountID: accountID, Name: strPtr("ze/hir")},
	}
	return tx.Create(&items).Error
}

func seedAddressTypes(tx *gorm.DB, accountID string) error {
	items := []AddressType{
		{AccountID: accountID, Name: strPtr("Home"), Type: strPtr("home")},
		{AccountID: accountID, Name: strPtr("Secondary residence"), Type: strPtr("secondary")},
		{AccountID: accountID, Name: strPtr("Work"), Type: strPtr("work")},
		{AccountID: accountID, Name: strPtr("Chalet"), Type: strPtr("chalet")},
		{AccountID: accountID, Name: strPtr("Other"), Type: strPtr("other")},
	}
	return tx.Create(&items).Error
}

func seedPetCategories(tx *gorm.DB, accountID string) error {
	items := []PetCategory{
		{AccountID: accountID, Name: strPtr("Dog")},
		{AccountID: accountID, Name: strPtr("Cat")},
		{AccountID: accountID, Name: strPtr("Bird")},
		{AccountID: accountID, Name: strPtr("Fish")},
		{AccountID: accountID, Name: strPtr("Small animal")},
		{AccountID: accountID, Name: strPtr("Hamster")},
		{AccountID: accountID, Name: strPtr("Horse")},
		{AccountID: accountID, Name: strPtr("Rabbit")},
		{AccountID: accountID, Name: strPtr("Rat")},
		{AccountID: accountID, Name: strPtr("Reptile")},
	}
	return tx.Create(&items).Error
}

func seedContactInfoTypes(tx *gorm.DB, accountID string) error {
	items := []ContactInformationType{
		{AccountID: accountID, Name: strPtr("Email address"), Type: strPtr("email"), Protocol: strPtr("mailto:")},
		{AccountID: accountID, Name: strPtr("Phone"), Type: strPtr("phone"), Protocol: strPtr("tel:")},
		{AccountID: accountID, Name: strPtr("Facebook"), Type: strPtr("social")},
		{AccountID: accountID, Name: strPtr("WhatsApp"), Type: strPtr("social")},
		{AccountID: accountID, Name: strPtr("Telegram"), Type: strPtr("social")},
		{AccountID: accountID, Name: strPtr("LinkedIn"), Type: strPtr("social")},
		{AccountID: accountID, Name: strPtr("Instagram"), Type: strPtr("social")},
		{AccountID: accountID, Name: strPtr("Twitter"), Type: strPtr("social")},
		{AccountID: accountID, Name: strPtr("Mastodon"), Type: strPtr("social")},
		{AccountID: accountID, Name: strPtr("Bluesky"), Type: strPtr("social")},
		{AccountID: accountID, Name: strPtr("Threads"), Type: strPtr("social")},
		{AccountID: accountID, Name: strPtr("GitHub"), Type: strPtr("social")},
	}
	if err := tx.Create(&items).Error; err != nil {
		return err
	}
	return tx.Model(&ContactInformationType{}).
		Where("account_id = ? AND type IN ?", accountID, []string{"email", "phone"}).
		Update("can_be_deleted", false).Error
}

func seedRelationshipGroupTypes(tx *gorm.DB, accountID string) error {
	type relType struct {
		name, reverse, typ string
		canBeDeleted       bool
	}
	type relGroup struct {
		name         string
		canBeDeleted bool
		types        []relType
	}

	groups := []relGroup{
		{
			name: "Love", canBeDeleted: false,
			types: []relType{
				{"significant other", "significant other", "love", false},
				{"spouse", "spouse", "love", false},
				{"date", "date", "", true},
				{"lover", "lover", "", true},
				{"in love with", "loved by", "", true},
				{"ex-boyfriend", "ex-boyfriend", "", true},
			},
		},
		{
			name: "Family", canBeDeleted: false,
			types: []relType{
				{"parent", "child", "child", false},
				{"brother/sister", "brother/sister", "", true},
				{"grand parent", "grand child", "", true},
				{"uncle/aunt", "nephew/niece", "", true},
				{"cousin", "cousin", "", true},
				{"godparent", "godchild", "", true},
			},
		},
		{
			name: "Friend", canBeDeleted: true,
			types: []relType{
				{"friend", "friend", "", true},
				{"best friend", "best friend", "", true},
			},
		},
		{
			name: "Work", canBeDeleted: true,
			types: []relType{
				{"colleague", "colleague", "", true},
				{"subordinate", "boss", "", true},
				{"mentor", "protege", "", true},
			},
		},
	}

	var undeletableGroupIDs []uint
	var undeletableTypeIDs []uint

	for _, g := range groups {
		group := RelationshipGroupType{
			AccountID: accountID,
			Name:      strPtr(g.name),
		}
		if err := tx.Create(&group).Error; err != nil {
			return err
		}
		if !g.canBeDeleted {
			undeletableGroupIDs = append(undeletableGroupIDs, group.ID)
		}
		for _, t := range g.types {
			rt := RelationshipType{
				RelationshipGroupTypeID: group.ID,
				Name:                    strPtr(t.name),
				NameReverseRelationship: strPtr(t.reverse),
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

func seedCallReasonTypes(tx *gorm.DB, accountID string) error {
	personal := CallReasonType{AccountID: accountID, Label: strPtr("Personal")}
	if err := tx.Create(&personal).Error; err != nil {
		return err
	}
	personalReasons := []CallReason{
		{CallReasonTypeID: personal.ID, Label: strPtr("For advice")},
		{CallReasonTypeID: personal.ID, Label: strPtr("Just to say hello")},
		{CallReasonTypeID: personal.ID, Label: strPtr("To see if they need anything")},
		{CallReasonTypeID: personal.ID, Label: strPtr("Out of respect and appreciation")},
		{CallReasonTypeID: personal.ID, Label: strPtr("To hear their story")},
	}
	if err := tx.Create(&personalReasons).Error; err != nil {
		return err
	}

	business := CallReasonType{AccountID: accountID, Label: strPtr("Business")}
	if err := tx.Create(&business).Error; err != nil {
		return err
	}
	businessReasons := []CallReason{
		{CallReasonTypeID: business.ID, Label: strPtr("Discuss recent purchases")},
		{CallReasonTypeID: business.ID, Label: strPtr("Discuss partnership")},
	}
	return tx.Create(&businessReasons).Error
}

func seedReligions(tx *gorm.DB, accountID string) error {
	names := []string{
		"Christian", "Muslim", "Hinduist", "Buddhist", "Shintoist",
		"Taoist", "Sikh", "Jew", "Atheist",
	}
	items := make([]Religion, len(names))
	for i, n := range names {
		pos := i + 1
		items[i] = Religion{AccountID: accountID, Name: strPtr(n), Position: &pos}
	}
	return tx.Create(&items).Error
}

func seedGroupTypes(tx *gorm.DB, accountID string) error {
	groups := []struct {
		label    string
		position int
		roles    []string
	}{
		{"Family", 1, []string{"Parent", "Child"}},
		{"Couple", 2, []string{"Partner"}},
		{"Club", 3, nil},
		{"Association", 4, nil},
		{"Roommates", 5, nil},
	}

	for _, g := range groups {
		gt := GroupType{
			AccountID: accountID,
			Label:     strPtr(g.label),
			Position:  g.position,
		}
		if err := tx.Create(&gt).Error; err != nil {
			return err
		}
		for i, r := range g.roles {
			pos := i + 1
			role := GroupTypeRole{
				GroupTypeID: gt.ID,
				Label:       strPtr(r),
				Position:    &pos,
			}
			if err := tx.Create(&role).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func seedEmotions(tx *gorm.DB, accountID string) error {
	items := []Emotion{
		{AccountID: accountID, Name: strPtr("Negative"), Type: "negative"},
		{AccountID: accountID, Name: strPtr("Neutral"), Type: "neutral"},
		{AccountID: accountID, Name: strPtr("Positive"), Type: "positive"},
	}
	return tx.Create(&items).Error
}

func seedGiftOccasions(tx *gorm.DB, accountID string) error {
	names := []string{"Birthday", "Anniversary", "Christmas", "Just because", "Wedding"}
	items := make([]GiftOccasion, len(names))
	for i, n := range names {
		pos := i + 1
		items[i] = GiftOccasion{AccountID: accountID, Label: strPtr(n), Position: &pos}
	}
	return tx.Create(&items).Error
}

func seedGiftStates(tx *gorm.DB, accountID string) error {
	names := []string{"Idea", "Searched", "Found", "Bought", "Offered"}
	items := make([]GiftState, len(names))
	for i, n := range names {
		pos := i + 1
		items[i] = GiftState{AccountID: accountID, Label: strPtr(n), Position: &pos}
	}
	return tx.Create(&items).Error
}

func seedPostTemplates(tx *gorm.DB, accountID string) error {
	regular := PostTemplate{AccountID: accountID, Label: strPtr("Regular post"), Position: 1}
	if err := tx.Create(&regular).Error; err != nil {
		return err
	}
	if err := tx.Model(&regular).Update("can_be_deleted", false).Error; err != nil {
		return err
	}
	inspirational := PostTemplate{AccountID: accountID, Label: strPtr("Inspirational post"), Position: 2}
	return tx.Create(&inspirational).Error
}

func seedDefaultTemplate(tx *gorm.DB, accountID string) error {
	tmpl := Template{AccountID: accountID, Name: strPtr("Default template")}
	if err := tx.Create(&tmpl).Error; err != nil {
		return err
	}
	if err := tx.Model(&tmpl).Update("can_be_deleted", false).Error; err != nil {
		return err
	}

	type pageDef struct {
		name     string
		slug     string
		position int
		typ      *string
		undel    bool
	}
	pages := []pageDef{
		{"Contact information", "contact", 1, strPtr("contact"), true},
		{"Feed", "feed", 2, nil, false},
		{"Social", "social", 3, nil, false},
		{"Life & goals", "life-goals", 4, nil, false},
		{"Information", "information", 5, nil, false},
	}

	for _, p := range pages {
		pos := p.position
		page := TemplatePage{
			TemplateID: tmpl.ID,
			Name:       strPtr(p.name),
			Slug:       p.slug,
			Position:   &pos,
			Type:       p.typ,
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

func seedNotificationChannel(tx *gorm.DB, userID, userEmail string) error {
	now := time.Now()
	channel := UserNotificationChannel{
		UserID:        &userID,
		Type:          "email",
		Label:         strPtr("Email address"),
		Content:       userEmail,
		PreferredTime: strPtr("09:00"),
		VerifiedAt:    &now,
	}
	if err := tx.Create(&channel).Error; err != nil {
		return err
	}
	return tx.Model(&channel).Update("active", true).Error
}

func seedAccountCurrencies(tx *gorm.DB, accountID string) error {
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
