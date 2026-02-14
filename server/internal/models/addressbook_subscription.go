package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AddressBookSubscription struct {
	ID                 string     `json:"id" gorm:"primaryKey;type:text"`
	UserID             string     `json:"user_id" gorm:"type:text;not null;index"`
	VaultID            string     `json:"vault_id" gorm:"type:text;not null;index"`
	URI                string     `json:"uri" gorm:"size:2096;not null"`
	Username           string     `json:"username" gorm:"size:1024;not null"`
	Password           string     `json:"-" gorm:"size:2048;not null"`
	Active             bool       `json:"active" gorm:"default:true"`
	SyncWay            uint8      `json:"sync_way" gorm:"default:2"`
	Capabilities       string     `json:"capabilities" gorm:"size:2048;not null"`
	DistantSyncToken   *string    `json:"distant_sync_token" gorm:"size:512"`
	LastBatch          *string    `json:"last_batch"`
	CurrentLogID       *uint64    `json:"current_logid"`
	SyncTokenID        *uint      `json:"sync_token_id" gorm:"index"`
	Frequency          int        `json:"frequency" gorm:"default:180"`
	LastSynchronizedAt *time.Time `json:"last_synchronized_at"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`

	User      User       `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Vault     Vault      `json:"vault,omitempty" gorm:"foreignKey:VaultID"`
	SyncToken *SyncToken `json:"sync_token,omitempty" gorm:"foreignKey:SyncTokenID"`
}

func (AddressBookSubscription) TableName() string {
	return "addressbook_subscriptions"
}

func (a *AddressBookSubscription) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	return nil
}
