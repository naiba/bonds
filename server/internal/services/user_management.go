package services

import (
	"errors"
	"math"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/pkg/response"
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

func (s *UserManagementService) List(accountID string, page, perPage int) ([]dto.UserManagementResponse, response.Meta, error) {
	query := s.db.Where("account_id = ?", accountID)

	var total int64
	if err := query.Model(&models.User{}).Count(&total).Error; err != nil {
		return nil, response.Meta{}, err
	}

	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	var users []models.User
	if err := query.Order("created_at ASC").Offset(offset).Limit(perPage).Find(&users).Error; err != nil {
		return nil, response.Meta{}, err
	}
	result := make([]dto.UserManagementResponse, len(users))
	for i, u := range users {
		result[i] = toUserManagementResponse(&u)
	}

	meta := response.Meta{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: int(math.Ceil(float64(total) / float64(perPage))),
	}
	return result, meta, nil
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
