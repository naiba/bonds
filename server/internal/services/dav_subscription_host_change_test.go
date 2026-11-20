package services

import (
	"errors"
	"testing"

	"github.com/naiba/bonds/internal/dto"
)

// The stored remote password is sent to whatever host the subscription points
// at. Anyone allowed to edit the subscription could otherwise repoint it at a
// server they control and collect the creator's remote credentials.
func TestDavSubscriptionUpdate_RequiresPasswordWhenHostChanges(t *testing.T) {
	_, clientSvc, _, vaultID, userID, _ := setupDavSyncTest(t)
	sub, err := clientSvc.Create(vaultID, userID, dto.CreateDavSubscriptionRequest{
		URI: "https://dav.example.com/contacts/", Username: "user", Password: "remote-secret",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	_, err = clientSvc.Update(sub.ID, vaultID, dto.UpdateDavSubscriptionRequest{URI: "https://collector.example.net/contacts/"})
	if !errors.Is(err, ErrSubscriptionPasswordRequired) {
		t.Fatalf("host change without password: err = %v, want ErrSubscriptionPasswordRequired", err)
	}
	stored, password, err := clientSvc.GetDecryptedPassword(sub.ID, vaultID)
	if err != nil {
		t.Fatalf("reload subscription: %v", err)
	}
	if stored.URI != "https://dav.example.com/contacts/" || password != "remote-secret" {
		t.Fatalf("subscription after rejected update: uri=%q, want it unchanged", stored.URI)
	}

	if _, err := clientSvc.Update(sub.ID, vaultID, dto.UpdateDavSubscriptionRequest{URI: "https://DAV.example.com/other-book/"}); err != nil {
		t.Fatalf("path change on the same host must not need the password: %v", err)
	}
	if _, err := clientSvc.Update(sub.ID, vaultID, dto.UpdateDavSubscriptionRequest{
		URI: "https://new-server.example.org/contacts/", Password: "new-secret",
	}); err != nil {
		t.Fatalf("host change with a new password failed: %v", err)
	}
	stored, password, err = clientSvc.GetDecryptedPassword(sub.ID, vaultID)
	if err != nil || stored.URI != "https://new-server.example.org/contacts/" || password != "new-secret" {
		t.Fatalf("subscription after allowed update: uri=%q password=%q err=%v", stored.URI, password, err)
	}
}
