package services

import (
	"errors"
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
)

func makeAccountAdministrator(t *testing.T, svc *UserManagementService, accountID string, user models.User) {
	t.Helper()
	promoted := svc.db.Model(&models.AccountMembership{}).Where("account_id = ? AND user_id = ?", accountID, user.ID).Update("is_admin", true)
	if promoted.Error != nil || promoted.RowsAffected != 1 {
		t.Fatalf("promote membership: %v, %d row(s)", promoted.Error, promoted.RowsAffected)
	}
	if err := svc.db.Model(&models.User{}).Where("id = ?", user.ID).Update("is_account_administrator", true).Error; err != nil {
		t.Fatalf("mark account administrator: %v", err)
	}
}

// An account administrator is a co-administrator of one account. Account user
// management let them delete or demote the instance administrator.
func TestUserManagement_InstanceAdministratorCannotBeRemovedOrDemoted(t *testing.T) {
	svc, db, accountID, instanceAdministratorID := setupUserManagementTest(t)
	coAdministrator := createTestUser(t, db, accountID, "co-administrator@example.com")
	makeAccountAdministrator(t, svc, accountID, coAdministrator)

	if err := svc.Delete(instanceAdministratorID, accountID, coAdministrator.ID); !errors.Is(err, ErrProtectedAdministrator) {
		t.Errorf("deleting the instance administrator: err = %v, want ErrProtectedAdministrator", err)
	}
	if _, err := svc.Update(instanceAdministratorID, accountID, dto.UpdateManagedUserRequest{FirstName: "Demoted", IsAdmin: false}); !errors.Is(err, ErrProtectedAdministrator) {
		t.Errorf("demoting the instance administrator: err = %v, want ErrProtectedAdministrator", err)
	}

	var instanceAdministrator models.User
	if err := db.First(&instanceAdministrator, "id = ?", instanceAdministratorID).Error; err != nil {
		t.Fatalf("the instance administrator is gone: %v", err)
	}
	if !instanceAdministrator.IsAccountAdministrator || !instanceAdministrator.IsInstanceAdministrator {
		t.Errorf("administrator flags after the refused changes: account=%t instance=%t", instanceAdministrator.IsAccountAdministrator, instanceAdministrator.IsInstanceAdministrator)
	}
}

// An account without any administrator can no longer be managed by anyone.
func TestUserManagement_LastAccountAdministratorCannotBeDemoted(t *testing.T) {
	svc, db, accountID, onlyAdministratorID := setupUserManagementTest(t)
	if err := db.Model(&models.User{}).Where("id = ?", onlyAdministratorID).Update("is_instance_administrator", false).Error; err != nil {
		t.Fatalf("make the administrator an ordinary account administrator: %v", err)
	}

	if _, err := svc.Update(onlyAdministratorID, accountID, dto.UpdateManagedUserRequest{FirstName: "Admin", IsAdmin: false}); !errors.Is(err, ErrProtectedAdministrator) {
		t.Fatalf("demoting the only administrator: err = %v, want ErrProtectedAdministrator", err)
	}

	coAdministrator := createTestUser(t, db, accountID, "second-administrator@example.com")
	makeAccountAdministrator(t, svc, accountID, coAdministrator)
	updated, err := svc.Update(onlyAdministratorID, accountID, dto.UpdateManagedUserRequest{FirstName: "Admin", IsAdmin: false})
	if err != nil {
		t.Fatalf("demoting an administrator while another one remains: %v", err)
	}
	if updated.IsAdmin {
		t.Error("the administrator was not demoted although another administrator remains")
	}
}
