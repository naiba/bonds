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
	var contact models.Contact
	if err := s.db.Where("id = ? AND vault_id = ?", contactID, vaultID).First(&contact).Error; err != nil {
		return nil, ErrContactNotFound
	}

	var contactInfos []models.ContactInformation
	s.db.Preload("ContactInformationType").Where("contact_id = ?", contactID).Find(&contactInfos)

	var addresses []models.Address
	var pivots []models.ContactAddress
	s.db.Where("contact_id = ?", contactID).Find(&pivots)
	if len(pivots) > 0 {
		addressIDs := make([]uint, len(pivots))
		for i, p := range pivots {
			addressIDs[i] = p.AddressID
		}
		s.db.Where("id IN ?", addressIDs).Find(&addresses)
	}

	return buildVCard(&contact, contactInfos, addresses), nil
}

func (s *VCardService) ExportContact(contactID string, vaultID string) ([]byte, error) {
	var contact models.Contact
	if err := s.db.Where("id = ? AND vault_id = ?", contactID, vaultID).First(&contact).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrContactNotFound
		}
		return nil, err
	}

	var contactInfos []models.ContactInformation
	s.db.Preload("ContactInformationType").Where("contact_id = ?", contactID).Find(&contactInfos)

	var addresses []models.Address
	var pivots []models.ContactAddress
	s.db.Where("contact_id = ?", contactID).Find(&pivots)
	if len(pivots) > 0 {
		addressIDs := make([]uint, len(pivots))
		for i, p := range pivots {
			addressIDs[i] = p.AddressID
		}
		s.db.Where("id IN ?", addressIDs).Find(&addresses)
	}

	card := buildVCard(&contact, contactInfos, addresses)

	var buf bytes.Buffer
	enc := vcard.NewEncoder(&buf)
	if err := enc.Encode(card); err != nil {
		return nil, fmt.Errorf("failed to encode vcard: %w", err)
	}
	return buf.Bytes(), nil
}

func (s *VCardService) ExportVault(vaultID string) ([]byte, error) {
	var contacts []models.Contact
	if err := s.db.Where("vault_id = ?", vaultID).Find(&contacts).Error; err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	enc := vcard.NewEncoder(&buf)

	for _, contact := range contacts {
		var contactInfos []models.ContactInformation
		s.db.Preload("ContactInformationType").Where("contact_id = ?", contact.ID).Find(&contactInfos)

		var addresses []models.Address
		var pivots []models.ContactAddress
		s.db.Where("contact_id = ?", contact.ID).Find(&pivots)
		if len(pivots) > 0 {
			addressIDs := make([]uint, len(pivots))
			for i, p := range pivots {
				addressIDs[i] = p.AddressID
			}
			s.db.Where("id IN ?", addressIDs).Find(&addresses)
		}

		card := buildVCard(&contact, contactInfos, addresses)
		if err := enc.Encode(card); err != nil {
			return nil, fmt.Errorf("failed to encode vcard for contact %s: %w", contact.ID, err)
		}
	}

	return buf.Bytes(), nil
}

func (s *VCardService) ImportVCard(vaultID, userID string, data io.Reader) (*dto.VCardImportResponse, error) {
	dec := vcard.NewDecoder(data)

	var imported []dto.ContactResponse
	var skippedCount int
	var importErrors []string

	err := s.db.Transaction(func(tx *gorm.DB) error {
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

			firstName, lastName := extractNameFromCard(card)
			nickname := card.Value(vcard.FieldNickname)

			title := card.Value(vcard.FieldTitle)

			now := time.Now()
			contact := models.Contact{
				VaultID:       vaultID,
				FirstName:     strPtrOrNil(firstName),
				LastName:      strPtrOrNil(lastName),
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

			imported = append(imported, toContactResponse(&contact, false))
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

func buildVCard(contact *models.Contact, infos []models.ContactInformation, addresses []models.Address) vcard.Card {
	card := make(vcard.Card)
	card.SetValue(vcard.FieldVersion, "3.0")

	firstName := ptrToStr(contact.FirstName)
	lastName := ptrToStr(contact.LastName)

	card.SetName(&vcard.Name{
		FamilyName: lastName,
		GivenName:  firstName,
	})
	card.SetValue(vcard.FieldFormattedName, buildFullName(firstName, lastName))

	if contact.Nickname != nil && *contact.Nickname != "" {
		card.SetValue(vcard.FieldNickname, *contact.Nickname)
	}

	for _, info := range infos {
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

	for _, addr := range addresses {
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

			firstName, lastName := extractNameFromCard(card)
			nickname := card.Value(vcard.FieldNickname)
			title := card.Value(vcard.FieldTitle)
			now := time.Now()

			existing.FirstName = strPtrOrNil(firstName)
			existing.LastName = strPtrOrNil(lastName)
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

	firstName, lastName := extractNameFromCard(card)
	nickname := card.Value(vcard.FieldNickname)
	title := card.Value(vcard.FieldTitle)
	now := time.Now()

	uid := card.Value("UID")

	contact := models.Contact{
		VaultID:       vaultID,
		FirstName:     strPtrOrNil(firstName),
		LastName:      strPtrOrNil(lastName),
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
