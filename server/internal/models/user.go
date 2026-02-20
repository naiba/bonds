package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID                        string     `json:"id" gorm:"primaryKey;type:text"`
	AccountID                 string     `json:"account_id" gorm:"type:text;not null;index"`
	FirstName                 *string    `json:"first_name"`
	LastName                  *string    `json:"last_name"`
	Email                     string     `json:"email" gorm:"uniqueIndex;not null"`
	EmailVerifiedAt           *time.Time `json:"email_verified_at"`
	Password                  *string    `json:"-"`
	TwoFactorSecret           *string    `json:"-"`
	TwoFactorRecoveryCodes    *string    `json:"-"`
	TwoFactorConfirmedAt      *time.Time `json:"two_factor_confirmed_at"`
	IsAccountAdministrator    bool       `json:"is_account_administrator" gorm:"default:false"`
	IsInstanceAdministrator   bool       `json:"is_instance_administrator" gorm:"default:false"`
	Disabled                  bool       `json:"disabled" gorm:"default:false"`
	HelpShown                 bool       `json:"help_shown" gorm:"default:true"`
	InvitationCode            *string    `json:"invitation_code"`
	InvitationAcceptedAt      *time.Time `json:"invitation_accepted_at"`
	NameOrder                 string     `json:"name_order" gorm:"default:'%first_name% %last_name%'"`
	ContactSortOrder          string     `json:"contact_sort_order" gorm:"default:'last_updated'"`
	DateFormat                string     `json:"date_format" gorm:"default:'MMM DD, YYYY'"`
	NumberFormat              string     `json:"number_format" gorm:"size:8;default:'locale'"`
	DistanceFormat            string     `json:"distance_format" gorm:"default:'miles'"`
	Timezone                  *string    `json:"timezone"`
	DefaultMapSite            string     `json:"default_map_site" gorm:"default:'openstreetmap'"`
	EnableAlternativeCalendar bool       `json:"enable_alternative_calendar" gorm:"default:false"`
	Locale                    string     `json:"locale" gorm:"default:'en'"`
	RememberToken             *string    `json:"-"`
	CreatedAt                 time.Time  `json:"created_at"`
	UpdatedAt                 time.Time  `json:"updated_at"`

	Account              Account                   `json:"account,omitempty" gorm:"foreignKey:AccountID"`
	Vaults               []Vault                   `json:"vaults,omitempty" gorm:"many2many:user_vault"`
	NotificationChannels []UserNotificationChannel `json:"notification_channels,omitempty" gorm:"foreignKey:UserID"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		u.ID = uuid.New().String()
	}
	return nil
}
