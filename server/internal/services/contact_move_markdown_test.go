package services

import (
	"fmt"
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
)

func TestContactMoveKeepsMarkdownNoteFileReferencesInTargetVault(t *testing.T) {
	ctx := setupContactMoveLifecycle(t)
	contactID := ctx.contact.ID
	file := models.File{
		VaultID: ctx.sourceVaultID, UfileableID: &contactID, UUID: "moved-note-file",
		Name: "moved-note.pdf", MimeType: "application/pdf", Type: "document",
	}
	if err := ctx.db.Create(&file).Error; err != nil {
		t.Fatalf("create note file: %v", err)
	}
	body := fmt.Sprintf("[moved note](bonds-file:%d)", file.ID)
	if _, err := NewNoteService(ctx.db).Update(ctx.note.ID, contactID, ctx.sourceVaultID, dto.UpdateNoteRequest{
		Title: "Moving note", Body: body, BodyFormat: "markdown",
	}); err != nil {
		t.Fatalf("attach file to Markdown note: %v", err)
	}

	if _, err := NewContactMoveService(ctx.db).MoveMany(
		[]string{contactID}, ctx.sourceVaultID, ctx.targetVaultID, ctx.userID,
	); err != nil {
		t.Fatalf("move contact: %v", err)
	}

	var movedFile models.File
	if err := ctx.db.First(&movedFile, file.ID).Error; err != nil {
		t.Fatalf("load moved file: %v", err)
	}
	if movedFile.VaultID != ctx.targetVaultID {
		t.Fatalf("file vault = %q, want %q", movedFile.VaultID, ctx.targetVaultID)
	}
	var reference models.ContentFileReference
	if err := ctx.db.Where("owner_type = ? AND owner_id = ? AND file_id = ?", models.ContentOwnerNote, ctx.note.ID, file.ID).
		First(&reference).Error; err != nil {
		t.Fatalf("load moved content reference: %v", err)
	}
	if reference.VaultID != ctx.targetVaultID {
		t.Fatalf("reference vault = %q, want %q", reference.VaultID, ctx.targetVaultID)
	}
}
