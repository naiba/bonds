package services

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

type DavPushService struct {
	db            *gorm.DB
	clientService *DavClientService
	vcardService  *VCardService
	clientFactory CardDAVClientFactory
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
	subs, err := s.findPushSubscriptions(vaultID)
	if err != nil {
		log.Printf("[dav-push] failed to find push subscriptions for vault %s: %v", vaultID, err)
		return
	}
	if len(subs) == 0 {
		return
	}

	card, err := s.vcardService.ExportContactToVCard(contactID, vaultID)
	if err != nil {
		log.Printf("[dav-push] failed to export contact %s to vCard: %v", contactID, err)
		return
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

			client, err := s.clientFactory.NewClient(sub.URI, sub.Username, password)
			if err != nil {
				s.logPushAction(sub.ID, &contactID, "", "", "error", fmt.Sprintf("create client failed: %v", err))
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			var state models.ContactSubscriptionState
			hasState := s.db.Where("contact_id = ? AND address_book_subscription_id = ?", contactID, sub.ID).First(&state).Error == nil

			var putPath string
			if hasState {
				putPath = state.DistantURI
			} else {
				putPath = strings.TrimRight(sub.URI, "/") + "/" + contactID + ".vcf"
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

			if hasState {
				s.db.Model(&state).Updates(map[string]interface{}{
					"distant_uri":  resultPath,
					"distant_etag": resultEtag,
				})
			} else {
				s.db.Create(&models.ContactSubscriptionState{
					ContactID:                 contactID,
					AddressBookSubscriptionID: sub.ID,
					DistantURI:                resultPath,
					DistantEtag:               resultEtag,
				})
			}

			s.logPushAction(sub.ID, &contactID, resultPath, resultEtag, "pushed", "")
		}()
	}
}

func (s *DavPushService) PushContactDelete(contactID, vaultID string) {
	var states []models.ContactSubscriptionState
	s.db.Where("contact_id = ?", contactID).Find(&states)
	if len(states) == 0 {
		return
	}

	for _, state := range states {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[dav-push] panic deleting contact %s from subscription %s: %v", contactID, state.AddressBookSubscriptionID, r)
				}
			}()

			var sub models.AddressBookSubscription
			if err := s.db.Where("id = ? AND active = ? AND (sync_way & ?) != 0", state.AddressBookSubscriptionID, true, SyncWayPush).First(&sub).Error; err != nil {
				s.db.Delete(&state)
				return
			}

			password, err := s.clientService.decryptPassword(sub.Password)
			if err != nil {
				s.logPushAction(sub.ID, &contactID, state.DistantURI, "", "error", fmt.Sprintf("decrypt password failed: %v", err))
				return
			}

			client, err := s.clientFactory.NewClient(sub.URI, sub.Username, password)
			if err != nil {
				s.logPushAction(sub.ID, &contactID, state.DistantURI, "", "error", fmt.Sprintf("create client failed: %v", err))
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			if err := client.RemoveAll(ctx, state.DistantURI); err != nil {
				s.logPushAction(sub.ID, &contactID, state.DistantURI, "", "error", fmt.Sprintf("DELETE failed: %v", err))
				return
			}

			s.db.Delete(&state)
			s.logPushAction(sub.ID, &contactID, state.DistantURI, "", "push_deleted", "")
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
