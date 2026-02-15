package services

import (
	"errors"
	"strings"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var ErrPostTagNotFound = errors.New("post tag not found")
var ErrTagNotFound = errors.New("tag not found")

type PostTagService struct {
	db *gorm.DB
}

func NewPostTagService(db *gorm.DB) *PostTagService {
	return &PostTagService{db: db}
}

func (s *PostTagService) Add(postID uint, journalID uint, vaultID string, req dto.AddPostTagRequest) (*dto.PostTagResponse, error) {
	var post models.Post
	if err := s.db.Where("id = ? AND journal_id = ?", postID, journalID).First(&post).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPostNotFound
		}
		return nil, err
	}

	var tag models.Tag
	if req.TagID > 0 {
		if err := s.db.Where("id = ? AND vault_id = ?", req.TagID, vaultID).First(&tag).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrTagNotFound
			}
			return nil, err
		}
	} else if req.Name != "" {
		slug := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(req.Name), " ", "-"))
		tag = models.Tag{
			VaultID: vaultID,
			Name:    req.Name,
			Slug:    slug,
		}
		if err := s.db.Create(&tag).Error; err != nil {
			return nil, err
		}
	} else {
		return nil, errors.New("tag_id or name is required")
	}

	pt := models.PostTag{
		PostID: postID,
		TagID:  tag.ID,
	}
	if err := s.db.Create(&pt).Error; err != nil {
		return nil, err
	}

	resp := dto.PostTagResponse{
		ID:   tag.ID,
		Name: tag.Name,
		Slug: tag.Slug,
	}
	return &resp, nil
}

func (s *PostTagService) Update(tagID uint, postID uint, journalID uint, vaultID string, req dto.UpdatePostTagRequest) (*dto.PostTagResponse, error) {
	var post models.Post
	if err := s.db.Where("id = ? AND journal_id = ?", postID, journalID).First(&post).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPostNotFound
		}
		return nil, err
	}

	var tag models.Tag
	if err := s.db.Where("id = ? AND vault_id = ?", tagID, vaultID).First(&tag).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTagNotFound
		}
		return nil, err
	}

	tag.Name = req.Name
	tag.Slug = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(req.Name), " ", "-"))
	if err := s.db.Save(&tag).Error; err != nil {
		return nil, err
	}

	resp := dto.PostTagResponse{
		ID:   tag.ID,
		Name: tag.Name,
		Slug: tag.Slug,
	}
	return &resp, nil
}

func (s *PostTagService) Remove(tagID uint, postID uint, journalID uint) error {
	var post models.Post
	if err := s.db.Where("id = ? AND journal_id = ?", postID, journalID).First(&post).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPostNotFound
		}
		return err
	}
	result := s.db.Where("post_id = ? AND tag_id = ?", postID, tagID).Delete(&models.PostTag{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrPostTagNotFound
	}
	return nil
}
