package services

import (
	"errors"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var (
	ErrCannotDeleteSelf    = errors.New("cannot delete yourself")
	ErrManagedUserNotFound = errors.New("managed user not found")
)

type UserManagementService struct {
	db *gorm.DB
}

func NewUserManagementService(db *gorm.DB) *UserManagementService {
	return &UserManagementService{db: db}
}

func (s *UserManagementService) List(accountID string) ([]dto.UserManagementResponse, error) {
	var users []models.User
	if err := s.db.Where("account_id = ?", accountID).Order("created_at ASC").Find(&users).Error; err != nil {
		return nil, err
	}
	result := make([]dto.UserManagementResponse, len(users))
	for i, u := range users {
		result[i] = toUserManagementResponse(&u)
	}
	return result, nil
}

func (s *UserManagementService) Update(id, accountID string, req dto.UpdateManagedUserRequest) (*dto.UserManagementResponse, error) {
	var user models.User
	if err := s.db.Where("id = ? AND account_id = ?", id, accountID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrManagedUserNotFound
		}
		return nil, err
	}
	user.FirstName = strPtrOrNil(req.FirstName)
	user.LastName = strPtrOrNil(req.LastName)
	user.IsAccountAdministrator = req.IsAdmin
	if err := s.db.Save(&user).Error; err != nil {
		return nil, err
	}
	resp := toUserManagementResponse(&user)
	return &resp, nil
}

func (s *UserManagementService) Delete(id, accountID, currentUserID string) error {
	if id == currentUserID {
		return ErrCannotDeleteSelf
	}
	var user models.User
	if err := s.db.Where("id = ? AND account_id = ?", id, accountID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrManagedUserNotFound
		}
		return err
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", id).Delete(&models.UserVault{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", id).Delete(&models.UserNotificationChannel{}).Error; err != nil {
			return err
		}
		return tx.Delete(&user).Error
	})
}

func toUserManagementResponse(u *models.User) dto.UserManagementResponse {
	return dto.UserManagementResponse{
		ID:        u.ID,
		Email:     u.Email,
		FirstName: ptrToStr(u.FirstName),
		LastName:  ptrToStr(u.LastName),
		IsAdmin:   u.IsAccountAdministrator,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}
