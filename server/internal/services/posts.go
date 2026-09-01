package services

import (
	"errors"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/markdown"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var ErrPostNotFound = errors.New("post not found")

type PostService struct {
	db        *gorm.DB
	uploadDir string
}

func NewPostService(db *gorm.DB) *PostService {
	return &PostService{db: db}
}

func (s *PostService) List(journalID uint, vaultID string) ([]dto.PostResponse, error) {
	if err := validateJournalBelongsToVault(s.db, journalID, vaultID); err != nil {
		return nil, err
	}
	var posts []models.Post
	if err := s.db.Where("journal_id = ?", journalID).Preload("Contacts", "vault_id = ?", vaultID).Order("written_at DESC").Find(&posts).Error; err != nil {
		return nil, err
	}
	result := make([]dto.PostResponse, len(posts))
	for i, p := range posts {
		result[i] = toPostResponse(&p)
	}
	return result, nil
}

func (s *PostService) Create(journalID uint, vaultID string, req dto.CreatePostRequest) (*dto.PostResponse, error) {
	post := models.Post{
		JournalID: journalID,
		Title:     strPtrOrNil(req.Title),
		Published: req.Published,
		WrittenAt: req.WrittenAt,
	}
	applyTimeCalendarFields(&post.CalendarType, &post.OriginalDay, &post.OriginalMonth, &post.OriginalYear,
		&post.WrittenAt, req.CalendarType, req.OriginalDay, req.OriginalMonth, req.OriginalYear)

	if err := validateJournalBelongsToVault(s.db, journalID, vaultID); err != nil {
		return nil, err
	}
	for _, section := range req.Sections {
		if !markdown.IsValidFormat(section.ContentFormat) {
			return nil, ErrInvalidContentFormat
		}
	}
	contactIDs, err := validateAndDedupeContactIDs(postContactIDsFromSections(req.Sections, req.ContactIDs))
	if err != nil {
		return nil, err
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := lockContactsBelongToVault(tx, contactIDs, vaultID); err != nil {
			return err
		}
		if err := lockPostJournal(tx, journalID, vaultID); err != nil {
			return err
		}
		if err := tx.Create(&post).Error; err != nil {
			return err
		}
		for _, sec := range req.Sections {
			section := models.PostSection{
				PostID:        post.ID,
				Position:      sec.Position,
				Label:         sec.Label,
				Content:       strPtrOrNil(sec.Content),
				ContentFormat: markdown.NormalizeFormat(sec.ContentFormat),
			}
			if err := tx.Create(&section).Error; err != nil {
				return err
			}
			if err := syncContentFileReferences(tx, vaultID, models.ContentOwnerPostSection, section.ID, sec.Content, section.ContentFormat); err != nil {
				return err
			}
		}
		if err := createContactPostAssociations(tx, post.ID, contactIDs); err != nil {
			return err
		}
		if !req.UpdateLastContacted {
			return nil
		}
		return advanceContactsLastTalkedTo(tx, postContactUpdate{
			vaultID:    vaultID,
			contactIDs: contactIDs,
			writtenAt:  post.WrittenAt,
		})
	})
	if err != nil {
		return nil, err
	}

	return s.Get(post.ID, journalID, vaultID)
}

func (s *PostService) Get(id uint, journalID uint, vaultID string) (*dto.PostResponse, error) {
	if err := validateJournalBelongsToVault(s.db, journalID, vaultID); err != nil {
		return nil, err
	}
	var post models.Post
	if err := s.db.Where("id = ? AND journal_id = ?", id, journalID).Preload("PostSections", func(db *gorm.DB) *gorm.DB {
		return db.Order("position ASC")
	}).Preload("Contacts", "vault_id = ?", vaultID).First(&post).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPostNotFound
		}
		return nil, err
	}
	// Updating the preloaded post would make GORM persist Contacts again.
	if err := s.db.Model(&models.Post{}).Where("id = ?", post.ID).Update("view_count", post.ViewCount+1).Error; err != nil {
		return nil, err
	}
	post.ViewCount++

	resp := toPostResponseWithSections(&post)
	return &resp, nil
}

func (s *PostService) Update(id uint, journalID uint, vaultID string, req dto.UpdatePostRequest) (*dto.PostResponse, error) {
	if err := validateJournalBelongsToVault(s.db, journalID, vaultID); err != nil {
		return nil, err
	}
	if err := validatePostBelongsToJournal(s.db, id, journalID); err != nil {
		return nil, err
	}
	for _, section := range req.Sections {
		if !markdown.IsValidFormat(section.ContentFormat) {
			return nil, ErrInvalidContentFormat
		}
	}
	var contactIDs []string
	associationsProvided := req.Sections != nil || req.ContactIDs != nil
	if associationsProvided {
		var err error
		contactIDs, err = validateAndDedupeContactIDs(postContactIDsFromSections(req.Sections, req.ContactIDs))
		if err != nil {
			return nil, err
		}
	}

	associationContactsNeedLocking := !associationsProvided && req.UpdateLastContacted
	var associatedContactIDs []string
	if associationContactsNeedLocking {
		var err error
		associatedContactIDs, err = inVaultPostContactIDs(s.db, id, vaultID)
		if err != nil {
			return nil, err
		}
	}

	for attempt := 0; attempt < maxPostUpdateAssociationLockAttempts; attempt++ {
		err := s.db.Transaction(func(tx *gorm.DB) error {
			lockedContactIDs := contactIDs
			if associationContactsNeedLocking {
				lockedContactIDs = associatedContactIDs
			}
			if err := lockContactsBelongToVault(tx, lockedContactIDs, vaultID); err != nil {
				return err
			}
			if err := lockPostJournal(tx, journalID, vaultID); err != nil {
				return err
			}
			post, err := lockJournalPost(tx, id, journalID)
			if err != nil {
				return err
			}

			contactIDsToAdvance := contactIDs
			if associationContactsNeedLocking {
				verifiedContactIDs, err := inVaultPostContactIDs(tx, post.ID, vaultID)
				if err != nil {
					return err
				}
				// Associations can change while waiting for Journal/Post locks; retry instead of updating an unlocked contact.
				if !equalPostContactIDs(lockedContactIDs, verifiedContactIDs) {
					return errPostContactAssociationsChanged
				}
				contactIDsToAdvance = verifiedContactIDs
			}

			post.Title = strPtrOrNil(req.Title)
			post.Published = req.Published
			if !req.WrittenAt.IsZero() {
				post.WrittenAt = req.WrittenAt
			}
			applyTimeCalendarFields(&post.CalendarType, &post.OriginalDay, &post.OriginalMonth, &post.OriginalYear,
				&post.WrittenAt, req.CalendarType, req.OriginalDay, req.OriginalMonth, req.OriginalYear)

			if err := tx.Save(post).Error; err != nil {
				return err
			}
			if req.Sections != nil {
				var existingSections []models.PostSection
				if err := tx.Where("post_id = ?", id).Find(&existingSections).Error; err != nil {
					return err
				}
				existingFormats := make(map[int]string, len(existingSections))
				existingSectionIDs := make([]uint, len(existingSections))
				for index, section := range existingSections {
					existingFormats[section.Position] = markdown.NormalizeFormat(section.ContentFormat)
					existingSectionIDs[index] = section.ID
				}
				if len(existingSectionIDs) > 0 {
					if err := tx.Where("owner_type = ? AND owner_id IN ?", models.ContentOwnerPostSection, existingSectionIDs).
						Delete(&models.ContentFileReference{}).Error; err != nil {
						return err
					}
				}
				if err := tx.Where("post_id = ?", id).Delete(&models.PostSection{}).Error; err != nil {
					return err
				}
				for _, sec := range req.Sections {
					contentFormat := sec.ContentFormat
					if contentFormat == "" {
						contentFormat = existingFormats[sec.Position]
					}
					section := models.PostSection{
						PostID:        post.ID,
						Position:      sec.Position,
						Label:         sec.Label,
						Content:       strPtrOrNil(sec.Content),
						ContentFormat: markdown.NormalizeFormat(contentFormat),
					}
					if err := tx.Create(&section).Error; err != nil {
						return err
					}
					if err := syncContentFileReferences(tx, vaultID, models.ContentOwnerPostSection, section.ID, sec.Content, section.ContentFormat); err != nil {
						return err
					}
				}
			}
			if associationsProvided {
				if err := tx.Where("post_id = ?", id).Delete(&models.ContactPost{}).Error; err != nil {
					return err
				}
				if err := createContactPostAssociations(tx, post.ID, contactIDs); err != nil {
					return err
				}
			}
			if !req.UpdateLastContacted {
				return nil
			}
			return advanceContactsLastTalkedTo(tx, postContactUpdate{
				vaultID:    vaultID,
				contactIDs: contactIDsToAdvance,
				writtenAt:  post.WrittenAt,
			})
		})
		if !errors.Is(err, errPostContactAssociationsChanged) {
			if err != nil {
				return nil, err
			}
			return s.Get(id, journalID, vaultID)
		}
		if attempt == maxPostUpdateAssociationLockAttempts-1 {
			return nil, err
		}
		associatedContactIDs, err = inVaultPostContactIDs(s.db, id, vaultID)
		if err != nil {
			return nil, err
		}
	}
	return nil, errPostContactAssociationsChanged
}

// postContactIDsFromSections makes stable inline markers authoritative. The
// explicit IDs remain a fallback for older clients whose text predates inline
// mentions, so upgrades do not silently discard existing associations.
func postContactIDsFromSections(sections []dto.PostSectionInput, fallback []string) []string {
	var markerIDs []string
	for _, section := range sections {
		markerIDs = append(markerIDs, contactMentionIDs(section.Content)...)
	}
	if len(markerIDs) > 0 {
		return markerIDs
	}
	return fallback
}

func validateJournalBelongsToVault(db *gorm.DB, journalID uint, vaultID string) error {
	var journal models.Journal
	if err := db.Where("id = ? AND vault_id = ?", journalID, vaultID).First(&journal).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrJournalNotFound
		}
		return err
	}
	return nil
}
