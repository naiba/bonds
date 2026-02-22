package services

import (
	"context"
	"fmt"
	"log"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/emersion/go-vcard"
	"github.com/emersion/go-webdav/carddav"
	"github.com/icholy/digest"
	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/pkg/response"
	"gorm.io/gorm"
)

const (
	SyncWayPush = 0x1
	SyncWayPull = 0x2
	SyncWayBoth = 0x3
)

type CardDAVClient interface {
	FindCurrentUserPrincipal(ctx context.Context) (string, error)
	FindAddressBookHomeSet(ctx context.Context, principal string) (string, error)
	FindAddressBooks(ctx context.Context, homeSet string) ([]carddav.AddressBook, error)
	SyncCollection(ctx context.Context, path string, query *carddav.SyncQuery) (*carddav.SyncResponse, error)
	MultiGetAddressBook(ctx context.Context, path string, mg *carddav.AddressBookMultiGet) ([]carddav.AddressObject, error)
	QueryAddressBook(ctx context.Context, path string, query *carddav.AddressBookQuery) ([]carddav.AddressObject, error)
	GetAddressObject(ctx context.Context, path string) (*carddav.AddressObject, error)
	PutAddressObject(ctx context.Context, path string, card vcard.Card) (*carddav.AddressObject, error)
	RemoveAll(ctx context.Context, path string) error
}

type CardDAVClientFactory interface {
	NewClient(uri, username, password string) (CardDAVClient, error)
}

type DefaultCardDAVClientFactory struct{}

func (f *DefaultCardDAVClientFactory) NewClient(uri, username, password string) (CardDAVClient, error) {
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &fallbackAuthTransport{
			username: username,
			password: password,
		},
	}
	return carddav.NewClient(httpClient, uri)
}

type fallbackAuthTransport struct {
	username   string
	password   string
	useDigest  bool
	digestOnce sync.Once
	digestT    *digest.Transport
}

func (t *fallbackAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.useDigest {
		return t.getDigestTransport().RoundTrip(req)
	}

	req.SetBasicAuth(t.username, t.password)
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusUnauthorized {
		wwwAuth := resp.Header.Get("Www-Authenticate")
		if _, parseErr := digest.ParseChallenge(wwwAuth); parseErr == nil {
			resp.Body.Close()
			t.useDigest = true
			newReq := req.Clone(req.Context())
			newReq.Header.Del("Authorization")
			return t.getDigestTransport().RoundTrip(newReq)
		}
	}
	return resp, nil
}

func (t *fallbackAuthTransport) getDigestTransport() *digest.Transport {
	t.digestOnce.Do(func() {
		t.digestT = &digest.Transport{
			Username: t.username,
			Password: t.password,
		}
	})
	return t.digestT
}

type DavSyncService struct {
	db            *gorm.DB
	clientService *DavClientService
	vcardService  *VCardService
	clientFactory CardDAVClientFactory
}

func NewDavSyncService(db *gorm.DB, clientService *DavClientService, vcardService *VCardService) *DavSyncService {
	return &DavSyncService{
		db:            db,
		clientService: clientService,
		vcardService:  vcardService,
		clientFactory: &DefaultCardDAVClientFactory{},
	}
}

func (s *DavSyncService) SetClientFactory(factory CardDAVClientFactory) {
	s.clientFactory = factory
}

func (s *DavSyncService) TestConnection(req dto.TestDavConnectionRequest) (*dto.TestDavConnectionResponse, error) {
	client, err := s.clientFactory.NewClient(req.URI, req.Username, req.Password)
	if err != nil {
		return &dto.TestDavConnectionResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to create client: %v", err),
		}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Step 1: Discover principal (SabreDAV/Baikal requires this)
	principal, principalErr := client.FindCurrentUserPrincipal(ctx)
	if principalErr != nil {
		// Fallback: try empty path for simple servers
		principal = ""
	}

	// Step 2: Find address book home set using principal
	homeSet, err := client.FindAddressBookHomeSet(ctx, principal)
	if err != nil {
		return &dto.TestDavConnectionResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to find address book home set: %v", err),
		}, nil
	}

	books, err := client.FindAddressBooks(ctx, homeSet)
	if err != nil {
		return &dto.TestDavConnectionResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to list address books: %v", err),
		}, nil
	}

	names := make([]string, len(books))
	for i, b := range books {
		names[i] = b.Name
	}

	return &dto.TestDavConnectionResponse{
		Success:      true,
		AddressBooks: names,
	}, nil
}

func (s *DavSyncService) SyncSubscription(ctx context.Context, subID, vaultID string) (*dto.TriggerSyncResponse, error) {
	sub, password, err := s.clientService.GetDecryptedPassword(subID, vaultID)
	if err != nil {
		return nil, err
	}

	client, err := s.clientFactory.NewClient(sub.URI, sub.Username, password)
	if err != nil {
		return nil, fmt.Errorf("failed to create CardDAV client: %w", err)
	}

	var vault models.Vault
	if err := s.db.First(&vault, "id = ?", vaultID).Error; err != nil {
		return nil, fmt.Errorf("vault not found: %w", err)
	}
	accountID := vault.AccountID
	userID := sub.UserID

	result := &dto.TriggerSyncResponse{}

	hasToken := sub.DistantSyncToken != nil && *sub.DistantSyncToken != ""
	if hasToken {
		syncResp, syncErr := client.SyncCollection(ctx, sub.URI, &carddav.SyncQuery{
			DataRequest: carddav.AddressDataRequest{AllProp: true},
			SyncToken:   *sub.DistantSyncToken,
		})
		if syncErr == nil {
			s.processIncrementalSync(ctx, client, syncResp, sub, vaultID, userID, accountID, result)
			if err := s.clientService.UpdateSyncStatus(sub.ID, &syncResp.SyncToken); err != nil {
				log.Printf("[dav-sync] failed to update sync status: %v", err)
			}
			return result, nil
		}
		log.Printf("[dav-sync] incremental sync failed for sub %s, falling back to full sync: %v", sub.ID, syncErr)
	}

	s.performFullSync(ctx, client, sub, vaultID, userID, accountID, result)

	return result, nil
}

func (s *DavSyncService) processIncrementalSync(
	ctx context.Context,
	client CardDAVClient,
	syncResp *carddav.SyncResponse,
	sub *models.AddressBookSubscription,
	vaultID, userID, accountID string,
	result *dto.TriggerSyncResponse,
) {
	if len(syncResp.Updated) > 0 {
		paths := make([]string, len(syncResp.Updated))
		for i, obj := range syncResp.Updated {
			paths[i] = obj.Path
		}

		for i := 0; i < len(paths); i += 50 {
			end := i + 50
			if end > len(paths) {
				end = len(paths)
			}
			batch := paths[i:end]

			objects, err := client.MultiGetAddressBook(ctx, sub.URI, &carddav.AddressBookMultiGet{
				Paths:       batch,
				DataRequest: carddav.AddressDataRequest{AllProp: true},
			})
			if err != nil {
				log.Printf("[dav-sync] MultiGet failed for batch: %v", err)
				for _, p := range batch {
					s.logSyncAction(sub.ID, nil, p, "", "error", fmt.Sprintf("MultiGet failed: %v", err))
					result.Errors++
				}
				continue
			}

			for _, obj := range objects {
				if obj.Card == nil {
					continue
				}
				s.upsertFromObject(obj, sub.ID, vaultID, userID, accountID, sub.LastSynchronizedAt, result)
			}
		}
	}

	if len(syncResp.Deleted) > 0 {
		s.processDeletedPaths(syncResp.Deleted, sub.ID, vaultID, result)
	}
}

func (s *DavSyncService) performFullSync(
	ctx context.Context,
	client CardDAVClient,
	sub *models.AddressBookSubscription,
	vaultID, userID, accountID string,
	result *dto.TriggerSyncResponse,
) {
	syncResp, syncErr := client.SyncCollection(ctx, sub.URI, &carddav.SyncQuery{
		DataRequest: carddav.AddressDataRequest{AllProp: true},
		SyncToken:   "",
	})

	if syncErr == nil && len(syncResp.Updated) > 0 {
		paths := make([]string, len(syncResp.Updated))
		etagMap := make(map[string]string)
		for i, obj := range syncResp.Updated {
			paths[i] = obj.Path
			etagMap[obj.Path] = obj.ETag
		}

		for i := 0; i < len(paths); i += 50 {
			end := i + 50
			if end > len(paths) {
				end = len(paths)
			}
			batch := paths[i:end]

			objects, err := client.MultiGetAddressBook(ctx, sub.URI, &carddav.AddressBookMultiGet{
				Paths:       batch,
				DataRequest: carddav.AddressDataRequest{AllProp: true},
			})
			if err != nil {
				log.Printf("[dav-sync] MultiGet failed for batch: %v", err)
				for _, p := range batch {
					s.logSyncAction(sub.ID, nil, p, "", "error", fmt.Sprintf("MultiGet failed: %v", err))
					result.Errors++
				}
				continue
			}

			for _, obj := range objects {
				if obj.Card == nil {
					continue
				}
				s.upsertFromObject(obj, sub.ID, vaultID, userID, accountID, sub.LastSynchronizedAt, result)
			}
		}

		if err := s.clientService.UpdateSyncStatus(sub.ID, &syncResp.SyncToken); err != nil {
			log.Printf("[dav-sync] failed to update sync status: %v", err)
		}
		return
	}

	if syncErr != nil {
		log.Printf("[dav-sync] SyncCollection failed, falling back to QueryAddressBook: %v", syncErr)
	}

	objects, err := client.QueryAddressBook(ctx, sub.URI, &carddav.AddressBookQuery{
		DataRequest: carddav.AddressDataRequest{AllProp: true},
	})
	if err != nil {
		log.Printf("[dav-sync] QueryAddressBook failed: %v", err)
		s.logSyncAction(sub.ID, nil, sub.URI, "", "error", fmt.Sprintf("QueryAddressBook failed: %v", err))
		result.Errors++
		return
	}

	for _, obj := range objects {
		if obj.Card == nil {
			continue
		}
		s.upsertFromObject(obj, sub.ID, vaultID, userID, accountID, sub.LastSynchronizedAt, result)
	}

	if err := s.clientService.UpdateSyncStatus(sub.ID, nil); err != nil {
		log.Printf("[dav-sync] failed to update sync status: %v", err)
	}
}

func (s *DavSyncService) upsertFromObject(
	obj carddav.AddressObject,
	subID, vaultID, userID, accountID string,
	lastSyncAt *time.Time,
	result *dto.TriggerSyncResponse,
) {
	var contactID string
	var action string

	err := s.db.Transaction(func(tx *gorm.DB) error {
		var upsertErr error
		contactID, action, upsertErr = s.vcardService.UpsertContactFromVCard(
			tx, obj.Card, vaultID, userID, accountID, obj.Path, obj.ETag, lastSyncAt,
		)
		return upsertErr
	})
	if err != nil {
		errMsg := fmt.Sprintf("upsert failed: %v", err)
		s.logSyncAction(subID, nil, obj.Path, obj.ETag, "error", errMsg)
		result.Errors++
		return
	}

	switch action {
	case "created":
		result.Created++
		s.logSyncAction(subID, &contactID, obj.Path, obj.ETag, "created", "")
	case "updated":
		result.Updated++
		s.logSyncAction(subID, &contactID, obj.Path, obj.ETag, "updated", "")
	case "conflict_local_wins":
		result.Skipped++
		s.logSyncAction(subID, &contactID, obj.Path, obj.ETag, "conflict_local_wins", "local contact modified after last sync, keeping local version")
	default:
		result.Skipped++
		s.logSyncAction(subID, nil, obj.Path, obj.ETag, "skipped", "")
	}
}

func (s *DavSyncService) processDeletedPaths(
	deletedPaths []string,
	subID, vaultID string,
	result *dto.TriggerSyncResponse,
) {
	if len(deletedPaths) == 0 {
		return
	}

	var contacts []models.Contact
	s.db.Where("vault_id = ? AND distant_uri IN ?", vaultID, deletedPaths).Find(&contacts)

	for _, contact := range contacts {
		if err := s.db.Delete(&contact).Error; err != nil {
			errMsg := fmt.Sprintf("delete failed: %v", err)
			contactID := contact.ID
			s.logSyncAction(subID, &contactID, ptrToStr(contact.DistantURI), "", "error", errMsg)
			result.Errors++
			continue
		}
		contactID := contact.ID
		s.logSyncAction(subID, &contactID, ptrToStr(contact.DistantURI), "", "deleted", "")
		result.Deleted++
	}
}

func (s *DavSyncService) logSyncAction(subID string, contactID *string, distantURI, distantEtag, action, errMsg string) {
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
		log.Printf("[dav-sync] failed to create sync log: %v", err)
	}
}

func (s *DavSyncService) SyncAllDue(ctx context.Context) error {
	subs, err := s.clientService.ListDue()
	if err != nil {
		return fmt.Errorf("failed to list due subscriptions: %w", err)
	}

	for _, sub := range subs {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if _, err := s.SyncSubscription(ctx, sub.ID, sub.VaultID); err != nil {
			log.Printf("[dav-sync] sync failed for subscription %s: %v", sub.ID, err)
		}
	}
	return nil
}

func (s *DavSyncService) GetSyncLogs(subID, vaultID string, page, perPage int) ([]dto.DavSyncLogResponse, response.Meta, error) {
	var sub models.AddressBookSubscription
	if err := s.db.Where("id = ? AND vault_id = ?", subID, vaultID).First(&sub).Error; err != nil {
		return nil, response.Meta{}, ErrSubscriptionNotFound
	}

	query := s.db.Where("address_book_subscription_id = ?", subID)

	var total int64
	if err := query.Model(&models.DavSyncLog{}).Count(&total).Error; err != nil {
		return nil, response.Meta{}, err
	}

	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 15
	}
	offset := (page - 1) * perPage

	var logs []models.DavSyncLog
	if err := query.Offset(offset).Limit(perPage).Order("created_at DESC").Find(&logs).Error; err != nil {
		return nil, response.Meta{}, err
	}

	result := make([]dto.DavSyncLogResponse, len(logs))
	for i, l := range logs {
		result[i] = dto.DavSyncLogResponse{
			ID:         l.ID,
			ContactID:  l.ContactID,
			DistantURI: l.DistantURI,
			Action:     l.Action,
			Error:      l.ErrorMessage,
			CreatedAt:  l.CreatedAt,
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
