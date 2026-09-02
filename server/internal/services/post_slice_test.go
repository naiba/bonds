package services

import (
	"errors"
	"testing"
	"time"

	"github.com/naiba/bonds/internal/dto"
)

func TestPostSliceAssociationIsReturnedAndCanBeCleared(t *testing.T) {
	ctx := setupPostTestFull(t)
	post, err := ctx.svc.Create(ctx.journalID, ctx.vaultID, dto.CreatePostRequest{
		Title: "Linked post", WrittenAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("Create post failed: %v", err)
	}
	slice, err := NewSliceOfLifeService(ctx.db).Create(ctx.journalID, ctx.vaultID, dto.CreateSliceOfLifeRequest{Name: "Trip"})
	if err != nil {
		t.Fatalf("Create slice failed: %v", err)
	}

	if err := ctx.svc.SetSliceOfLife(post.ID, ctx.journalID, ctx.vaultID, slice.ID); err != nil {
		t.Fatalf("Set slice failed: %v", err)
	}
	linked, err := ctx.svc.Get(post.ID, ctx.journalID, ctx.vaultID)
	if err != nil {
		t.Fatalf("Get linked post failed: %v", err)
	}
	if linked.SliceOfLifeID == nil || *linked.SliceOfLifeID != slice.ID {
		t.Fatalf("SliceOfLifeID = %v, want %d", linked.SliceOfLifeID, slice.ID)
	}

	if err := ctx.svc.ClearSliceOfLife(post.ID, ctx.journalID, ctx.vaultID); err != nil {
		t.Fatalf("Clear slice failed: %v", err)
	}
	cleared, err := ctx.svc.Get(post.ID, ctx.journalID, ctx.vaultID)
	if err != nil {
		t.Fatalf("Get cleared post failed: %v", err)
	}
	if cleared.SliceOfLifeID != nil {
		t.Fatalf("SliceOfLifeID = %v, want nil", *cleared.SliceOfLifeID)
	}
}

func TestPostSliceAssociationRejectsSliceFromAnotherJournal(t *testing.T) {
	ctx := setupPostTestFull(t)
	post, err := ctx.svc.Create(ctx.journalID, ctx.vaultID, dto.CreatePostRequest{
		Title: "Linked post", WrittenAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("Create post failed: %v", err)
	}
	otherJournal, err := NewJournalService(ctx.db).Create(ctx.vaultID, dto.CreateJournalRequest{Name: "Other journal"})
	if err != nil {
		t.Fatalf("Create other journal failed: %v", err)
	}
	otherSlice, err := NewSliceOfLifeService(ctx.db).Create(otherJournal.ID, ctx.vaultID, dto.CreateSliceOfLifeRequest{Name: "Other slice"})
	if err != nil {
		t.Fatalf("Create other slice failed: %v", err)
	}

	err = ctx.svc.SetSliceOfLife(post.ID, ctx.journalID, ctx.vaultID, otherSlice.ID)
	if !errors.Is(err, ErrSliceOfLifeNotFound) {
		t.Fatalf("Set slice error = %v, want ErrSliceOfLifeNotFound", err)
	}
}
