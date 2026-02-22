package services

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrInvitationNotFound = errors.New("invitation not found")
	ErrInvitationExpired  = errors.New("invitation expired")
	ErrUserAlreadyExists  = errors.New("user already exists in this account")
)

type InvitationService struct {
	db       *gorm.DB
	mailer   Mailer
	appURL   string
	settings *SystemSettingService
}

func NewInvitationService(db *gorm.DB, mailer Mailer, appURL string) *InvitationService {
	return &InvitationService{db: db, mailer: mailer, appURL: appURL}
}

func (s *InvitationService) SetSystemSettings(settings *SystemSettingService) {
	s.settings = settings
}

func (s *InvitationService) getAppURL() string {
	if s.settings != nil {
		return s.settings.GetWithDefault("app.url", s.appURL)
	}
	return s.appURL
}

func (s *InvitationService) Create(accountID, createdBy string, req dto.CreateInvitationRequest) (*dto.InvitationResponse, error) {
	var existingUser models.User
	err := s.db.Where("email = ? AND account_id = ?", req.Email, accountID).First(&existingUser).Error
	if err == nil {
		return nil, ErrUserAlreadyExists
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	s.db.Where("account_id = ? AND email = ? AND accepted_at IS NULL", accountID, req.Email).Delete(&models.Invitation{})

	token := uuid.New().String()
	invitation := models.Invitation{
		AccountID:  accountID,
		Email:      req.Email,
		Token:      token,
		Permission: req.Permission,
		ExpiresAt:  time.Now().Add(7 * 24 * time.Hour),
		CreatedBy:  createdBy,
	}

	if err := s.db.Create(&invitation).Error; err != nil {
		return nil, err
	}

	inviteLink := fmt.Sprintf("%s/accept-invite?token=%s", s.getAppURL(), token)
	subject := "You've been invited to Bonds"
	body := fmt.Sprintf(
		`<h2>You've been invited!</h2>
<p>You've been invited to join a Bonds account. Click the link below to accept the invitation:</p>
<p><a href="%s">Accept Invitation</a></p>
<p>This invitation expires in 7 days.</p>`,
		inviteLink,
	)
	if err := s.mailer.Send(req.Email, subject, body); err != nil {
		fmt.Printf("[InvitationService] Failed to send invitation email to %s: %v\n", req.Email, err)
	}

	resp := toInvitationResponse(&invitation)
	return &resp, nil
}

func (s *InvitationService) Accept(req dto.AcceptInvitationRequest) (*dto.InvitationResponse, error) {
	var invitation models.Invitation
	err := s.db.Where("token = ?", req.Token).First(&invitation).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvitationNotFound
		}
		return nil, err
	}

	if invitation.AcceptedAt != nil {
		return nil, ErrInvitationNotFound
	}

	if time.Now().After(invitation.ExpiresAt) {
		return nil, ErrInvitationExpired
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	hashedStr := string(hashedPassword)
	now := time.Now()

	err = s.db.Transaction(func(tx *gorm.DB) error {
		user := models.User{
			AccountID:            invitation.AccountID,
			FirstName:            strPtrOrNil(req.FirstName),
			LastName:             strPtrOrNil(req.LastName),
			Email:                invitation.Email,
			Password:             &hashedStr,
			InvitationCode:       &invitation.Token,
			InvitationAcceptedAt: &now,
			EmailVerifiedAt:      &now,
		}
		if err := tx.Create(&user).Error; err != nil {
			return err
		}

		invitation.AcceptedAt = &now
		return tx.Save(&invitation).Error
	})
	if err != nil {
		return nil, err
	}

	resp := toInvitationResponse(&invitation)
	return &resp, nil
}

func (s *InvitationService) List(accountID string) ([]dto.InvitationResponse, error) {
	var invitations []models.Invitation
	if err := s.db.Where("account_id = ?", accountID).Order("created_at DESC").Find(&invitations).Error; err != nil {
		return nil, err
	}
	result := make([]dto.InvitationResponse, len(invitations))
	for i, inv := range invitations {
		result[i] = toInvitationResponse(&inv)
	}
	return result, nil
}

func (s *InvitationService) Delete(id uint, accountID string) error {
	var invitation models.Invitation
	if err := s.db.Where("id = ? AND account_id = ?", id, accountID).First(&invitation).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrInvitationNotFound
		}
		return err
	}
	if invitation.AcceptedAt != nil {
		return ErrInvitationNotFound
	}
	return s.db.Delete(&invitation).Error
}

func toInvitationResponse(inv *models.Invitation) dto.InvitationResponse {
	return dto.InvitationResponse{
		ID:         inv.ID,
		Email:      inv.Email,
		Permission: inv.Permission,
		ExpiresAt:  inv.ExpiresAt,
		AcceptedAt: inv.AcceptedAt,
		CreatedAt:  inv.CreatedAt,
	}
}
