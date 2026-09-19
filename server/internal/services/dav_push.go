package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

type DavPushService struct {
	db             *gorm.DB
	clientService  *DavClientService
	vcardService   *VCardService
	clientFactory  CardDAVClientFactory
	operationLocks contactDAVOperationLockRegistry
}

type contactRemoteDeletionTarget struct {
	contactID      string
	subscriptionID string
	distantURI     string
}

func newContactRemoteDeletionTarget(state models.ContactSubscriptionState) contactRemoteDeletionTarget {
	return contactRemoteDeletionTarget{
		contactID:      state.ContactID,
		subscriptionID: state.AddressBookSubscriptionID,
		distantURI:     state.DistantURI,
	}
}

func NewDavPushService(db *gorm.DB, clientService *DavClientService, vcardService *VCardService) *DavPushService {
	return &DavPushService{
		db:            db,
		clientService: clientService,
		vcardService:  vcardService,
		clientFactory: &DefaultCardDAVClientFactory{},
	}
}

func (s *DavPushService) SetClientFactory(factory CardDAVClientFactory) {
	s.clientFactory = factory
}

func (s *DavPushService) findPushSubscriptions(vaultID string) ([]models.AddressBookSubscription, error) {
	var subs []models.AddressBookSubscription
	err := s.db.Where("vault_id = ? AND active = ? AND (sync_way & ?) != 0", vaultID, true, SyncWayPush).Find(&subs).Error
	return subs, err
}

func (s *DavPushService) PushContactChange(contactID, vaultID string) {
	release := s.operationLocks.lock(contactID)
	defer release()
	s.pushContactChange(contactID, vaultID)
}

// pushContactChange requires the caller to hold contactID's DAV operation lock.
func (s *DavPushService) pushContactChange(contactID, vaultID string) {
	subs, err := s.findPushSubscriptions(vaultID)
	if err != nil {
		log.Printf("[dav-push] failed to find push subscriptions for vault %s: %v", vaultID, err)
		return
	}
	if len(subs) == 0 {
		return
	}

	var contact models.Contact
	if err := s.db.Where("id = ?", contactID).First(&contact).Error; err != nil {
		log.Printf("[dav-push] contact %s not found: %v", contactID, err)
		return
	}

	card, err := s.vcardService.ExportContactToVCard(contactID, vaultID)
	if err != nil {
		log.Printf("[dav-push] failed to export contact %s to vCard: %v", contactID, err)
		return
	}
	if contact.DistantUUID != nil && strings.TrimSpace(*contact.DistantUUID) != "" {
		// A CardDAV UID is stable even when the object is updated in place.
		card.SetValue("UID", *contact.DistantUUID)
	}

	for _, sub := range subs {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[dav-push] panic pushing contact %s to subscription %s: %v", contactID, sub.ID, r)
				}
			}()

			password, err := s.clientService.decryptPassword(sub.Password)
			if err != nil {
				s.logPushAction(sub.ID, &contactID, "", "", "error", fmt.Sprintf("decrypt password failed: %v", err))
				return
			}

			client, err := s.clientFactory.NewClient(sub.URI, sub.Username, password, DavTLSConfig{CustomCAPEM: sub.CustomCAPEM, SkipTLSVerify: sub.SkipTLSVerify})
			if err != nil {
				s.logPushAction(sub.ID, &contactID, "", "", "error", fmt.Sprintf("create client failed: %v", err))
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			var state models.ContactSubscriptionState
			stateErr := s.db.Where("contact_id = ? AND address_book_subscription_id = ?", contactID, sub.ID).First(&state).Error
			hasState := stateErr == nil
			if stateErr != nil && !errors.Is(stateErr, gorm.ErrRecordNotFound) {
				s.logPushAction(sub.ID, &contactID, "", "", "error", fmt.Sprintf("load subscription state failed: %v", stateErr))
				return
			}

			var putPath string
			if hasState {
				putPath = state.DistantURI
			} else if contact.DistantURI != nil && distantURIIsWithinSubscription(*contact.DistantURI, &sub) {
				// Compatibility for contacts pulled before pull-side subscription states
				// were recorded. Update the original CardDAV object in place.
				putPath = *contact.DistantURI
			} else {
				basePath := sub.AddressBookPath
				if basePath == "" {
					basePath = sub.URI
				}
				putPath = strings.TrimRight(basePath, "/") + "/" + contactID + ".vcf"
			}

			result, err := client.PutAddressObject(ctx, putPath, card)
			if err != nil {
				s.logPushAction(sub.ID, &contactID, putPath, "", "error", fmt.Sprintf("PUT failed: %v", err))
				return
			}

			var resultPath string
			var resultEtag string
			if result != nil {
				resultPath = result.Path
				resultEtag = result.ETag
			}
			if resultPath == "" {
				resultPath = putPath
			}

			if err := upsertContactSubscriptionState(s.db, contactID, sub.ID, resultPath, resultEtag); err != nil {
				s.logPushAction(sub.ID, &contactID, resultPath, resultEtag, "error", fmt.Sprintf("save subscription state failed: %v", err))
				return
			}

			s.logPushAction(sub.ID, &contactID, resultPath, resultEtag, "pushed", "")
		}()
	}
}

func distantURIIsWithinSubscription(distantURI string, sub *models.AddressBookSubscription) bool {
	for _, base := range []string{sub.AddressBookPath, sub.URI} {
		if davURIHasBase(distantURI, base) {
			return true
		}
	}
	return false
}

func davURIHasBase(rawURI, rawBase string) bool {
	if rawURI == "" || rawBase == "" {
		return false
	}

	parsedURI, uriErr := url.Parse(rawURI)
	parsedBase, baseErr := url.Parse(rawBase)
	if uriErr != nil || baseErr != nil {
		return false
	}
	if parsedURI.Host != "" && parsedBase.Host != "" && !strings.EqualFold(parsedURI.Host, parsedBase.Host) {
		return false
	}

	uriPath := parsedURI.Path
	basePath := parsedBase.Path
	if uriPath == "" {
		if parsedURI.Host != "" {
			uriPath = "/"
		} else {
			uriPath = rawURI
		}
	}
	if basePath == "" {
		if parsedBase.Host != "" {
			basePath = "/"
		} else {
			basePath = rawBase
		}
	}
	basePath = strings.TrimRight(basePath, "/")
	if basePath == "" {
		return strings.HasPrefix(uriPath, "/")
	}
	return uriPath == basePath || strings.HasPrefix(uriPath, basePath+"/")
}

func (s *DavPushService) PushContactDelete(contactID, vaultID string) {
	release := s.operationLocks.lock(contactID)
	defer release()

	var states []models.ContactSubscriptionState
	if err := s.db.Where("contact_id = ?", contactID).Find(&states).Error; err != nil {
		log.Printf("[dav-push] failed to find delete states for contact %s: %v", contactID, err)
		return
	}
	if len(states) == 0 {
		return
	}
	targets := make([]contactRemoteDeletionTarget, len(states))
	for index := range states {
		targets[index] = newContactRemoteDeletionTarget(states[index])
	}
	s.pushContactDeleteTargets(targets, true)
}

func (s *DavPushService) pushCapturedContactDelete(targets []contactRemoteDeletionTarget) {
	s.pushContactDeleteTargets(targets, false)
}

func (s *DavPushService) pushContactDeleteTargets(targets []contactRemoteDeletionTarget, deleteLocalState bool) {
	for _, target := range targets {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[dav-push] panic deleting contact %s from subscription %s: %v", target.contactID, target.subscriptionID, r)
				}
			}()

			var sub models.AddressBookSubscription
			if err := s.db.Where("id = ? AND active = ? AND (sync_way & ?) != 0", target.subscriptionID, true, SyncWayPush).First(&sub).Error; err != nil {
				if deleteLocalState {
					if err := s.db.Where("contact_id = ? AND address_book_subscription_id = ?", target.contactID, target.subscriptionID).Delete(&models.ContactSubscriptionState{}).Error; err != nil {
						log.Printf("[dav-push] failed to delete stale state for contact %s: %v", target.contactID, err)
					}
				}
				return
			}

			password, err := s.clientService.decryptPassword(sub.Password)
			if err != nil {
				s.logPushAction(sub.ID, &target.contactID, target.distantURI, "", "error", fmt.Sprintf("decrypt password failed: %v", err))
				return
			}

			client, err := s.clientFactory.NewClient(sub.URI, sub.Username, password, DavTLSConfig{CustomCAPEM: sub.CustomCAPEM, SkipTLSVerify: sub.SkipTLSVerify})
			if err != nil {
				s.logPushAction(sub.ID, &target.contactID, target.distantURI, "", "error", fmt.Sprintf("create client failed: %v", err))
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			if err := client.RemoveAll(ctx, target.distantURI); err != nil {
				s.logPushAction(sub.ID, &target.contactID, target.distantURI, "", "error", fmt.Sprintf("DELETE failed: %v", err))
				return
			}

			if deleteLocalState {
				if err := s.db.Where("contact_id = ? AND address_book_subscription_id = ?", target.contactID, target.subscriptionID).Delete(&models.ContactSubscriptionState{}).Error; err != nil {
					log.Printf("[dav-push] failed to delete state for contact %s: %v", target.contactID, err)
					return
				}
			}
			s.logPushAction(sub.ID, &target.contactID, target.distantURI, "", "push_deleted", "")
		}()
	}
}

func (s *DavPushService) logPushAction(subID string, contactID *string, distantURI, distantEtag, action, errMsg string) {
	logEntry := models.DavSyncLog{
		AddressBookSubscriptionID: subID,
		ContactID:                 contactID,
		DistantURI:                distantURI,
		DistantEtag:               distantEtag,
		Action:                    action,
	}
	if errMsg != "" {
		logEntry.ErrorMessage = &errMsg
	}
	if err := s.db.Create(&logEntry).Error; err != nil {
		log.Printf("[dav-push] failed to create sync log: %v", err)
	}
}
