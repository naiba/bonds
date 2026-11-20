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

// findPostInVault 是所有 post-tag 操作的入口校验。
// 路由中间件只证明调用者是 :vault_id 的成员，路径里的 journal_id / post id 由调用者任填；
// 只按 "id = ? AND journal_id = ?" 查 post 的话，任何用户都能拿自己的 vault 作掩护，
// 枚举 ID 去读、改、删别的租户的帖子标签。必须先把 journal 绑回这个 vault。
func (s *PostTagService) findPostInVault(postID uint, journalID uint, vaultID string) (*models.Post, error) {
	if err := validateJournalBelongsToVault(s.db, journalID, vaultID); err != nil {
		return nil, err
	}
	var post models.Post
	if err := s.db.Where("id = ? AND journal_id = ?", postID, journalID).First(&post).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPostNotFound
		}
		return nil, err
	}
	return &post, nil
}

func (s *PostTagService) Add(postID uint, journalID uint, vaultID string, req dto.AddPostTagRequest) (*dto.PostTagResponse, error) {
	if _, err := s.findPostInVault(postID, journalID, vaultID); err != nil {
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

func (s *PostTagService) List(postID uint, journalID uint, vaultID string) ([]dto.PostTagResponse, error) {
	post, err := s.findPostInVault(postID, journalID, vaultID)
	if err != nil {
		return nil, err
	}

	var tags []models.Tag
	if err := s.db.Model(post).Association("Tags").Find(&tags); err != nil {
		return nil, err
	}

	resp := make([]dto.PostTagResponse, len(tags))
	for i, tag := range tags {
		resp[i] = dto.PostTagResponse{
			ID:   tag.ID,
			Name: tag.Name,
			Slug: tag.Slug,
		}
	}
	return resp, nil
}

func (s *PostTagService) Update(tagID uint, postID uint, journalID uint, vaultID string, req dto.UpdatePostTagRequest) (*dto.PostTagResponse, error) {
	if _, err := s.findPostInVault(postID, journalID, vaultID); err != nil {
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

func (s *PostTagService) Remove(tagID uint, postID uint, journalID uint, vaultID string) error {
	if _, err := s.findPostInVault(postID, journalID, vaultID); err != nil {
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
