package services

import (
	"math"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/pkg/response"
	"gorm.io/gorm"
)

type FeedService struct {
	db *gorm.DB
}

func NewFeedService(db *gorm.DB) *FeedService {
	return &FeedService{db: db}
}

func (s *FeedService) GetFeed(vaultID string, page, perPage int) ([]dto.FeedItemResponse, response.Meta, error) {
	var contacts []models.Contact
	if err := s.db.Where("vault_id = ?", vaultID).Select("id").Find(&contacts).Error; err != nil {
		return nil, response.Meta{}, err
	}
	contactIDs := make([]string, len(contacts))
	for i, c := range contacts {
		contactIDs[i] = c.ID
	}
	if len(contactIDs) == 0 {
		return []dto.FeedItemResponse{}, response.Meta{Page: 1, PerPage: perPage}, nil
	}

	query := s.db.Where("contact_id IN ?", contactIDs)

	var total int64
	if err := query.Model(&models.ContactFeedItem{}).Count(&total).Error; err != nil {
		return nil, response.Meta{}, err
	}

	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 15
	}
	offset := (page - 1) * perPage

	var items []models.ContactFeedItem
	if err := query.Offset(offset).Limit(perPage).Order("created_at DESC").Find(&items).Error; err != nil {
		return nil, response.Meta{}, err
	}

	result := make([]dto.FeedItemResponse, len(items))
	for i, item := range items {
		authorID := ""
		if item.AuthorID != nil {
			authorID = *item.AuthorID
		}
		desc := ""
		if item.Description != nil {
			desc = *item.Description
		}
		result[i] = dto.FeedItemResponse{
			ID:          item.ID,
			ContactID:   item.ContactID,
			AuthorID:    authorID,
			Action:      item.Action,
			Description: desc,
			CreatedAt:   item.CreatedAt,
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

func (s *FeedService) ListContactFeed(contactID string, page, perPage int) ([]dto.FeedItemResponse, response.Meta, error) {
	query := s.db.Where("contact_id = ?", contactID)

	var total int64
	if err := query.Model(&models.ContactFeedItem{}).Count(&total).Error; err != nil {
		return nil, response.Meta{}, err
	}

	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 15
	}
	offset := (page - 1) * perPage

	var items []models.ContactFeedItem
	if err := query.Offset(offset).Limit(perPage).Order("created_at DESC").Find(&items).Error; err != nil {
		return nil, response.Meta{}, err
	}

	result := make([]dto.FeedItemResponse, len(items))
	for i, item := range items {
		authorID := ""
		if item.AuthorID != nil {
			authorID = *item.AuthorID
		}
		desc := ""
		if item.Description != nil {
			desc = *item.Description
		}
		result[i] = dto.FeedItemResponse{
			ID:          item.ID,
			ContactID:   item.ContactID,
			AuthorID:    authorID,
			Action:      item.Action,
			Description: desc,
			CreatedAt:   item.CreatedAt,
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
