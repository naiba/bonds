package services

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/emersion/go-vcard"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var ErrVCardInvalidData = errors.New("invalid vcard data")

type VCardService struct {
	db *gorm.DB
}

func NewVCardService(db *gorm.DB) *VCardService {
	return &VCardService{db: db}
}

func (s *VCardService) ExportContactToVCard(contactID, vaultID string) (vcard.Card, error) {
	contact, err := s.loadContactForVCard(contactID, vaultID)
	if err != nil {
		return nil, err
	}
	return BuildContactVCard(contact), nil
}

func (s *VCardService) ExportContact(contactID string, vaultID string) ([]byte, error) {
	contact, err := s.loadContactForVCard(contactID, vaultID)
	if err != nil {
		return nil, err
	}
	card := BuildContactVCard(contact)

	var buf bytes.Buffer
	enc := vcard.NewEncoder(&buf)
	if err := enc.Encode(card); err != nil {
		return nil, fmt.Errorf("failed to encode vcard: %w", err)
	}
	return buf.Bytes(), nil
}

func (s *VCardService) ExportVault(vaultID string) ([]byte, error) {
	var contacts []models.Contact
	// Exclude shadow contacts (Listed=false) — they are UserVault self-contacts, not real contacts
	if err := preloadContactVCardRelations(s.db).Where("vault_id = ? AND listed = ?", vaultID, true).Find(&contacts).Error; err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	enc := vcard.NewEncoder(&buf)

	for _, contact := range contacts {
		card := BuildContactVCard(&contact)
		if err := enc.Encode(card); err != nil {
			return nil, fmt.Errorf("failed to encode vcard for contact %s: %w", contact.ID, err)
		}
	}

	return buf.Bytes(), nil
}

func (s *VCardService) loadContactForVCard(contactID, vaultID string) (*models.Contact, error) {
	var contact models.Contact
	if err := preloadContactVCardRelations(s.db).Where("id = ? AND vault_id = ?", contactID, vaultID).First(&contact).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrContactNotFound
		}
		return nil, err
	}
	return &contact, nil
}

func preloadContactVCardRelations(db *gorm.DB) *gorm.DB {
	return db.Preload("ContactInformations.ContactInformationType").
		Preload("Addresses").
		Preload("ImportantDates.ContactImportantDateType").
		Preload("Company").
		Preload("File")
}

func (s *VCardService) ImportVCard(vaultID, userID string, data io.Reader) (*dto.VCardImportResponse, error) {
	dec := vcard.NewDecoder(data)
	formatter, err := newContactNameFormatter(s.db, userID)
	if err != nil {
		return nil, err
	}

	var imported []dto.ContactResponse
	var skippedCount int
	var importErrors []string

	err = s.db.Transaction(func(tx *gorm.DB) error {
		var vault models.Vault
		if err := tx.First(&vault, "id = ?", vaultID).Error; err != nil {
			return fmt.Errorf("vault not found: %w", err)
		}
		accountID := vault.AccountID

		for {
			card, err := dec.Decode()
			if err == io.EOF {
				break
			}
			if err != nil {
				skippedCount++
				importErrors = append(importErrors, fmt.Sprintf("decode error: %v", err))
				continue
			}

			nameComponents := extractFullNameFromCard(card)
			nickname := card.Value(vcard.FieldNickname)

			title := card.Value(vcard.FieldTitle)

			now := time.Now()
			contact := models.Contact{
				VaultID:       vaultID,
				FirstName:     strPtrOrNil(nameComponents.firstName),
				LastName:      strPtrOrNil(nameComponents.lastName),
				MiddleName:    strPtrOrNil(nameComponents.middleName),
				Prefix:        strPtrOrNil(nameComponents.prefix),
				Suffix:        strPtrOrNil(nameComponents.suffix),
				Nickname:      strPtrOrNil(nickname),
				JobPosition:   strPtrOrNil(title),
				LastUpdatedAt: &now,
			}
			if err := tx.Create(&contact).Error; err != nil {
				return err
			}

			cvu := models.ContactVaultUser{
				ContactID: contact.ID,
				UserID:    userID,
				VaultID:   vaultID,
			}
			if err := tx.Create(&cvu).Error; err != nil {
				return err
			}

			if err := importVCardFields(tx, card, contact.ID, vaultID, accountID); err != nil {
				return err
			}
			if err := tx.Preload("FirstMetThrough", "vault_id = ?", vaultID).First(&contact, "id = ?", contact.ID).Error; err != nil {
				return err
			}

			resp, err := toContactResponse(&contact, false, formatter)
			if err != nil {
				return err
			}
			imported = append(imported, resp)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &dto.VCardImportResponse{
		ImportedCount: len(imported),
		SkippedCount:  skippedCount,
		Errors:        importErrors,
		Contacts:      imported,
	}, nil
}

// importVCardFields parses TEL, EMAIL, ADR, BDAY from a vCard and stores them.
func importVCardFields(tx *gorm.DB, card vcard.Card, contactID, vaultID, accountID string) error {
	// TEL → ContactInformation
	if fields := card[vcard.FieldTelephone]; len(fields) > 0 {
		var phoneType models.ContactInformationType
		if err := tx.Where("account_id = ? AND type = ?", accountID, "phone").First(&phoneType).Error; err == nil {
			for _, f := range fields {
				if f.Value == "" {
					continue
				}
				ci := models.ContactInformation{
					ContactID: contactID,
					TypeID:    phoneType.ID,
					Data:      f.Value,
				}
				if err := tx.Create(&ci).Error; err != nil {
					return err
				}
			}
		}
	}

	// EMAIL → ContactInformation
	if fields := card[vcard.FieldEmail]; len(fields) > 0 {
		var emailType models.ContactInformationType
		if err := tx.Where("account_id = ? AND type = ?", accountID, "email").First(&emailType).Error; err == nil {
			for _, f := range fields {
				if f.Value == "" {
					continue
				}
				ci := models.ContactInformation{
					ContactID: contactID,
					TypeID:    emailType.ID,
					Data:      f.Value,
				}
				if err := tx.Create(&ci).Error; err != nil {
					return err
				}
			}
		}
	}

	// ADR → Address + ContactAddress
	if addrs := card.Addresses(); len(addrs) > 0 {
		for _, addr := range addrs {
			if addr.StreetAddress == "" && addr.Locality == "" && addr.Region == "" && addr.PostalCode == "" && addr.Country == "" {
				continue
			}
			a := models.Address{
				VaultID:    vaultID,
				Line1:      strPtrOrNil(addr.StreetAddress),
				City:       strPtrOrNil(addr.Locality),
				Province:   strPtrOrNil(addr.Region),
				PostalCode: strPtrOrNil(addr.PostalCode),
				Country:    strPtrOrNil(addr.Country),
			}
			if err := tx.Create(&a).Error; err != nil {
				return err
			}
			ca := models.ContactAddress{
				ContactID: contactID,
				AddressID: a.ID,
			}
			if err := tx.Create(&ca).Error; err != nil {
				return err
			}
		}
	}

	// BDAY → ContactImportantDate
	if bday := card.Value(vcard.FieldBirthday); bday != "" {
		year, month, day := parseBirthdayString(bday)
		if month > 0 && day > 0 {
			var bdayType models.ContactImportantDateType
			if err := tx.Where("vault_id = ? AND internal_type = ?", vaultID, "birthdate").First(&bdayType).Error; err == nil {
				cid := models.ContactImportantDate{
					ContactID:                  contactID,
					ContactImportantDateTypeID: &bdayType.ID,
					Label:                      "Birthdate",
					Day:                        &day,
					Month:                      &month,
				}
				if year > 0 {
					cid.Year = &year
				}
				if err := tx.Create(&cid).Error; err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// parseBirthdayString parses vCard BDAY formats: "19900115", "1990-01-15", "--0115" (no year)
func parseBirthdayString(bday string) (year, month, day int) {
	bday = strings.TrimSpace(bday)
	// Try "YYYY-MM-DD"
	if t, err := time.Parse("2006-01-02", bday); err == nil {
		return t.Year(), int(t.Month()), t.Day()
	}
	// Try "YYYYMMDD"
	if len(bday) == 8 {
		if y, err := strconv.Atoi(bday[0:4]); err == nil {
			if m, err := strconv.Atoi(bday[4:6]); err == nil {
				if d, err := strconv.Atoi(bday[6:8]); err == nil {
					return y, m, d
				}
			}
		}
	}
	// Try "--MMDD" (no year)
	if strings.HasPrefix(bday, "--") && len(bday) >= 6 {
		s := bday[2:]
		if m, err := strconv.Atoi(s[0:2]); err == nil {
			if d, err := strconv.Atoi(s[2:4]); err == nil {
				return 0, m, d
			}
		}
	}
	return 0, 0, 0
}

// BuildContactVCard builds the canonical vCard 4.0 representation for a contact.
func BuildContactVCard(contact *models.Contact) vcard.Card {
	card := make(vcard.Card)
	card.SetValue(vcard.FieldVersion, "4.0")
	card.SetKind(vcard.KindIndividual)

	firstName := ptrToStr(contact.FirstName)
	lastName := ptrToStr(contact.LastName)
	middleName := ptrToStr(contact.MiddleName)
	prefix := ptrToStr(contact.Prefix)
	suffix := ptrToStr(contact.Suffix)

	card.SetName(&vcard.Name{
		FamilyName:      lastName,
		GivenName:       firstName,
		AdditionalName:  middleName,
		HonorificPrefix: prefix,
		HonorificSuffix: suffix,
	})
	formattedName := buildVCardFormattedName(prefix, firstName, middleName, lastName, suffix)
	if formattedName == "" {
		formattedName = "Unknown"
	}
	card.SetValue(vcard.FieldFormattedName, formattedName)

	if contact.Nickname != nil && *contact.Nickname != "" {
		card.SetValue(vcard.FieldNickname, *contact.Nickname)
	}
	if contact.JobPosition != nil && *contact.JobPosition != "" {
		card.SetValue(vcard.FieldTitle, *contact.JobPosition)
	}
	if contact.Company != nil && strings.TrimSpace(contact.Company.Name) != "" {
		card.SetValue(vcard.FieldOrganization, strings.TrimSpace(contact.Company.Name))
	}
	if contact.Description != nil && *contact.Description != "" {
		card.SetValue(vcard.FieldNote, *contact.Description)
	}
	if contact.ID != "" {
		card.SetValue(vcard.FieldUID, contact.ID)
	}
	if !contact.UpdatedAt.IsZero() {
		card.SetRevision(contact.UpdatedAt.UTC())
	}
	if birthday := contactBirthdayVCardValue(contact.ImportantDates); birthday != "" {
		card.SetValue(vcard.FieldBirthday, birthday)
	}
	if anniversary := contactAnniversaryVCardValue(contact.ImportantDates); anniversary != "" {
		card.SetValue(vcard.FieldAnniversary, anniversary)
	}
	if photoURL, mediaType := contactPhotoVCardValue(contact.File); photoURL != "" {
		params := vcard.Params{}
		if mediaType != "" {
			params.Set(vcard.ParamMediaType, mediaType)
		}
		card.Set(vcard.FieldPhoto, &vcard.Field{Value: photoURL, Params: params})
	}

	for _, info := range contact.ContactInformations {
		typeName := ptrToStr(info.ContactInformationType.Type)
		switch typeName {
		case "phone":
			params := contactInformationParams(&info, phoneKindToVCardTypes)
			if !paramHasType(params) {
				params.Add(vcard.ParamType, vcard.TypeVoice)
			}
			card.Add(vcard.FieldTelephone, &vcard.Field{
				Value:  info.Data,
				Params: params,
			})
		case "email":
			card.Add(vcard.FieldEmail, &vcard.Field{
				Value:  info.Data,
				Params: contactInformationParams(&info, emailKindToVCardTypes),
			})
		case "social":
			addSocialVCardFields(card, &info)
		}
	}

	for _, addr := range contact.Addresses {
		card.AddAddress(&vcard.Address{
			StreetAddress: ptrToStr(addr.Line1),
			Locality:      ptrToStr(addr.City),
			Region:        ptrToStr(addr.Province),
			PostalCode:    ptrToStr(addr.PostalCode),
			Country:       ptrToStr(addr.Country),
		})
	}

	return card
}

const vcardFieldSocialProfile = "X-SOCIALPROFILE"

func contactInformationParams(info *models.ContactInformation, kindMapper func(string) []string) vcard.Params {
	params := vcard.Params{}
	if info.Kind != nil && strings.TrimSpace(*info.Kind) != "" {
		kind := strings.ToLower(strings.TrimSpace(*info.Kind))
		types := []string{kind}
		if kindMapper != nil {
			if mapped := kindMapper(kind); len(mapped) > 0 {
				types = mapped
			}
		}
		for _, t := range types {
			params.Add(vcard.ParamType, t)
		}
	}
	if info.Pref {
		params.Set(vcard.ParamPreferred, "1")
	}
	return params
}

func paramHasType(params vcard.Params) bool {
	return len(params[vcard.ParamType]) > 0
}

// phoneKindToVCardTypes maps a user kind to RFC 6350 TEL TYPE values (e.g. mobile -> cell).
func phoneKindToVCardTypes(kind string) []string {
	switch kind {
	case "mobile", "cell", "cellphone", "cell phone":
		return []string{vcard.TypeCell, vcard.TypeVoice}
	case "home":
		return []string{vcard.TypeHome, vcard.TypeVoice}
	case "work", "office":
		return []string{vcard.TypeWork, vcard.TypeVoice}
	case "main":
		return []string{"main", vcard.TypeVoice}
	case "fax":
		return []string{vcard.TypeFax}
	case "home fax", "home_fax":
		return []string{vcard.TypeHome, vcard.TypeFax}
	case "work fax", "work_fax":
		return []string{vcard.TypeWork, vcard.TypeFax}
	case "pager":
		return []string{vcard.TypePager}
	case "voice":
		return []string{vcard.TypeVoice}
	default:
		return nil
	}
}

// emailKindToVCardTypes maps a user kind to RFC 6350 EMAIL TYPE values (home/work).
func emailKindToVCardTypes(kind string) []string {
	switch kind {
	case "home", "personal":
		return []string{vcard.TypeHome}
	case "work", "office":
		return []string{vcard.TypeWork}
	default:
		return nil
	}
}

func addSocialVCardFields(card vcard.Card, info *models.ContactInformation) {
	if strings.TrimSpace(info.Data) == "" {
		return
	}
	platform := contactInformationPlatform(info.ContactInformationType)
	params := contactInformationParams(info, nil)
	if platform != "" {
		params.Add(vcard.ParamType, platform)
	}

	card.Add(vcardFieldSocialProfile, &vcard.Field{Value: info.Data, Params: params})
	if isVCardURI(info.Data) {
		card.Add(vcard.FieldIMPP, &vcard.Field{Value: info.Data, Params: params})
	}
	if isWebURL(info.Data) {
		card.Add(vcard.FieldURL, &vcard.Field{Value: info.Data, Params: params})
	}
}

func contactInformationPlatform(infoType models.ContactInformationType) string {
	if infoType.NameTranslationKey != nil {
		parts := strings.Split(*infoType.NameTranslationKey, ".")
		return normalizeVCardToken(parts[len(parts)-1])
	}
	return normalizeVCardToken(ptrToStr(infoType.Name))
}

func normalizeVCardToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func isWebURL(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://")
}

func isVCardURI(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || strings.ContainsAny(value, " \t\r\n") {
		return false
	}
	colon := strings.Index(value, ":")
	return colon > 0
}

func contactPhotoVCardValue(file *models.File) (string, string) {
	if file == nil || file.ID == 0 {
		return "", ""
	}
	for _, candidate := range []string{ptrToStr(file.CdnURL), ptrToStr(file.OriginalURL)} {
		if isWebURL(candidate) || strings.HasPrefix(strings.ToLower(candidate), "data:") {
			return candidate, file.MimeType
		}
	}
	return "", ""
}

func contactBirthdayVCardValue(dates []models.ContactImportantDate) string {
	for _, date := range dates {
		if !isBirthdateImportantDate(&date) || date.Month == nil || date.Day == nil {
			continue
		}
		return formatImportantDateVCardValue(&date)
	}
	return ""
}

func contactAnniversaryVCardValue(dates []models.ContactImportantDate) string {
	for _, date := range dates {
		if !isAnniversaryImportantDate(&date) || date.Month == nil || date.Day == nil {
			continue
		}
		return formatImportantDateVCardValue(&date)
	}
	return ""
}

func formatImportantDateVCardValue(date *models.ContactImportantDate) string {
	if date.Year != nil {
		return fmt.Sprintf("%04d-%02d-%02d", *date.Year, *date.Month, *date.Day)
	}
	return fmt.Sprintf("--%02d-%02d", *date.Month, *date.Day)
}

func isBirthdateImportantDate(date *models.ContactImportantDate) bool {
	if date.ContactImportantDateType != nil && date.ContactImportantDateType.InternalType != nil {
		return *date.ContactImportantDateType.InternalType == "birthdate"
	}
	return strings.EqualFold(date.Label, "Birthdate") || strings.EqualFold(date.Label, "Birthday")
}

func isAnniversaryImportantDate(date *models.ContactImportantDate) bool {
	if date.ContactImportantDateType != nil && date.ContactImportantDateType.InternalType != nil {
		return *date.ContactImportantDateType.InternalType == "anniversary"
	}
	return strings.EqualFold(date.Label, "Anniversary")
}

func buildVCardFormattedName(prefix, firstName, middleName, lastName, suffix string) string {
	parts := []string{prefix, firstName, middleName, lastName, suffix}
	filled := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.TrimSpace(part) != "" {
			filled = append(filled, strings.TrimSpace(part))
		}
	}
	return strings.Join(filled, " ")
}

type vcardNameComponents struct {
	firstName  string
	lastName   string
	middleName string
	prefix     string
	suffix     string
}

func extractNameFromCard(card vcard.Card) (string, string) {
	c := extractFullNameFromCard(card)
	return c.firstName, c.lastName
}

func extractFullNameFromCard(card vcard.Card) vcardNameComponents {
	name := card.Name()
	if name != nil && (name.GivenName != "" || name.FamilyName != "") {
		return vcardNameComponents{
			firstName:  name.GivenName,
			lastName:   name.FamilyName,
			middleName: name.AdditionalName,
			prefix:     name.HonorificPrefix,
			suffix:     name.HonorificSuffix,
		}
	}
	fn := card.Value(vcard.FieldFormattedName)
	if fn != "" {
		return vcardNameComponents{firstName: fn}
	}
	return vcardNameComponents{}
}

func buildFullName(firstName, lastName string) string {
	if firstName != "" && lastName != "" {
		return firstName + " " + lastName
	}
	if firstName != "" {
		return firstName
	}
	return lastName
}

// UpsertContactFromVCard creates or updates a contact from a vCard.
// If distantURI is non-empty, it looks up an existing contact by DistantURI in the vault.
// lastSyncAt is used for conflict detection: if the contact was locally modified after lastSyncAt, the local version wins.
// Returns the contact ID and an action string: "created", "updated", "skipped", or "conflict_local_wins".
func (s *VCardService) UpsertContactFromVCard(tx *gorm.DB, card vcard.Card, vaultID, userID, accountID string, distantURI, distantEtag string, lastSyncAt *time.Time) (contactID string, action string, err error) {
	if distantURI != "" {
		var existing models.Contact
		if findErr := tx.Where("vault_id = ? AND distant_uri = ?", vaultID, distantURI).First(&existing).Error; findErr == nil {
			if existing.DistantEtag != nil && *existing.DistantEtag == distantEtag {
				return existing.ID, "skipped", nil
			}

			// Conflict detection: if contact was locally modified since last sync, local wins
			if lastSyncAt != nil && existing.LastUpdatedAt != nil && existing.LastUpdatedAt.After(*lastSyncAt) {
				return existing.ID, "conflict_local_wins", nil
			}

			nameComponents := extractFullNameFromCard(card)
			nickname := card.Value(vcard.FieldNickname)
			title := card.Value(vcard.FieldTitle)
			now := time.Now()

			existing.FirstName = strPtrOrNil(nameComponents.firstName)
			existing.LastName = strPtrOrNil(nameComponents.lastName)
			existing.MiddleName = strPtrOrNil(nameComponents.middleName)
			existing.Prefix = strPtrOrNil(nameComponents.prefix)
			existing.Suffix = strPtrOrNil(nameComponents.suffix)
			existing.Nickname = strPtrOrNil(nickname)
			existing.JobPosition = strPtrOrNil(title)
			existing.DistantEtag = strPtrOrNil(distantEtag)
			existing.LastUpdatedAt = &now
			if err := tx.Save(&existing).Error; err != nil {
				return "", "", err
			}

			if err := replaceVCardFields(tx, card, existing.ID, vaultID, accountID); err != nil {
				return "", "", err
			}
			return existing.ID, "updated", nil
		}
	}

	nameComponents := extractFullNameFromCard(card)
	nickname := card.Value(vcard.FieldNickname)
	title := card.Value(vcard.FieldTitle)
	now := time.Now()

	uid := card.Value("UID")

	contact := models.Contact{
		VaultID:       vaultID,
		FirstName:     strPtrOrNil(nameComponents.firstName),
		LastName:      strPtrOrNil(nameComponents.lastName),
		MiddleName:    strPtrOrNil(nameComponents.middleName),
		Prefix:        strPtrOrNil(nameComponents.prefix),
		Suffix:        strPtrOrNil(nameComponents.suffix),
		Nickname:      strPtrOrNil(nickname),
		JobPosition:   strPtrOrNil(title),
		DistantUUID:   strPtrOrNil(uid),
		DistantURI:    strPtrOrNil(distantURI),
		DistantEtag:   strPtrOrNil(distantEtag),
		LastUpdatedAt: &now,
	}
	if err := tx.Create(&contact).Error; err != nil {
		return "", "", err
	}

	cvu := models.ContactVaultUser{
		ContactID: contact.ID,
		UserID:    userID,
		VaultID:   vaultID,
	}
	if err := tx.Create(&cvu).Error; err != nil {
		return "", "", err
	}

	if err := importVCardFields(tx, card, contact.ID, vaultID, accountID); err != nil {
		return "", "", err
	}

	return contact.ID, "created", nil
}

func replaceVCardFields(tx *gorm.DB, card vcard.Card, contactID, vaultID, accountID string) error {
	tx.Where("contact_id = ?", contactID).Delete(&models.ContactInformation{})

	var pivots []models.ContactAddress
	tx.Where("contact_id = ?", contactID).Find(&pivots)
	if len(pivots) > 0 {
		addressIDs := make([]uint, len(pivots))
		for i, p := range pivots {
			addressIDs[i] = p.AddressID
		}
		tx.Where("contact_id = ?", contactID).Delete(&models.ContactAddress{})
		tx.Where("id IN ?", addressIDs).Delete(&models.Address{})
	}

	tx.Where("contact_id = ?", contactID).Delete(&models.ContactImportantDate{})

	return importVCardFields(tx, card, contactID, vaultID, accountID)
}
