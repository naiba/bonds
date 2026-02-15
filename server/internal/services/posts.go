package services

import (
	"errors"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var ErrPostNotFound = errors.New("post not found")

type PostService struct {
	db *gorm.DB
}

func NewPostService(db *gorm.DB) *PostService {
	return &PostService{db: db}
}

func (s *PostService) List(journalID uint) ([]dto.PostResponse, error) {
	var posts []models.Post
	if err := s.db.Where("journal_id = ?", journalID).Order("written_at DESC").Find(&posts).Error; err != nil {
		return nil, err
	}
	result := make([]dto.PostResponse, len(posts))
	for i, p := range posts {
		result[i] = toPostResponse(&p)
	}
	return result, nil
}

func (s *PostService) Create(journalID uint, req dto.CreatePostRequest) (*dto.PostResponse, error) {
	post := models.Post{
		JournalID: journalID,
		Title:     strPtrOrNil(req.Title),
		Published: req.Published,
		WrittenAt: req.WrittenAt,
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&post).Error; err != nil {
			return err
		}
		for _, sec := range req.Sections {
			section := models.PostSection{
				PostID:   post.ID,
				Position: sec.Position,
				Label:    sec.Label,
				Content:  strPtrOrNil(sec.Content),
			}
			if err := tx.Create(&section).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return s.Get(post.ID, journalID)
}

func (s *PostService) Get(id uint, journalID uint) (*dto.PostResponse, error) {
	var post models.Post
	if err := s.db.Where("id = ? AND journal_id = ?", id, journalID).Preload("PostSections", func(db *gorm.DB) *gorm.DB {
		return db.Order("position ASC")
	}).First(&post).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPostNotFound
		}
		return nil, err
	}
	resp := toPostResponseWithSections(&post)
	return &resp, nil
}

func (s *PostService) Update(id uint, journalID uint, req dto.UpdatePostRequest) (*dto.PostResponse, error) {
	var post models.Post
	if err := s.db.Where("id = ? AND journal_id = ?", id, journalID).First(&post).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPostNotFound
		}
		return nil, err
	}
	post.Title = strPtrOrNil(req.Title)
	post.Published = req.Published
	if !req.WrittenAt.IsZero() {
		post.WrittenAt = req.WrittenAt
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&post).Error; err != nil {
			return err
		}
		if req.Sections != nil {
			if err := tx.Where("post_id = ?", id).Delete(&models.PostSection{}).Error; err != nil {
				return err
			}
			for _, sec := range req.Sections {
				section := models.PostSection{
					PostID:   post.ID,
					Position: sec.Position,
					Label:    sec.Label,
					Content:  strPtrOrNil(sec.Content),
				}
				if err := tx.Create(&section).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.Get(id, journalID)
}

func (s *PostService) Delete(id uint, journalID uint) error {
	var post models.Post
	if err := s.db.Where("id = ? AND journal_id = ?", id, journalID).First(&post).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPostNotFound
		}
		return err
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("post_id = ?", id).Delete(&models.PostSection{}).Error; err != nil {
			return err
		}
		return tx.Delete(&post).Error
	})
}

func toPostResponse(p *models.Post) dto.PostResponse {
	return dto.PostResponse{
		ID:        p.ID,
		JournalID: p.JournalID,
		Title:     ptrToStr(p.Title),
		Published: p.Published,
		WrittenAt: p.WrittenAt,
		ViewCount: p.ViewCount,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}

func toPostResponseWithSections(p *models.Post) dto.PostResponse {
	resp := toPostResponse(p)
	sections := make([]dto.PostSectionResponse, len(p.PostSections))
	for i, s := range p.PostSections {
		sections[i] = dto.PostSectionResponse{
			ID:       s.ID,
			Position: s.Position,
			Label:    s.Label,
			Content:  ptrToStr(s.Content),
		}
	}
	resp.Sections = sections
	return resp
}
