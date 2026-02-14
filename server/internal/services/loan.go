package services

import (
	"errors"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var ErrLoanNotFound = errors.New("loan not found")

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
	loan := models.Loan{
		VaultID:     vaultID,
		Name:        req.Name,
		Type:        req.Type,
		Description: strPtrOrNil(req.Description),
		AmountLent:  req.AmountLent,
		CurrencyID:  req.CurrencyID,
		LoanedAt:    req.LoanedAt,
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
	loan.Name = req.Name
	loan.Type = req.Type
	loan.Description = strPtrOrNil(req.Description)
	loan.AmountLent = req.AmountLent
	loan.CurrencyID = req.CurrencyID
	loan.LoanedAt = req.LoanedAt
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
	} else {
		loan.SettledAt = nil
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

func toLoanResponse(l *models.Loan) dto.LoanResponse {
	return dto.LoanResponse{
		ID:          l.ID,
		VaultID:     l.VaultID,
		Name:        l.Name,
		Type:        l.Type,
		Description: ptrToStr(l.Description),
		AmountLent:  l.AmountLent,
		CurrencyID:  l.CurrencyID,
		LoanedAt:    l.LoanedAt,
		Settled:     l.Settled,
		SettledAt:   l.SettledAt,
		CreatedAt:   l.CreatedAt,
		UpdatedAt:   l.UpdatedAt,
	}
}
