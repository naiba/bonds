package dav

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/emersion/go-vcard"
	"github.com/emersion/go-webdav"
	"github.com/emersion/go-webdav/carddav"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

// CardDAVBackend implements the carddav.Backend interface.
type CardDAVBackend struct {
	db *gorm.DB
}

// NewCardDAVBackend creates a new CardDAV backend.
func NewCardDAVBackend(db *gorm.DB) *CardDAVBackend {
	return &CardDAVBackend{db: db}
}

func (b *CardDAVBackend) CurrentUserPrincipal(ctx context.Context) (string, error) {
	userID := UserIDFromContext(ctx)
	if userID == "" {
		return "", fmt.Errorf("no user in context")
	}
	return "/dav/principals/" + userID + "/", nil
}

func (b *CardDAVBackend) AddressBookHomeSetPath(ctx context.Context) (string, error) {
	userID := UserIDFromContext(ctx)
	if userID == "" {
		return "", fmt.Errorf("no user in context")
	}
	return "/dav/addressbooks/" + userID + "/", nil
}

func (b *CardDAVBackend) ListAddressBooks(ctx context.Context) ([]carddav.AddressBook, error) {
	userID := UserIDFromContext(ctx)
	if userID == "" {
		return nil, fmt.Errorf("no user in context")
	}

	var userVaults []models.UserVault
	if err := b.db.Where("user_id = ?", userID).Find(&userVaults).Error; err != nil {
		return nil, err
	}

	var books []carddav.AddressBook
	for _, uv := range userVaults {
		var vault models.Vault
		if err := b.db.First(&vault, "id = ?", uv.VaultID).Error; err != nil {
			continue
		}
		books = append(books, carddav.AddressBook{
			Path:        "/dav/addressbooks/" + userID + "/" + vault.ID + "/",
			Name:        vault.Name,
			Description: ptrToStr(vault.Description),
			SupportedAddressData: []carddav.AddressDataType{
				{ContentType: "text/vcard", Version: "3.0"},
				{ContentType: "text/vcard", Version: "4.0"},
			},
		})
	}
	return books, nil
}

func (b *CardDAVBackend) GetAddressBook(ctx context.Context, path string) (*carddav.AddressBook, error) {
	userID := UserIDFromContext(ctx)
	if userID == "" {
		return nil, fmt.Errorf("no user in context")
	}

	vaultID := extractVaultIDFromPath(path, "addressbooks", userID)
	if vaultID == "" {
		return nil, webdav.NewHTTPError(http.StatusNotFound, fmt.Errorf("address book not found"))
	}

	// Verify user has access
	var uv models.UserVault
	if err := b.db.Where("user_id = ? AND vault_id = ?", userID, vaultID).First(&uv).Error; err != nil {
		return nil, webdav.NewHTTPError(http.StatusNotFound, fmt.Errorf("address book not found"))
	}

	var vault models.Vault
	if err := b.db.First(&vault, "id = ?", vaultID).Error; err != nil {
		return nil, webdav.NewHTTPError(http.StatusNotFound, fmt.Errorf("address book not found"))
	}

	return &carddav.AddressBook{
		Path:        "/dav/addressbooks/" + userID + "/" + vault.ID + "/",
		Name:        vault.Name,
		Description: ptrToStr(vault.Description),
		SupportedAddressData: []carddav.AddressDataType{
			{ContentType: "text/vcard", Version: "3.0"},
			{ContentType: "text/vcard", Version: "4.0"},
		},
	}, nil
}

func (b *CardDAVBackend) CreateAddressBook(_ context.Context, _ *carddav.AddressBook) error {
	return webdav.NewHTTPError(http.StatusForbidden, fmt.Errorf("creating address books is not supported"))
}

func (b *CardDAVBackend) DeleteAddressBook(_ context.Context, _ string) error {
	return webdav.NewHTTPError(http.StatusForbidden, fmt.Errorf("deleting address books is not supported"))
}

func (b *CardDAVBackend) GetAddressObject(ctx context.Context, path string, _ *carddav.AddressDataRequest) (*carddav.AddressObject, error) {
	userID := UserIDFromContext(ctx)
	if userID == "" {
		return nil, fmt.Errorf("no user in context")
	}

	contactID := extractObjectIDFromPath(path, ".vcf")
	if contactID == "" {
		return nil, webdav.NewHTTPError(http.StatusNotFound, fmt.Errorf("address object not found"))
	}

	var contact models.Contact
	if err := b.db.Preload("ContactInformations.ContactInformationType").
		Preload("Addresses").
		First(&contact, "id = ?", contactID).Error; err != nil {
		return nil, webdav.NewHTTPError(http.StatusNotFound, fmt.Errorf("address object not found"))
	}

	if err := b.verifyVaultAccess(userID, contact.VaultID); err != nil {
		return nil, err
	}

	return contactToAddressObject(&contact, userID), nil
}

func (b *CardDAVBackend) ListAddressObjects(ctx context.Context, path string, _ *carddav.AddressDataRequest) ([]carddav.AddressObject, error) {
	userID := UserIDFromContext(ctx)
	if userID == "" {
		return nil, fmt.Errorf("no user in context")
	}

	vaultID := extractVaultIDFromPath(path, "addressbooks", userID)
	if vaultID == "" {
		return nil, webdav.NewHTTPError(http.StatusNotFound, fmt.Errorf("address book not found"))
	}

	if err := b.verifyVaultAccess(userID, vaultID); err != nil {
		return nil, err
	}

	var contacts []models.Contact
	if err := b.db.Preload("ContactInformations.ContactInformationType").
		Preload("Addresses").
		Where("vault_id = ?", vaultID).Find(&contacts).Error; err != nil {
		return nil, err
	}

	objects := make([]carddav.AddressObject, 0, len(contacts))
	for i := range contacts {
		objects = append(objects, *contactToAddressObject(&contacts[i], userID))
	}
	return objects, nil
}

func (b *CardDAVBackend) QueryAddressObjects(ctx context.Context, path string, query *carddav.AddressBookQuery) ([]carddav.AddressObject, error) {
	// For simplicity, list all and filter
	objects, err := b.ListAddressObjects(ctx, path, &carddav.AddressDataRequest{AllProp: true})
	if err != nil {
		return nil, err
	}

	if query == nil || len(query.PropFilters) == 0 {
		return objects, nil
	}

	var filtered []carddav.AddressObject
	for _, obj := range objects {
		if matchesQuery(obj.Card, query) {
			filtered = append(filtered, obj)
		}
	}
	return filtered, nil
}

func (b *CardDAVBackend) PutAddressObject(ctx context.Context, path string, card vcard.Card, _ *carddav.PutAddressObjectOptions) (*carddav.AddressObject, error) {
	userID := UserIDFromContext(ctx)
	if userID == "" {
		return nil, fmt.Errorf("no user in context")
	}

	// Determine vault ID from path
	vaultID := extractVaultIDFromAddressObjectPath(path, userID)
	if vaultID == "" {
		return nil, webdav.NewHTTPError(http.StatusBadRequest, fmt.Errorf("invalid path"))
	}

	if err := b.verifyVaultAccess(userID, vaultID); err != nil {
		return nil, err
	}

	firstName, lastName := extractNameFromCard(card)
	nickname := card.Value(vcard.FieldNickname)

	contactID := extractObjectIDFromPath(path, ".vcf")
	now := time.Now()

	var vault models.Vault
	if err := b.db.First(&vault, "id = ?", vaultID).Error; err != nil {
		return nil, fmt.Errorf("vault not found: %w", err)
	}
	accountID := AccountIDFromContext(ctx)
	if accountID == "" {
		accountID = vault.AccountID
	}

	title := card.Value(vcard.FieldTitle)

	var contact models.Contact
	if contactID != "" {
		err := b.db.First(&contact, "id = ?", contactID).Error
		if err == nil {
			contact.FirstName = strPtrOrNil(firstName)
			contact.LastName = strPtrOrNil(lastName)
			contact.Nickname = strPtrOrNil(nickname)
			contact.JobPosition = strPtrOrNil(title)
			contact.LastUpdatedAt = &now
			if err := b.db.Save(&contact).Error; err != nil {
				return nil, err
			}

			if err := replaceContactVCardFields(b.db, card, contact.ID, vaultID, accountID); err != nil {
				return nil, err
			}

			b.db.Preload("ContactInformations.ContactInformationType").First(&contact, "id = ?", contact.ID)
			return contactToAddressObject(&contact, userID), nil
		}
	}

	contact = models.Contact{
		VaultID:       vaultID,
		FirstName:     strPtrOrNil(firstName),
		LastName:      strPtrOrNil(lastName),
		Nickname:      strPtrOrNil(nickname),
		JobPosition:   strPtrOrNil(title),
		LastUpdatedAt: &now,
	}
	if err := b.db.Create(&contact).Error; err != nil {
		return nil, err
	}

	cvu := models.ContactVaultUser{
		ContactID: contact.ID,
		UserID:    userID,
		VaultID:   vaultID,
	}
	if err := b.db.Create(&cvu).Error; err != nil {
		return nil, err
	}

	if err := saveContactVCardFields(b.db, card, contact.ID, vaultID, accountID); err != nil {
		return nil, err
	}

	b.db.Preload("ContactInformations.ContactInformationType").First(&contact, "id = ?", contact.ID)
	return contactToAddressObject(&contact, userID), nil
}

func (b *CardDAVBackend) DeleteAddressObject(ctx context.Context, path string) error {
	userID := UserIDFromContext(ctx)
	if userID == "" {
		return fmt.Errorf("no user in context")
	}

	contactID := extractObjectIDFromPath(path, ".vcf")
	if contactID == "" {
		return webdav.NewHTTPError(http.StatusNotFound, fmt.Errorf("address object not found"))
	}

	var contact models.Contact
	if err := b.db.First(&contact, "id = ?", contactID).Error; err != nil {
		return webdav.NewHTTPError(http.StatusNotFound, fmt.Errorf("address object not found"))
	}

	if err := b.verifyVaultAccess(userID, contact.VaultID); err != nil {
		return err
	}

	if !contact.CanBeDeleted {
		return webdav.NewHTTPError(http.StatusForbidden, fmt.Errorf("contact cannot be deleted"))
	}

	return b.db.Delete(&contact).Error
}

func (b *CardDAVBackend) verifyVaultAccess(userID, vaultID string) error {
	var uv models.UserVault
	if err := b.db.Where("user_id = ? AND vault_id = ?", userID, vaultID).First(&uv).Error; err != nil {
		return webdav.NewHTTPError(http.StatusForbidden, fmt.Errorf("access denied"))
	}
	return nil
}

// contactToAddressObject converts a Contact model to a CardDAV AddressObject.
func contactToAddressObject(c *models.Contact, userID string) *carddav.AddressObject {
	card := contactToVCard(c)
	return &carddav.AddressObject{
		Path:    "/dav/addressbooks/" + userID + "/" + c.VaultID + "/" + c.ID + ".vcf",
		ModTime: c.UpdatedAt,
		ETag:    fmt.Sprintf("%d", c.UpdatedAt.Unix()),
		Card:    card,
	}
}

// contactToVCard builds a vCard from a Contact model.
func contactToVCard(c *models.Contact) vcard.Card {
	card := make(vcard.Card)
	card.SetValue(vcard.FieldVersion, "3.0")

	firstName := ptrToStr(c.FirstName)
	lastName := ptrToStr(c.LastName)

	card.SetName(&vcard.Name{
		FamilyName: lastName,
		GivenName:  firstName,
	})

	fn := buildFullName(firstName, lastName)
	if fn == "" {
		fn = "Unknown"
	}
	card.SetValue(vcard.FieldFormattedName, fn)

	if c.Nickname != nil && *c.Nickname != "" {
		card.SetValue(vcard.FieldNickname, *c.Nickname)
	}

	if c.Prefix != nil && *c.Prefix != "" {
		name := card.Name()
		if name != nil {
			name.HonorificPrefix = *c.Prefix
			card.SetName(name)
		}
	}

	if c.Suffix != nil && *c.Suffix != "" {
		name := card.Name()
		if name != nil {
			name.HonorificSuffix = *c.Suffix
			card.SetName(name)
		}
	}

	// Map contact information (email, phone)
	for _, info := range c.ContactInformations {
		typeName := ptrToStr(info.ContactInformationType.Type)
		switch typeName {
		case "phone":
			card.Add(vcard.FieldTelephone, &vcard.Field{
				Value:  info.Data,
				Params: vcard.Params{vcard.ParamType: {"VOICE"}},
			})
		case "email":
			card.Add(vcard.FieldEmail, &vcard.Field{
				Value:  info.Data,
				Params: vcard.Params{vcard.ParamType: {"INTERNET"}},
			})
		}
	}

	for _, addr := range c.Addresses {
		card.AddAddress(&vcard.Address{
			StreetAddress: ptrToStr(addr.Line1),
			Locality:      ptrToStr(addr.City),
			Region:        ptrToStr(addr.Province),
			PostalCode:    ptrToStr(addr.PostalCode),
			Country:       ptrToStr(addr.Country),
		})
	}

	card.SetValue("UID", c.ID)

	return card
}

// extractNameFromCard extracts first name and last name from a vCard.
func extractNameFromCard(card vcard.Card) (string, string) {
	name := card.Name()
	if name != nil && (name.GivenName != "" || name.FamilyName != "") {
		return name.GivenName, name.FamilyName
	}
	fn := card.Value(vcard.FieldFormattedName)
	if fn != "" {
		return fn, ""
	}
	return "", ""
}

// matchesQuery checks if a vCard matches a CardDAV address book query.
func matchesQuery(card vcard.Card, query *carddav.AddressBookQuery) bool {
	for _, pf := range query.PropFilters {
		values := card[pf.Name]
		if pf.IsNotDefined {
			if len(values) > 0 {
				return false
			}
			continue
		}
		if len(values) == 0 {
			return false
		}
		for _, tm := range pf.TextMatches {
			match := false
			for _, v := range values {
				if textMatchField(v.Value, tm) {
					match = true
					break
				}
			}
			if !match {
				return false
			}
		}
	}
	return true
}

func textMatchField(value string, tm carddav.TextMatch) bool {
	v := strings.ToLower(value)
	t := strings.ToLower(tm.Text)
	var result bool
	switch tm.MatchType {
	case carddav.MatchEquals:
		result = v == t
	case carddav.MatchStartsWith:
		result = strings.HasPrefix(v, t)
	case carddav.MatchEndsWith:
		result = strings.HasSuffix(v, t)
	default: // MatchContains
		result = strings.Contains(v, t)
	}
	if tm.NegateCondition {
		return !result
	}
	return result
}

// saveContactVCardFields creates TEL, EMAIL, ADR, BDAY records from a vCard.
func saveContactVCardFields(db *gorm.DB, card vcard.Card, contactID, vaultID, accountID string) error {
	// TEL
	if fields := card[vcard.FieldTelephone]; len(fields) > 0 {
		var phoneType models.ContactInformationType
		if err := db.Where("account_id = ? AND type = ?", accountID, "phone").First(&phoneType).Error; err == nil {
			for _, f := range fields {
				if f.Value == "" {
					continue
				}
				if err := db.Create(&models.ContactInformation{
					ContactID: contactID,
					TypeID:    phoneType.ID,
					Data:      f.Value,
				}).Error; err != nil {
					return err
				}
			}
		}
	}

	// EMAIL
	if fields := card[vcard.FieldEmail]; len(fields) > 0 {
		var emailType models.ContactInformationType
		if err := db.Where("account_id = ? AND type = ?", accountID, "email").First(&emailType).Error; err == nil {
			for _, f := range fields {
				if f.Value == "" {
					continue
				}
				if err := db.Create(&models.ContactInformation{
					ContactID: contactID,
					TypeID:    emailType.ID,
					Data:      f.Value,
				}).Error; err != nil {
					return err
				}
			}
		}
	}

	// ADR
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
			if err := db.Create(&a).Error; err != nil {
				return err
			}
			if err := db.Create(&models.ContactAddress{
				ContactID: contactID,
				AddressID: a.ID,
			}).Error; err != nil {
				return err
			}
		}
	}

	// BDAY
	if bday := card.Value(vcard.FieldBirthday); bday != "" {
		year, month, day := parseBirthdayString(bday)
		if month > 0 && day > 0 {
			var bdayType models.ContactImportantDateType
			if err := db.Where("vault_id = ? AND internal_type = ?", vaultID, "birthdate").First(&bdayType).Error; err == nil {
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
				if err := db.Create(&cid).Error; err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// replaceContactVCardFields deletes existing records and recreates from vCard.
func replaceContactVCardFields(db *gorm.DB, card vcard.Card, contactID, vaultID, accountID string) error {
	db.Where("contact_id = ?", contactID).Delete(&models.ContactInformation{})

	var pivots []models.ContactAddress
	db.Where("contact_id = ?", contactID).Find(&pivots)
	if len(pivots) > 0 {
		addressIDs := make([]uint, len(pivots))
		for i, p := range pivots {
			addressIDs[i] = p.AddressID
		}
		db.Where("contact_id = ?", contactID).Delete(&models.ContactAddress{})
		db.Where("id IN ?", addressIDs).Delete(&models.Address{})
	}

	db.Where("contact_id = ?", contactID).Delete(&models.ContactImportantDate{})

	return saveContactVCardFields(db, card, contactID, vaultID, accountID)
}

// parseBirthdayString parses vCard BDAY formats: "19900115", "1990-01-15", "--0115" (no year)
func parseBirthdayString(bday string) (year, month, day int) {
	bday = strings.TrimSpace(bday)
	if t, err := time.Parse("2006-01-02", bday); err == nil {
		return t.Year(), int(t.Month()), t.Day()
	}
	if len(bday) == 8 {
		if y, err := strconv.Atoi(bday[0:4]); err == nil {
			if m, err := strconv.Atoi(bday[4:6]); err == nil {
				if d, err := strconv.Atoi(bday[6:8]); err == nil {
					return y, m, d
				}
			}
		}
	}
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

// Path parsing helpers

// extractVaultIDFromPath extracts the vault ID from a DAV path like
// /dav/{collection}/{userID}/{vaultID}/
func extractVaultIDFromPath(path, collection, userID string) string {
	prefix := "/dav/" + collection + "/" + userID + "/"
	path = strings.TrimSuffix(path, "/")
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	rest := strings.TrimPrefix(path, prefix)
	// rest could be "vaultID" or "vaultID/objectID.ext"
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) == 0 || parts[0] == "" {
		return ""
	}
	return parts[0]
}

// extractObjectIDFromPath extracts the object ID from a path like
// .../objectID.vcf or .../objectID.ics
func extractObjectIDFromPath(path, suffix string) string {
	parts := strings.Split(strings.TrimSuffix(path, "/"), "/")
	if len(parts) == 0 {
		return ""
	}
	last := parts[len(parts)-1]
	if !strings.HasSuffix(last, suffix) {
		return ""
	}
	return strings.TrimSuffix(last, suffix)
}

// extractVaultIDFromAddressObjectPath extracts vault ID from a full object path
// like /dav/addressbooks/{userID}/{vaultID}/{objectID}.vcf
func extractVaultIDFromAddressObjectPath(path, userID string) string {
	prefix := "/dav/addressbooks/" + userID + "/"
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	rest := strings.TrimPrefix(path, prefix)
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) < 2 || parts[0] == "" {
		return ""
	}
	return parts[0]
}

// String helpers

func ptrToStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func strPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
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
