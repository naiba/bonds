package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID                     string     `json:"id" gorm:"primaryKey;type:text"`
	AccountID              string     `json:"account_id" gorm:"type:text;not null;index"`
	FirstName              *string    `json:"first_name"`
	LastName               *string    `json:"last_name"`
	Email                  string     `json:"email" gorm:"uniqueIndex;not null"`
	EmailVerifiedAt        *time.Time `json:"email_verified_at"`
	EmailVerificationToken *string    `json:"-" gorm:"type:text"`
	// EmailOwnershipUnproven 为 true 表示 EmailVerifiedAt 只是因为实例不要求邮箱验证
	// 才被填上的，没有任何人证明过这个邮箱属于注册者。EmailVerifiedAt 同时被用来
	// 放行功能，不能拿它当归属证明：否则在不发验证邮件的实例上，任何人用别人的邮箱
	// 注册后，账号都会被第三方登录按邮箱自动绑定给真正的所有者，而抢注者的密码照样能用。
	EmailOwnershipUnproven  bool       `json:"-" gorm:"not null;default:false"`
	Password                *string    `json:"-"`
	TwoFactorSecret         *string    `json:"-"`
	TwoFactorRecoveryCodes  *string    `json:"-"`
	TwoFactorConfirmedAt    *time.Time `json:"two_factor_confirmed_at"`
	IsAccountAdministrator  bool       `json:"is_account_administrator" gorm:"default:false"`
	IsInstanceAdministrator bool       `json:"is_instance_administrator" gorm:"default:false"`
	Disabled                bool       `json:"disabled" gorm:"default:false"`
	// Incrementing AuthVersion revokes previously issued JWTs without changing
	// memberships. PATs must be revoked separately when credentials are reset.
	AuthVersion               uint       `json:"-" gorm:"not null;default:0"`
	HelpShown                 bool       `json:"help_shown" gorm:"default:true"`
	InvitationCode            *string    `json:"invitation_code"`
	InvitationAcceptedAt      *time.Time `json:"invitation_accepted_at"`
	NameOrder                 string     `json:"name_order" gorm:"default:'%first_name% %last_name%'"`
	ContactSortOrder          string     `json:"contact_sort_order" gorm:"size:32;default:'name'"`
	ContactListColumns        string     `json:"contact_list_columns" gorm:"type:text;default:'[\"name\",\"nickname\",\"first_met_at\",\"status\",\"updated_at\"]'"`
	DashboardTab              string     `json:"dashboard_tab" gorm:"size:32;default:'feed'"`
	TaskView                  string     `json:"task_view" gorm:"size:16;default:'list'"`
	TaskSort                  string     `json:"task_sort" gorm:"size:16;default:'custom'"`
	Theme                     string     `json:"theme" gorm:"size:16;default:'system'"`
	DateFormat                string     `json:"date_format" gorm:"default:'MMM DD, YYYY'"`
	WeekStart                 string     `json:"week_start" gorm:"size:8;default:'sunday'"`
	NumberFormat              string     `json:"number_format" gorm:"size:8;default:'locale'"`
	DistanceFormat            string     `json:"distance_format" gorm:"default:'miles'"`
	Timezone                  *string    `json:"timezone" gorm:"default:'UTC'"`
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
