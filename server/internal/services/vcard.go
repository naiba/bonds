package services

import (
	"bytes"
	"errors"
	"fmt"
	"io"
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

	err := s.db.Transaction(func(tx *gorm.DB) error {
		for {
			card, err := dec.Decode()
			if err == io.EOF {
				break
			}
			if err != nil {
				return fmt.Errorf("%w: %v", ErrVCardInvalidData, err)
			}

			firstName, lastName := extractNameFromCard(card)
			nickname := card.Value(vcard.FieldNickname)

			now := time.Now()
			contact := models.Contact{
				VaultID:       vaultID,
				FirstName:     strPtrOrNil(firstName),
				LastName:      strPtrOrNil(lastName),
				Nickname:      strPtrOrNil(nickname),
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

			imported = append(imported, toContactResponse(&contact, false))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &dto.VCardImportResponse{
		ImportedCount: len(imported),
		Contacts:      imported,
	}, nil
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
