package services

import (
	"errors"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var (
	ErrLoanNotFound           = errors.New("loan not found")
	ErrLoanCurrencyNotEnabled = errors.New("loan currency is not enabled")
)

type LoanService struct {
	db           *gorm.DB
	feedRecorder *FeedRecorder
}

func NewLoanService(db *gorm.DB) *LoanService {
	return &LoanService{db: db}
}

func (s *LoanService) SetFeedRecorder(fr *FeedRecorder) {
	s.feedRecorder = fr
}

func (s *LoanService) List(contactID, vaultID string) ([]dto.LoanResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
	var loanIDs []uint
	if err := s.db.Model(&models.ContactLoan{}).
		Where("loaner_id = ? OR loanee_id = ?", contactID, contactID).
		Distinct("loan_id").
		Pluck("loan_id", &loanIDs).Error; err != nil {
		return nil, err
	}
	if len(loanIDs) == 0 {
		return []dto.LoanResponse{}, nil
	}

	var loans []models.Loan
	if err := s.db.Where("id IN ? AND vault_id = ?", loanIDs, vaultID).Order("created_at DESC").Find(&loans).Error; err != nil {
		return nil, err
	}

	result := make([]dto.LoanResponse, len(loans))
	for i, l := range loans {
		result[i] = toLoanResponse(&l)
	}
	return result, nil
}

func (s *LoanService) Create(contactID, vaultID string, req dto.CreateLoanRequest) (*dto.LoanResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
	if err := s.validateCurrencyEnabled(vaultID, req.CurrencyID); err != nil {
		return nil, err
	}
	loan := models.Loan{
		VaultID:     vaultID,
		Name:        req.Name,
		Type:        req.Type,
		Category:    loanCategoryOrDefault(req.Category),
		Description: strPtrOrNil(req.Description),
		ItemName:    req.ItemName,
		Quantity:    req.Quantity,
		AmountLent:  req.AmountLent,
		CurrencyID:  req.CurrencyID,
		LoanedAt:    req.LoanedAt,
		DueAt:       req.DueAt,
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&loan).Error; err != nil {
			return err
		}
		pivot := models.ContactLoan{
			LoanID:   loan.ID,
			LoanerID: contactID,
			LoaneeID: contactID,
		}
		return tx.Create(&pivot).Error
	})
	if err != nil {
		return nil, err
	}

	if s.feedRecorder != nil {
		entityType := "Loan"
		s.feedRecorder.Record(contactID, "", ActionLoanCreated, "Created loan: "+req.Name, &loan.ID, &entityType)
	}

	resp := toLoanResponse(&loan)
	return &resp, nil
}

func (s *LoanService) Update(id uint, vaultID string, req dto.UpdateLoanRequest) (*dto.LoanResponse, error) {
	var loan models.Loan
	if err := s.db.Where("id = ? AND vault_id = ?", id, vaultID).First(&loan).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrLoanNotFound
		}
		return nil, err
	}
	if !sameOptionalUint(loan.CurrencyID, req.CurrencyID) {
		if err := s.validateCurrencyEnabled(vaultID, req.CurrencyID); err != nil {
			return nil, err
		}
	}
	loan.Name = req.Name
	loan.Type = req.Type
	loan.Category = loanCategoryOrDefault(req.Category)
	loan.Description = strPtrOrNil(req.Description)
	loan.ItemName = req.ItemName
	loan.Quantity = req.Quantity
	loan.AmountLent = req.AmountLent
	loan.CurrencyID = req.CurrencyID
	loan.LoanedAt = req.LoanedAt
	loan.DueAt = req.DueAt
	if err := s.db.Save(&loan).Error; err != nil {
		return nil, err
	}
	resp := toLoanResponse(&loan)
	return &resp, nil
}

func (s *LoanService) ToggleSettled(id uint, vaultID string) (*dto.LoanResponse, error) {
	var loan models.Loan
	if err := s.db.Where("id = ? AND vault_id = ?", id, vaultID).First(&loan).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrLoanNotFound
		}
		return nil, err
	}
	loan.Settled = !loan.Settled
	if loan.Settled {
		now := time.Now()
		loan.SettledAt = &now
		loan.ReturnedAt = &now
	} else {
		loan.SettledAt = nil
		loan.ReturnedAt = nil
	}
	if err := s.db.Save(&loan).Error; err != nil {
		return nil, err
	}
	resp := toLoanResponse(&loan)
	return &resp, nil
}

func (s *LoanService) Delete(id uint, vaultID string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("loan_id = ?", id).Delete(&models.ContactLoan{}).Error; err != nil {
			return err
		}
		result := tx.Where("id = ? AND vault_id = ?", id, vaultID).Delete(&models.Loan{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrLoanNotFound
		}
		return nil
	})
}

func loanCategoryOrDefault(category string) string {
	if category == "" {
		return "money"
	}
	return category
}

func (s *LoanService) validateCurrencyEnabled(vaultID string, currencyID *uint) error {
	if currencyID == nil {
		return nil
	}
	var vault models.Vault
	if err := s.db.Select("account_id").First(&vault, "id = ?", vaultID).Error; err != nil {
		return err
	}
	var count int64
	if err := s.db.Model(&models.AccountCurrency{}).
		Where("account_id = ? AND currency_id = ?", vault.AccountID, *currencyID).
		Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return ErrLoanCurrencyNotEnabled
	}
	return nil
}

func sameOptionalUint(a, b *uint) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func toLoanResponse(l *models.Loan) dto.LoanResponse {
	return dto.LoanResponse{
		ID:          l.ID,
		VaultID:     l.VaultID,
		Name:        l.Name,
		Type:        l.Type,
		Category:    loanCategoryOrDefault(l.Category),
		Description: ptrToStr(l.Description),
		ItemName:    l.ItemName,
		Quantity:    l.Quantity,
		AmountLent:  l.AmountLent,
		CurrencyID:  l.CurrencyID,
		LoanedAt:    l.LoanedAt,
		DueAt:       l.DueAt,
		Settled:     l.Settled,
		SettledAt:   l.SettledAt,
		ReturnedAt:  l.ReturnedAt,
		CreatedAt:   l.CreatedAt,
		UpdatedAt:   l.UpdatedAt,
	}
}
