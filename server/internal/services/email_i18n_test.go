package services

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/testutil"
	"gorm.io/gorm"
)

// recordingMailer captures every Send() so a test can assert on the subject /
// body / recipient that a service produced — without ever hitting SMTP. Each
// service call appends one entry; tests typically grab the last one.
type recordingMailer struct {
	mu       sync.Mutex
	messages []recordedEmail
	failNext bool
}

type recordedEmail struct {
	to      string
	subject string
	body    string
}

func (m *recordingMailer) Send(to, subject, htmlBody string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, recordedEmail{to: to, subject: subject, body: htmlBody})
	if m.failNext {
		m.failNext = false
		return errors.New("forced failure")
	}
	return nil
}

func (m *recordingMailer) Close() {}

func (m *recordingMailer) last(t *testing.T) recordedEmail {
	t.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.messages) == 0 {
		t.Fatal("recordingMailer: no messages captured")
	}
	return m.messages[len(m.messages)-1]
}

// requireEmailVerifyEnabled flips system settings so AuthService.Register
// actually invokes sendVerificationEmail. By default the test path bypasses
// verification because no SMTP host is set.
func requireEmailVerifyEnabled(t *testing.T, db *gorm.DB) *SystemSettingService {
	t.Helper()
	ss := NewSystemSettingService(db)
	if err := ss.Set("auth.require_email_verification", "true"); err != nil {
		t.Fatalf("set require_email_verification: %v", err)
	}
	if err := ss.Set("smtp.host", "fake.smtp"); err != nil {
		t.Fatalf("set smtp.host: %v", err)
	}
	if err := ss.Set("app.url", "https://bonds.test"); err != nil {
		t.Fatalf("set app.url: %v", err)
	}
	return ss
}

// TestRegisterPersistsRequestedLocale guards a hidden gap: the locale param
// to Register was only used for seed translations; it never flowed onto the
// user row. So the very first verification email — and every later email
// based on user.Locale — defaulted to English even for accounts created
// from the Chinese UI.
func TestRegisterPersistsRequestedLocale(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	svc := NewAuthService(db, cfg)
	resp, err := svc.Register(dto.RegisterRequest{
		FirstName: "Locale",
		LastName:  "Tester",
		Email:     "locale-tester@example.com",
		Password:  "password123",
	}, "zh")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	var user models.User
	if err := db.First(&user, "id = ?", resp.User.ID).Error; err != nil {
		t.Fatalf("load user: %v", err)
	}
	if user.Locale != "zh" {
		t.Errorf("Register did not persist locale: user.Locale = %q, want %q", user.Locale, "zh")
	}
}

func TestVerificationEmailHonorsUserLocale(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	ss := requireEmailVerifyEnabled(t, db)
	mailer := &recordingMailer{}

	auth := NewAuthService(db, cfg)
	auth.SetMailer(mailer)
	auth.SetSystemSettings(ss)

	// Mark first user already created so Register goes through the
	// verification path (the very first registered user is auto-verified).
	if err := db.Create(&models.Account{}).Error; err != nil {
		t.Fatalf("seed dummy account: %v", err)
	}
	dummyHash := "x"
	if err := db.Create(&models.User{
		Email:    "first-user@example.com",
		Password: &dummyHash,
	}).Error; err != nil {
		t.Fatalf("seed dummy user: %v", err)
	}

	_, err := auth.Register(dto.RegisterRequest{
		FirstName: "Zh",
		LastName:  "User",
		Email:     "zh-user@example.com",
		Password:  "password123",
	}, "zh")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	last := mailer.last(t)
	if last.to != "zh-user@example.com" {
		t.Errorf("verification email recipient: got %q want zh-user@example.com", last.to)
	}
	if !strings.Contains(last.subject, "验证") {
		t.Errorf("verification subject not Chinese: %q", last.subject)
	}
	if !strings.Contains(last.body, "verify-email?token=") {
		t.Errorf("verification body missing link: %q", last.body)
	}
	if strings.Contains(last.body, "{{") {
		t.Errorf("verification body has unsubstituted placeholder: %q", last.body)
	}
}

func TestInvitationEmailHonorsCreatorLocale(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	auth := NewAuthService(db, cfg)
	creator, err := auth.Register(dto.RegisterRequest{
		FirstName: "Inviter",
		LastName:  "Zh",
		Email:     "inviter-zh@example.com",
		Password:  "password123",
	}, "zh")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	mailer := &recordingMailer{}
	svc := NewInvitationService(db, mailer, "https://bonds.test")

	_, err = svc.Create(creator.User.AccountID, creator.User.ID, dto.CreateInvitationRequest{
		Email:      "invitee@example.com",
		Permission: 100,
	})
	if err != nil {
		t.Fatalf("Create invitation: %v", err)
	}

	last := mailer.last(t)
	if last.to != "invitee@example.com" {
		t.Errorf("invitation recipient: got %q", last.to)
	}
	if !strings.Contains(last.subject, "邀请") {
		t.Errorf("invitation subject not Chinese: %q", last.subject)
	}
	if strings.Contains(last.body, "{{link}}") {
		t.Errorf("invitation body has unsubstituted placeholder: %q", last.body)
	}
	if !strings.Contains(last.body, "/accept-invite?token=") {
		t.Errorf("invitation body missing link: %q", last.body)
	}
}

func TestNotificationChannelVerifyEmailHonorsUserLocale(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	auth := NewAuthService(db, cfg)
	resp, err := auth.Register(dto.RegisterRequest{
		FirstName: "Note",
		LastName:  "User",
		Email:     "note-user@example.com",
		Password:  "password123",
	}, "zh")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	mailer := &recordingMailer{}
	svc := NewNotificationService(db)
	svc.SetMailer(mailer)

	_, err = svc.Create(resp.User.ID, dto.CreateNotificationChannelRequest{
		Type:    "email",
		Label:   "Personal",
		Content: "note-user@example.com",
	})
	if err != nil {
		t.Fatalf("Create channel: %v", err)
	}

	last := mailer.last(t)
	if !strings.Contains(last.subject, "验证") {
		t.Errorf("notification verify subject not Chinese: %q", last.subject)
	}
	if strings.Contains(last.body, "{{link}}") {
		t.Errorf("notification verify body has unsubstituted placeholder: %q", last.body)
	}
}

func TestNotificationSendTestHonorsUserLocale(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := testutil.TestJWTConfig()
	auth := NewAuthService(db, cfg)
	resp, err := auth.Register(dto.RegisterRequest{
		FirstName: "Test",
		LastName:  "Push",
		Email:     "test-push@example.com",
		Password:  "password123",
	}, "zh")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	mailer := &recordingMailer{}
	svc := NewNotificationService(db)
	svc.SetMailer(mailer)

	now := time.Now()
	ch := models.UserNotificationChannel{
		UserID:     &resp.User.ID,
		Type:       "email",
		Content:    "test-push@example.com",
		VerifiedAt: &now,
		Active:     true,
	}
	if err := db.Create(&ch).Error; err != nil {
		t.Fatalf("seed channel: %v", err)
	}

	if err := svc.SendTest(ch.ID, resp.User.ID); err != nil {
		t.Fatalf("SendTest: %v", err)
	}

	last := mailer.last(t)
	if !strings.Contains(last.subject, "测试") {
		t.Errorf("test push subject not Chinese: %q", last.subject)
	}
}
