package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/naiba/bonds/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var ErrInvalidPasswordAction = errors.New("invalid or expired password action")
var ErrEmailChangeConflict = errors.New("email is already in use")

type CredentialActionService struct {
	db       *gorm.DB
	mailer   Mailer
	appURL   string
	settings *SystemSettingService
}

func NewCredentialActionService(db *gorm.DB, mailer Mailer, appURL string, settings *SystemSettingService) *CredentialActionService {
	return &CredentialActionService{db: db, mailer: mailer, appURL: appURL, settings: settings}
}

func (s *CredentialActionService) Send(tx *gorm.DB, user *models.User, purpose string) error {
	if purpose != "password_reset" && purpose != "password_setup" {
		return ErrInvalidPasswordAction
	}
	if s.mailer == nil {
		return ErrMailerNotConfigured
	}
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return err
	}
	raw := hex.EncodeToString(bytes)
	if err := tx.Where("user_id = ? AND purpose IN ? AND used_at IS NULL", user.ID, []string{"password_setup", "password_reset"}).
		Delete(&models.AuthActionToken{}).Error; err != nil {
		return err
	}
	key := sha256.Sum256([]byte(raw))
	if err := tx.Create(&models.AuthActionToken{
		UserID: user.ID, TokenHash: hex.EncodeToString(key[:]), Purpose: purpose,
		ExpiresAt: time.Now().Add(time.Hour),
	}).Error; err != nil {
		return err
	}
	appURL := s.appURL
	if s.settings != nil {
		appURL = s.settings.GetWithDefault("app.url", appURL)
	}
	link := fmt.Sprintf("%s/set-password?token=%s", appURL, raw)
	// Never include the token or URL in any operator-visible log or API result.
	return s.mailer.Send(user.Email, "Set your Bonds password", "Open this one-time link to set your password (valid for one hour): <a href=\""+html.EscapeString(link)+"\">Set password</a>")
}

// A change of login email needs consent from both addresses. Neither the
// operator nor a recipient of only one message can take over the identity.
func (s *CredentialActionService) RequestEmailChange(tx *gorm.DB, user *models.User, email string) error {
	if s.mailer == nil {
		return ErrMailerNotConfigured
	}
	email = strings.ToLower(strings.TrimSpace(email))
	if strings.EqualFold(email, user.Email) {
		return ErrEmailChangeConflict
	}
	var count int64
	if err := tx.Model(&models.User{}).Where("LOWER(email) = ?", email).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return ErrEmailChangeConflict
	}
	if err := tx.Where("user_id = ? AND purpose IN ?", user.ID, []string{"email_old", "email_new"}).Delete(&models.AuthActionToken{}).Error; err != nil {
		return err
	}
	appURL := s.appURL
	if s.settings != nil {
		appURL = s.settings.GetWithDefault("app.url", appURL)
	}
	for _, destination := range []struct{ address, purpose string }{{user.Email, "email_old"}, {email, "email_new"}} {
		bytes := make([]byte, 32)
		if _, err := rand.Read(bytes); err != nil {
			return err
		}
		raw := hex.EncodeToString(bytes)
		hash := sha256.Sum256([]byte(raw))
		if err := tx.Create(&models.AuthActionToken{UserID: user.ID, TokenHash: hex.EncodeToString(hash[:]), Purpose: destination.purpose, TargetEmail: email, ExpiresAt: time.Now().Add(time.Hour)}).Error; err != nil {
			return err
		}
		link := fmt.Sprintf("%s/confirm-email-change?token=%s", appURL, raw)
		if err := s.mailer.Send(destination.address, "Confirm your Bonds email change", "Confirm the change of login email to "+html.EscapeString(email)+" (valid for one hour): <a href=\""+html.EscapeString(link)+"\">Confirm email change</a>"); err != nil {
			return err
		}
	}
	return nil
}

func (s *CredentialActionService) ConfirmEmailChange(raw string) error {
	hash := sha256.Sum256([]byte(raw))
	return s.db.Transaction(func(tx *gorm.DB) error {
		var action models.AuthActionToken
		if err := tx.Where("token_hash = ? AND purpose IN ? AND used_at IS NULL AND expires_at > ?", hex.EncodeToString(hash[:]), []string{"email_old", "email_new"}, time.Now()).First(&action).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrInvalidPasswordAction
			}
			return err
		}
		now := time.Now()
		result := tx.Model(&models.AuthActionToken{}).Where("id = ? AND used_at IS NULL", action.ID).Update("used_at", now)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrInvalidPasswordAction
		}
		var counterpart models.AuthActionToken
		otherPurpose := "email_old"
		if action.Purpose == "email_old" {
			otherPurpose = "email_new"
		}
		if err := tx.Where("user_id = ? AND target_email = ? AND purpose = ? AND expires_at > ?", action.UserID, action.TargetEmail, otherPurpose, now).First(&counterpart).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrInvalidPasswordAction
			}
			return err
		}
		if counterpart.UsedAt == nil {
			return nil
		}
		var existing int64
		if err := tx.Model(&models.User{}).Where("LOWER(email) = ? AND id <> ?", action.TargetEmail, action.UserID).Count(&existing).Error; err != nil {
			return err
		}
		if existing != 0 {
			return ErrEmailChangeConflict
		}
		var user models.User
		if err := tx.Select("email").First(&user, "id = ?", action.UserID).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.User{}).Where("id = ?", action.UserID).Updates(map[string]interface{}{"email": action.TargetEmail, "email_verified_at": now, "email_ownership_unproven": false, "auth_version": gorm.Expr("auth_version + 1")}).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.UserNotificationChannel{}).Where("user_id = ? AND type = ? AND LOWER(content) = LOWER(?)", action.UserID, "email", user.Email).Update("content", action.TargetEmail).Error; err != nil {
			return err
		}
		return tx.Where("user_id = ?", action.UserID).Delete(&models.PersonalAccessToken{}).Error
	})
}

func (s *CredentialActionService) Complete(raw, password string) error {
	if len(password) < 8 {
		return ErrInvalidPasswordAction
	}
	hash := sha256.Sum256([]byte(raw))
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		var action models.AuthActionToken
		if err := tx.Where("token_hash = ? AND purpose IN ? AND used_at IS NULL AND expires_at > ?", hex.EncodeToString(hash[:]), []string{"password_reset", "password_setup"}, time.Now()).First(&action).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrInvalidPasswordAction
			}
			return err
		}
		now := time.Now()
		if update := tx.Model(&models.AuthActionToken{}).
			Where("id = ? AND used_at IS NULL", action.ID).Update("used_at", now); update.Error != nil || update.RowsAffected != 1 {
			if update.Error != nil {
				return update.Error
			}
			return ErrInvalidPasswordAction
		}
		if err := tx.Model(&models.User{}).Where("id = ?", action.UserID).
			Updates(map[string]interface{}{"password": string(hashed), "auth_version": gorm.Expr("auth_version + 1"), "email_verified_at": now, "email_ownership_unproven": false}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ? AND purpose IN ? AND used_at IS NULL", action.UserID, []string{"password_setup", "password_reset"}).Delete(&models.AuthActionToken{}).Error; err != nil {
			return err
		}
		return tx.Where("user_id = ?", action.UserID).Delete(&models.PersonalAccessToken{}).Error
	})
}
