package services

import (
	"errors"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var ErrPetNotFound = errors.New("pet not found")
var ErrPetCategoryNotFound = errors.New("pet category not found")

type PetService struct {
	db *gorm.DB
}

func NewPetService(db *gorm.DB) *PetService {
	return &PetService{db: db}
}

func (s *PetService) List(contactID, vaultID string) ([]dto.PetResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
	var pets []models.Pet
	if err := s.db.Preload("PetCategory").Where("contact_id = ?", contactID).Order("created_at DESC").Find(&pets).Error; err != nil {
		return nil, err
	}
	result := make([]dto.PetResponse, len(pets))
	for i, p := range pets {
		result[i] = toPetResponse(&p)
	}
	return result, nil
}

func (s *PetService) Create(contactID, vaultID string, req dto.CreatePetRequest) (*dto.PetResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
	petCategory, err := s.getPetCategoryForContact(contactID, req.PetCategoryID)
	if err != nil {
		return nil, err
	}
	pet := models.Pet{
		ContactID:     contactID,
		PetCategoryID: req.PetCategoryID,
		Name:          strPtrOrNil(req.Name),
	}
	if err := s.db.Create(&pet).Error; err != nil {
		return nil, err
	}
	pet.PetCategory = *petCategory
	resp := toPetResponse(&pet)
	return &resp, nil
}

func (s *PetService) Update(id uint, contactID, vaultID string, req dto.UpdatePetRequest) (*dto.PetResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
	var pet models.Pet
	if err := s.db.Where("id = ? AND contact_id = ?", id, contactID).First(&pet).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPetNotFound
		}
		return nil, err
	}
	petCategory, err := s.getPetCategoryForContact(contactID, req.PetCategoryID)
	if err != nil {
		return nil, err
	}
	pet.PetCategoryID = req.PetCategoryID
	pet.Name = strPtrOrNil(req.Name)
	if err := updateContactRecord(s.db, &pet, contactID, vaultID, ErrPetNotFound, "pet_category_id", "name"); err != nil {
		return nil, err
	}
	pet.PetCategory = *petCategory
	resp := toPetResponse(&pet)
	return &resp, nil
}

func (s *PetService) ListCategories(accountID string) ([]dto.PetCategoryResponse, error) {
	var categories []models.PetCategory
	if err := s.db.Where("account_id = ?", accountID).Order("id ASC").Find(&categories).Error; err != nil {
		return nil, err
	}
	result := make([]dto.PetCategoryResponse, len(categories))
	for i, category := range categories {
		result[i] = dto.PetCategoryResponse{
			ID:        category.ID,
			Name:      ptrToStr(category.Name),
			CreatedAt: category.CreatedAt,
			UpdatedAt: category.UpdatedAt,
		}
	}
	return result, nil
}

func (s *PetService) Delete(id uint, contactID, vaultID string) error {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return err
	}
	result := s.db.Where("id = ? AND contact_id = ?", id, contactID).Delete(&models.Pet{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrPetNotFound
	}
	return nil
}

func (s *PetService) getPetCategoryForContact(contactID string, petCategoryID uint) (*models.PetCategory, error) {
	if petCategoryID == 0 {
		return nil, ErrPetCategoryNotFound
	}

	var contact models.Contact
	if err := s.db.Select("vault_id").Where("id = ?", contactID).First(&contact).Error; err != nil {
		return nil, ErrContactNotFound
	}

	var vault models.Vault
	if err := s.db.Select("account_id").Where("id = ?", contact.VaultID).First(&vault).Error; err != nil {
		return nil, ErrContactNotFound
	}

	var petCategory models.PetCategory
	// Category IDs are account-scoped, so a positive ID alone is not enough.
	if err := s.db.Where("id = ? AND account_id = ?", petCategoryID, vault.AccountID).First(&petCategory).Error; err != nil {
		return nil, ErrPetCategoryNotFound
	}

	return &petCategory, nil
}

func toPetResponse(p *models.Pet) dto.PetResponse {
	return dto.PetResponse{
		ID:              p.ID,
		ContactID:       p.ContactID,
		PetCategoryID:   p.PetCategoryID,
		PetCategoryName: ptrToStr(p.PetCategory.Name),
		Name:            ptrToStr(p.Name),
		CreatedAt:       p.CreatedAt,
		UpdatedAt:       p.UpdatedAt,
	}
}
