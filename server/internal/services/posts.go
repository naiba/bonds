package services

import (
	"errors"
	"sort"

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
	// Omitted sections keep their stored mentions authoritative, even when a
	// caller explicitly sends an empty contact list. Do not detach live text.
	associationContactsNeedLocking := req.Sections == nil && (req.ContactIDs != nil || req.UpdateLastContacted)

	var associatedContactIDs []string
	if associationContactsNeedLocking {
		var err error
		associatedContactIDs, err = storedPostContactIDs(s.db, id, vaultID, req.ContactIDs)
		if err != nil {
			return nil, err
		}
	}

	for attempt := 0; attempt < maxPostUpdateAssociationLockAttempts; attempt++ {
		sections, err := postSectionsWithStoredFormats(s.db, id, req.Sections)
		if err != nil {
			return nil, err
		}
		if associationsProvided && !associationContactsNeedLocking {
			contactIDs, err = validateAndDedupeContactIDs(postContactIDsFromSections(sections, req.ContactIDs))
			if err != nil {
				return nil, err
			}
		}
		err = s.db.Transaction(func(tx *gorm.DB) error {
			lockedContactIDs := contactIDs
			if associationContactsNeedLocking {
				lockedContactIDs = associatedContactIDs
			}
			if err := lockContactsBelongToVault(tx, lockedContactIDs, vaultID); err != nil {
				// A merge can replace stored mentions while this metadata-only request
				// waits for its parent locks. Retry from current storage, never alias IDs
				// supplied in an old body or acquire new parents after journal locks.
				if associationContactsNeedLocking && errors.Is(err, ErrContactNotFound) {
					currentIDs, readErr := storedPostContactIDs(tx, id, vaultID, req.ContactIDs)
					if readErr != nil {
						return readErr
					}
					if !equalPostContactIDs(lockedContactIDs, currentIDs) {
						return errPostContactAssociationsChanged
					}
				}
				return err
			}
			if err := lockPostJournal(tx, journalID, vaultID); err != nil {
				return err
			}
			post, err := lockJournalPost(tx, id, journalID)
			if err != nil {
				return err
			}

			verifiedSections, err := postSectionsWithStoredFormats(tx, id, req.Sections)
			if err != nil {
				return err
			}
			for i := range sections {
				if sections[i].ContentFormat != verifiedSections[i].ContentFormat {
					return errPostContactAssociationsChanged
				}
			}

			contactIDsToAdvance := contactIDs
			if associationContactsNeedLocking {
				verifiedContactIDs, err := storedPostContactIDs(tx, post.ID, vaultID, req.ContactIDs)
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
				existingSectionIDs := make([]uint, len(existingSections))
				for index, section := range existingSections {
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
				for _, sec := range sections {
					contentFormat := sec.ContentFormat
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
				if err := createContactPostAssociations(tx, post.ID, contactIDsToAdvance); err != nil {
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
		associatedContactIDs, err = storedPostContactIDs(s.db, id, vaultID, req.ContactIDs)
		if err != nil {
			return nil, err
		}
	}
	return nil, errPostContactAssociationsChanged
}

// Omitted section formats retain their stored interpretation, including during
// association discovery. Recheck under the journal lock before saving a retry.
func postSectionsWithStoredFormats(db *gorm.DB, postID uint, sections []dto.PostSectionInput) ([]dto.PostSectionInput, error) {
	resolved := append([]dto.PostSectionInput(nil), sections...)
	needsStored := false
	for _, section := range resolved {
		if section.ContentFormat == "" {
			needsStored = true
		}
	}
	formats := map[int]string{}
	if needsStored {
		var stored []models.PostSection
		if err := db.Select("position", "content_format").Where("post_id = ?", postID).Find(&stored).Error; err != nil {
			return nil, err
		}
		for _, section := range stored {
			formats[section.Position] = section.ContentFormat
		}
	}
	for i := range resolved {
		if resolved[i].ContentFormat == "" {
			resolved[i].ContentFormat = formats[resolved[i].Position]
		}
		resolved[i].ContentFormat = markdown.NormalizeFormat(resolved[i].ContentFormat)
	}
	return resolved, nil
}

// postContactIDsFromSections makes stable inline markers authoritative. The
// explicit IDs remain a fallback for older clients whose text predates inline
// mentions, so upgrades do not silently discard existing associations.
func postContactIDsFromSections(sections []dto.PostSectionInput, fallback []string) []string {
	var markerIDs []string
	for _, section := range sections {
		markerIDs = append(markerIDs, contactMentionIDs(section.Content, section.ContentFormat)...)
	}
	if len(markerIDs) > 0 {
		return markerIDs
	}
	return fallback
}

// storedPostContactIDs is used both before parent locking and after journal/post
// locking. Comparing the effective IDs allows partial updates to retry safely
// when another editor or merge changes the body while they wait.
func storedPostContactIDs(db *gorm.DB, postID uint, vaultID string, fallback []string) ([]string, error) {
	// With neither sections nor contact_ids submitted, this is only advancing
	// existing associations. Historical prose may mention a moved/deleted contact;
	// do not revalidate that untouched text or write outside the current vault.
	// Use the same scoped lookup on every lock/retry pass. An explicit list,
	// including [], still requires validating the stored authoritative mentions.
	if fallback == nil {
		return inVaultPostContactIDs(db, postID, vaultID)
	}
	var sections []models.PostSection
	if err := db.Select("content", "content_format").Where("post_id = ?", postID).Find(&sections).Error; err != nil {
		return nil, err
	}
	inputs := make([]dto.PostSectionInput, len(sections))
	for i, section := range sections {
		inputs[i].Content = ptrToStr(section.Content)
		inputs[i].ContentFormat = section.ContentFormat
	}
	ids, err := validateAndDedupeContactIDs(postContactIDsFromSections(inputs, fallback))
	if err != nil {
		return nil, err
	}
	sort.Strings(ids)
	return ids, nil
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
