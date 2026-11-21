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
	ErrCannotDeleteSelf = errors.New("cannot delete yourself")
	// ErrProtectedAdministrator：账户里的用户管理不能删除或降级实例管理员，
	// 也不能把账户降到没有管理员。
	ErrProtectedAdministrator = errors.New("administrator cannot be removed or demoted here")
	ErrManagedUserNotFound    = errors.New("managed user not found")
)

type UserManagementService struct {
	db *gorm.DB
}

func NewUserManagementService(db *gorm.DB) *UserManagementService {
	return &UserManagementService{db: db}
}

func (s *UserManagementService) List(accountID string, page, perPage int) ([]dto.UserManagementResponse, response.Meta, error) {
	query := s.db.Model(&models.User{}).
		Joins("JOIN account_memberships ON account_memberships.user_id = users.id").
		Where("account_memberships.account_id = ?", accountID)

	var total int64
	if err := query.Count(&total).Error; err != nil {
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
	if err := query.Select("users.*").Order("users.created_at ASC").Offset(offset).Limit(perPage).Find(&users).Error; err != nil {
		return nil, response.Meta{}, err
	}
	result := make([]dto.UserManagementResponse, len(users))
	for i, u := range users {
		result[i] = toUserManagementResponse(&u)
		if u.AccountID != accountID {
			var membership models.AccountMembership
			if err := s.db.Where("user_id = ? AND account_id = ?", u.ID, accountID).First(&membership).Error; err != nil {
				return nil, response.Meta{}, err
			}
			result[i].IsAdmin = membership.IsAdmin
		}
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
	if err := s.db.Joins("JOIN account_memberships ON account_memberships.user_id = users.id").
		Where("users.id = ? AND account_memberships.account_id = ?", id, accountID).
		Select("users.*").First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrManagedUserNotFound
		}
		return nil, err
	}
	if !req.IsAdmin {
		// 账户管理员只是这个账户的协管理员。让他降级实例管理员，等于让低权限的人
		// 拿掉高权限的人；把最后一个管理员降级，账户就再也没人能管理。
		if user.IsInstanceAdministrator {
			return nil, ErrProtectedAdministrator
		}
		var otherAdministrators int64
		if err := s.db.Model(&models.AccountMembership{}).
			Where("account_id = ? AND user_id <> ? AND is_admin = ?", accountID, id, true).
			Count(&otherAdministrators).Error; err != nil {
			return nil, err
		}
		if otherAdministrators == 0 {
			return nil, ErrProtectedAdministrator
		}
	}
	if user.AccountID == accountID {
		user.FirstName = strPtrOrNil(req.FirstName)
		user.LastName = strPtrOrNil(req.LastName)
		user.IsAccountAdministrator = req.IsAdmin
		if err := s.db.Save(&user).Error; err != nil {
			return nil, err
		}
	}
	if err := s.db.Model(&models.AccountMembership{}).
		Where("account_id = ? AND user_id = ?", accountID, id).
		Update("is_admin", req.IsAdmin).Error; err != nil {
		return nil, err
	}
	resp := toUserManagementResponse(&user)
	resp.IsAdmin = req.IsAdmin
	return &resp, nil
}

func (s *UserManagementService) Delete(id, accountID, currentUserID string) error {
	if id == currentUserID {
		return ErrCannotDeleteSelf
	}
	var user models.User
	if err := s.db.Joins("JOIN account_memberships ON account_memberships.user_id = users.id").
		Where("users.id = ? AND account_memberships.account_id = ?", id, accountID).
		Select("users.*").First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrManagedUserNotFound
		}
		return err
	}
	// 理由同 Update：协管理员不能把实例管理员删掉。
	if user.IsInstanceAdministrator {
		return ErrProtectedAdministrator
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		vaultIDs := tx.Model(&models.Vault{}).Select("id").Where("account_id = ?", accountID)
		var memberships []models.UserVault
		if err := tx.Where("user_id = ? AND vault_id IN (?) AND permission = ?", id, vaultIDs, models.PermissionManager).Find(&memberships).Error; err != nil {
			return err
		}
		for _, membership := range memberships {
			if err := ensureAnotherVaultManager(tx, membership.VaultID, membership.ID); err != nil {
				return err
			}
		}
		if err := tx.Where("user_id = ? AND vault_id IN (?)", id, vaultIDs).Delete(&models.UserVault{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ? AND account_id = ?", id, accountID).Delete(&models.AccountMembership{}).Error; err != nil {
			return err
		}
		if user.AccountID != accountID {
			return nil
		}
		var another models.AccountMembership
		if err := tx.Where("user_id = ?", id).First(&another).Error; err == nil {
			return tx.Model(&user).Updates(map[string]interface{}{"account_id": another.AccountID, "is_account_administrator": another.IsAdmin}).Error
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err := tx.Where("user_id = ?", id).Delete(&models.UserNotificationChannel{}).Error; err != nil {
			return err
		}
		var externalVaults int64
		if err := tx.Model(&models.UserVault{}).Where("user_id = ?", id).Count(&externalVaults).Error; err != nil {
			return err
		}
		if externalVaults != 0 {
			return ErrAccountHasExternalVaultMembers
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
