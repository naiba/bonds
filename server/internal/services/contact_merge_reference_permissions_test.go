package services

import (
	"errors"
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
)

func TestMergeContactsAuthorizesOnlyChangedIncomingRelationships(t *testing.T) {
	for _, scenario := range []struct {
		name                                     string
		sourceReference, duplicateTarget, revoke bool
		permission                               int
		blocked                                  bool
	}{
		{name: "unchanged_target_viewer", permission: models.PermissionViewer},
		{name: "unchanged_target_no_membership"},
		{name: "source_reference_viewer", sourceReference: true, permission: models.PermissionViewer, blocked: true},
		{name: "duplicate_target_viewer", duplicateTarget: true, permission: models.PermissionViewer, blocked: true},
		{name: "duplicate_target_editor", duplicateTarget: true, permission: models.PermissionEditor},
		{name: "source_reference_editor", sourceReference: true, permission: models.PermissionEditor},
		{name: "source_permission_revoked", sourceReference: true, permission: models.PermissionEditor, revoke: true, blocked: true},
		{name: "duplicate_permission_revoked", duplicateTarget: true, permission: models.PermissionEditor, revoke: true, blocked: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			svc, vaultID, userID, accountID := setupContactTest(t)
			external, err := NewVaultService(svc.db).CreateVault(accountID, userID, dto.CreateVaultRequest{Name: "External relationships"}, "en")
			if err != nil {
				t.Fatal(err)
			}
			target := models.Contact{VaultID: vaultID, FirstName: strPtrOrNil("Alice")}
			source := models.Contact{VaultID: vaultID, FirstName: strPtrOrNil("Alice")}
			owner := models.Contact{VaultID: external.ID, FirstName: strPtrOrNil("Friend")}
			for _, c := range []*models.Contact{&target, &source, &owner} {
				if err := svc.db.Create(c).Error; err != nil {
					t.Fatal(err)
				}
			}
			var kind models.RelationshipType
			if err := svc.db.First(&kind).Error; err != nil {
				t.Fatal(err)
			}
			original := models.Relationship{ContactID: owner.ID, RelatedContactID: target.ID, RelationshipTypeID: kind.ID}
			if err := svc.db.Create(&original).Error; err != nil {
				t.Fatal(err)
			}
			// Compare persisted timestamps: PostgreSQL stores microsecond precision.
			if err := svc.db.First(&original, original.ID).Error; err != nil {
				t.Fatal(err)
			}
			extra := models.Relationship{ContactID: owner.ID, RelatedContactID: target.ID, RelationshipTypeID: kind.ID}
			if scenario.sourceReference {
				extra.RelatedContactID = source.ID
			}
			if scenario.sourceReference || scenario.duplicateTarget {
				if err := svc.db.Create(&extra).Error; err != nil {
					t.Fatal(err)
				}
			}
			membership := svc.db.Where("vault_id = ? AND user_id = ?", external.ID, userID)
			if scenario.permission == 0 {
				err = membership.Delete(&models.UserVault{}).Error
			} else {
				err = membership.Model(&models.UserVault{}).Update("permission", scenario.permission).Error
			}
			if err != nil {
				t.Fatal(err)
			}
			req := dto.MergeContactsRequest{TargetContactID: target.ID, SourceContactIDs: []string{source.ID}}
			preview, err := svc.PreviewContactMerge(vaultID, userID, req)
			if err != nil {
				t.Fatal(err)
			}
			if scenario.blocked && !scenario.revoke {
				if len(preview.Blockers) != 1 || preview.Blockers[0] != "incoming_permission" {
					t.Fatalf("expected incoming permission blocker, got %v", preview.Blockers)
				}
			} else if len(preview.Blockers) != 0 {
				t.Fatalf("unexpected blockers for %s: %v", scenario.name, preview.Blockers)
			}
			req.ReviewToken = preview.ReviewToken
			if scenario.revoke {
				if err := svc.db.Model(&models.UserVault{}).Where("vault_id = ? AND user_id = ?", external.ID, userID).Update("permission", models.PermissionViewer).Error; err != nil {
					t.Fatal(err)
				}
			}
			_, err = svc.MergeContacts(vaultID, userID, req)
			if scenario.blocked {
				if !errors.Is(err, ErrContactMergeBlocked) {
					t.Fatalf("expected permission rejection, got %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			var actual models.Relationship
			if err := svc.db.First(&actual, original.ID).Error; err != nil {
				t.Fatal(err)
			}
			if actual.ContactID != original.ContactID || actual.RelatedContactID != original.RelatedContactID || actual.RelationshipTypeID != original.RelationshipTypeID || !actual.UpdatedAt.Equal(original.UpdatedAt) {
				t.Fatalf("unchanged target relationship was modified: %+v", actual)
			}
			var sourceCount, relationshipCount int64
			if err := svc.db.Model(&models.Contact{}).Where("id = ?", source.ID).Count(&sourceCount).Error; err != nil {
				t.Fatal(err)
			}
			if err := svc.db.Model(&models.Relationship{}).Where("contact_id = ?", owner.ID).Count(&relationshipCount).Error; err != nil {
				t.Fatal(err)
			}
			expectedCount := int64(1)
			if scenario.blocked {
				if sourceCount != 1 {
					t.Fatal("blocked merge deleted source")
				}
				if extra.ID != 0 {
					expectedCount++
					var kept models.Relationship
					if err := svc.db.First(&kept, extra.ID).Error; err != nil {
						t.Fatal(err)
					}
					if kept.RelatedContactID != extra.RelatedContactID {
						t.Fatal("blocked merge redirected incoming edge")
					}
				}
			} else if sourceCount != 0 {
				t.Fatal("successful merge did not remove source")
			}
			if relationshipCount != expectedCount {
				t.Fatalf("relationships=%d want %d", relationshipCount, expectedCount)
			}
		})
	}
}
