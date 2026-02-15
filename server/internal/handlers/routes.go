package handlers

import (
	"log"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/config"
	"github.com/naiba/bonds/internal/middleware"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/search"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
	"gorm.io/gorm"
)

func RegisterRoutes(e *echo.Echo, db *gorm.DB, cfg *config.Config) {
	authMiddleware := middleware.NewAuthMiddleware(cfg.JWT.Secret)

	services.SetupOAuthProviders(&cfg.OAuth, cfg.App.URL)

	feedRecorder := services.NewFeedRecorder(db)

	authService := services.NewAuthService(db, &cfg.JWT)
	vaultService := services.NewVaultService(db)
	contactService := services.NewContactService(db)
	noteService := services.NewNoteService(db)
	reminderService := services.NewReminderService(db)
	importantDateService := services.NewImportantDateService(db)
	taskService := services.NewTaskService(db)
	callService := services.NewCallService(db)
	addressService := services.NewAddressService(db)
	contactInformationService := services.NewContactInformationService(db)
	loanService := services.NewLoanService(db)
	petService := services.NewPetService(db)
	relationshipService := services.NewRelationshipService(db)
	goalService := services.NewGoalService(db)
	lifeEventService := services.NewLifeEventService(db)
	moodTrackingService := services.NewMoodTrackingService(db)
	groupService := services.NewGroupService(db)
	quickFactService := services.NewQuickFactService(db)
	journalService := services.NewJournalService(db)
	postService := services.NewPostService(db)
	vaultTaskService := services.NewVaultTaskService(db)
	vaultFileService := services.NewVaultFileService(db, cfg.Storage.UploadDir)
	companyService := services.NewCompanyService(db)
	calendarService := services.NewCalendarService(db)
	reportService := services.NewReportService(db)
	feedService := services.NewFeedService(db)
	preferenceService := services.NewPreferenceService(db)
	notificationService := services.NewNotificationService(db)
	personalizeService := services.NewPersonalizeService(db)
	twoFactorService := services.NewTwoFactorService(db)
	vcardService := services.NewVCardService(db)

	mailer, mErr := services.NewSMTPMailer(&cfg.SMTP)
	if mErr != nil {
		log.Printf("WARNING: Failed to initialize mailer: %v", mErr)
		mailer = &services.NoopMailer{}
	}
	invitationService := services.NewInvitationService(db, mailer, cfg.App.URL)

	if cfg.Geocoding.Provider != "" {
		geocoder := services.NewGeocoder(cfg.Geocoding.Provider, cfg.Geocoding.APIKey)
		addressService.SetGeocoder(geocoder)
	}

	oauthService := services.NewOAuthService(db, &cfg.JWT, cfg.App.URL)
	webauthnService, err := services.NewWebAuthnService(db, &cfg.WebAuthn)
	if err != nil {
		log.Printf("WARNING: Failed to initialize WebAuthn: %v — WebAuthn disabled", err)
	}

	var searchEngine search.Engine
	if cfg.Bleve.IndexPath != "" {
		bleveEngine, err := search.NewBleveEngine(cfg.Bleve.IndexPath)
		if err != nil {
			log.Printf("WARNING: Failed to initialize Bleve search: %v — search disabled", err)
			searchEngine = &search.NoopEngine{}
		} else {
			searchEngine = bleveEngine
		}
	} else {
		searchEngine = &search.NoopEngine{}
	}
	searchService := services.NewSearchService(searchEngine)

	// Wire FeedRecorder into services
	contactService.SetFeedRecorder(feedRecorder)
	noteService.SetFeedRecorder(feedRecorder)
	reminderService.SetFeedRecorder(feedRecorder)
	callService.SetFeedRecorder(feedRecorder)
	taskService.SetFeedRecorder(feedRecorder)
	addressService.SetFeedRecorder(feedRecorder)
	lifeEventService.SetFeedRecorder(feedRecorder)
	loanService.SetFeedRecorder(feedRecorder)
	relationshipService.SetFeedRecorder(feedRecorder)

	contactService.SetSearchService(searchService)
	noteService.SetSearchService(searchService)

	authHandler := NewAuthHandler(authService)
	accountHandler := NewAccountHandler(db)
	vaultHandler := NewVaultHandler(vaultService)
	contactHandler := NewContactHandler(contactService)
	noteHandler := NewNoteHandler(noteService)
	reminderHandler := NewReminderHandler(reminderService)
	importantDateHandler := NewImportantDateHandler(importantDateService)
	taskHandler := NewTaskHandler(taskService)
	callHandler := NewCallHandler(callService)
	addressHandler := NewAddressHandler(addressService)
	contactInformationHandler := NewContactInformationHandler(contactInformationService)
	loanHandler := NewLoanHandler(loanService)
	petHandler := NewPetHandler(petService)
	relationshipHandler := NewRelationshipHandler(relationshipService)
	goalHandler := NewGoalHandler(goalService)
	lifeEventHandler := NewLifeEventHandler(lifeEventService)
	moodTrackingHandler := NewMoodTrackingHandler(moodTrackingService)
	groupHandler := NewGroupHandler(groupService)
	quickFactHandler := NewQuickFactHandler(quickFactService)
	journalHandler := NewJournalHandler(journalService)
	postHandler := NewPostHandler(postService)
	vaultTaskHandler := NewVaultTaskHandler(vaultTaskService)
	vaultFileHandler := NewVaultFileHandler(vaultFileService)
	avatarHandler := NewAvatarHandler(db, vaultFileService)
	companyHandler := NewCompanyHandler(companyService)
	calendarHandler := NewCalendarHandler(calendarService)
	reportHandler := NewReportHandler(reportService)
	feedHandler := NewFeedHandler(feedService)
	preferenceHandler := NewPreferenceHandler(preferenceService)
	notificationHandler := NewNotificationHandler(notificationService)
	personalizeHandler := NewPersonalizeHandler(personalizeService)
	twoFactorHandler := NewTwoFactorHandler(twoFactorService)
	searchHandler := NewSearchHandler(searchService)
	oauthHandler := NewOAuthHandler(oauthService, cfg.App.URL, cfg.JWT.Secret)
	vcardHandler := NewVCardHandler(vcardService)
	invitationHandler := NewInvitationHandler(invitationService)

	e.Use(middleware.CORS())

	api := e.Group("/api")

	api.GET("/announcement", func(c echo.Context) error {
		return response.OK(c, map[string]string{"content": cfg.Announcement})
	})

	auth := api.Group("/auth")
	auth.POST("/register", authHandler.Register)
	auth.POST("/login", authHandler.Login)
	auth.POST("/refresh", authHandler.Refresh, authMiddleware.Authenticate)
	auth.GET("/me", authHandler.Me, authMiddleware.Authenticate)
	auth.GET("/:provider", oauthHandler.BeginAuth)
	auth.GET("/:provider/callback", oauthHandler.Callback)

	if webauthnService != nil {
		webauthnHandler := NewWebAuthnHandler(webauthnService, authService)
		auth.POST("/webauthn/login/begin", webauthnHandler.BeginLogin)
		auth.POST("/webauthn/login/finish", webauthnHandler.FinishLogin)
	}

	api.POST("/invitations/accept", invitationHandler.Accept)

	protected := api.Group("", authMiddleware.Authenticate)

	protected.GET("/account", accountHandler.GetAccount)

	vaults := protected.Group("/vaults")
	vaults.GET("", vaultHandler.List)
	vaults.POST("", vaultHandler.Create)

	vaultDetail := vaults.Group("/:id", VaultPermissionMiddleware(vaultService, models.PermissionViewer))
	vaultDetail.GET("", vaultHandler.Get)
	vaultDetail.PUT("", vaultHandler.Update, VaultPermissionMiddleware(vaultService, models.PermissionEditor))
	vaultDetail.DELETE("", vaultHandler.Delete, VaultPermissionMiddleware(vaultService, models.PermissionManager))

	requireEditor := VaultPermissionMiddleware(vaultService, models.PermissionEditor)

	contacts := protected.Group("/vaults/:vault_id/contacts", VaultPermissionMiddleware(vaultService, models.PermissionViewer))
	contacts.GET("", contactHandler.List)
	contacts.POST("", contactHandler.Create, requireEditor)
	contacts.GET("/:id", contactHandler.Get)
	contacts.PUT("/:id", contactHandler.Update, requireEditor)
	contacts.DELETE("/:id", contactHandler.Delete, requireEditor)
	contacts.PUT("/:id/archive", contactHandler.ToggleArchive, requireEditor)
	contacts.PUT("/:id/favorite", contactHandler.ToggleFavorite)
	contacts.GET("/export", vcardHandler.ExportVault)
	contacts.POST("/import", vcardHandler.ImportVCard, requireEditor)

	contactSub := protected.Group("/vaults/:vault_id/contacts/:contact_id", VaultPermissionMiddleware(vaultService, models.PermissionViewer))
	contactSub.GET("/vcard", vcardHandler.ExportContact)

	notes := contactSub.Group("/notes")
	notes.GET("", noteHandler.List)
	notes.POST("", noteHandler.Create, requireEditor)
	notes.PUT("/:id", noteHandler.Update, requireEditor)
	notes.DELETE("/:id", noteHandler.Delete, requireEditor)

	reminders := contactSub.Group("/reminders")
	reminders.GET("", reminderHandler.List)
	reminders.POST("", reminderHandler.Create, requireEditor)
	reminders.PUT("/:id", reminderHandler.Update, requireEditor)
	reminders.DELETE("/:id", reminderHandler.Delete, requireEditor)

	dates := contactSub.Group("/dates")
	dates.GET("", importantDateHandler.List)
	dates.POST("", importantDateHandler.Create, requireEditor)
	dates.PUT("/:id", importantDateHandler.Update, requireEditor)
	dates.DELETE("/:id", importantDateHandler.Delete, requireEditor)

	tasks := contactSub.Group("/tasks")
	tasks.GET("", taskHandler.List)
	tasks.POST("", taskHandler.Create, requireEditor)
	tasks.PUT("/:id", taskHandler.Update, requireEditor)
	tasks.PUT("/:id/toggle", taskHandler.ToggleCompleted, requireEditor)
	tasks.DELETE("/:id", taskHandler.Delete, requireEditor)

	callRoutes := contactSub.Group("/calls")
	callRoutes.GET("", callHandler.List)
	callRoutes.POST("", callHandler.Create, requireEditor)
	callRoutes.PUT("/:id", callHandler.Update, requireEditor)
	callRoutes.DELETE("/:id", callHandler.Delete, requireEditor)

	addresses := contactSub.Group("/addresses")
	addresses.GET("", addressHandler.List)
	addresses.POST("", addressHandler.Create, requireEditor)
	addresses.PUT("/:id", addressHandler.Update, requireEditor)
	addresses.DELETE("/:id", addressHandler.Delete, requireEditor)

	contactInfo := contactSub.Group("/contactInformation")
	contactInfo.GET("", contactInformationHandler.List)
	contactInfo.POST("", contactInformationHandler.Create, requireEditor)
	contactInfo.PUT("/:id", contactInformationHandler.Update, requireEditor)
	contactInfo.DELETE("/:id", contactInformationHandler.Delete, requireEditor)

	loanRoutes := contactSub.Group("/loans")
	loanRoutes.GET("", loanHandler.List)
	loanRoutes.POST("", loanHandler.Create, requireEditor)
	loanRoutes.PUT("/:id", loanHandler.Update, requireEditor)
	loanRoutes.PUT("/:id/toggle", loanHandler.ToggleSettled, requireEditor)
	loanRoutes.DELETE("/:id", loanHandler.Delete, requireEditor)

	petRoutes := contactSub.Group("/pets")
	petRoutes.GET("", petHandler.List)
	petRoutes.POST("", petHandler.Create, requireEditor)
	petRoutes.PUT("/:id", petHandler.Update, requireEditor)
	petRoutes.DELETE("/:id", petHandler.Delete, requireEditor)

	relationshipRoutes := contactSub.Group("/relationships")
	relationshipRoutes.GET("", relationshipHandler.List)
	relationshipRoutes.POST("", relationshipHandler.Create, requireEditor)
	relationshipRoutes.PUT("/:id", relationshipHandler.Update, requireEditor)
	relationshipRoutes.DELETE("/:id", relationshipHandler.Delete, requireEditor)

	goalRoutes := contactSub.Group("/goals")
	goalRoutes.GET("", goalHandler.List)
	goalRoutes.POST("", goalHandler.Create, requireEditor)
	goalRoutes.GET("/:id", goalHandler.Get)
	goalRoutes.PUT("/:id", goalHandler.Update, requireEditor)
	goalRoutes.PUT("/:id/streaks", goalHandler.AddStreak, requireEditor)
	goalRoutes.DELETE("/:id", goalHandler.Delete, requireEditor)

	timelineRoutes := contactSub.Group("/timelineEvents")
	timelineRoutes.GET("", lifeEventHandler.ListTimelineEvents)
	timelineRoutes.POST("", lifeEventHandler.CreateTimelineEvent, requireEditor)
	timelineRoutes.POST("/:id/lifeEvents", lifeEventHandler.AddLifeEvent, requireEditor)
	timelineRoutes.PUT("/:id/lifeEvents/:lifeEventId", lifeEventHandler.UpdateLifeEvent, requireEditor)
	timelineRoutes.DELETE("/:id", lifeEventHandler.DeleteTimelineEvent, requireEditor)
	timelineRoutes.DELETE("/:id/lifeEvents/:lifeEventId", lifeEventHandler.DeleteLifeEvent, requireEditor)

	moodRoutes := contactSub.Group("/moodTrackingEvents")
	moodRoutes.POST("", moodTrackingHandler.Create, requireEditor)
	moodRoutes.GET("", moodTrackingHandler.List)

	contactSub.POST("/photos", vaultFileHandler.UploadContactFile, requireEditor)
	contactSub.POST("/documents", vaultFileHandler.UploadContactFile, requireEditor)
	contactSub.GET("/avatar", avatarHandler.GetAvatar)

	quickFactRoutes := contactSub.Group("/quickFacts")
	quickFactRoutes.GET("/:templateId", quickFactHandler.List)
	quickFactRoutes.POST("/:templateId", quickFactHandler.Create, requireEditor)
	quickFactRoutes.PUT("/:templateId/:id", quickFactHandler.Update, requireEditor)
	quickFactRoutes.DELETE("/:templateId/:id", quickFactHandler.Delete, requireEditor)

	vaultScoped := protected.Group("/vaults/:vault_id", VaultPermissionMiddleware(vaultService, models.PermissionViewer))
	vaultScoped.GET("/groups", groupHandler.List)
	vaultScoped.GET("/groups/:id", groupHandler.Get)
	vaultScoped.PUT("/groups/:id", groupHandler.Update, requireEditor)
	vaultScoped.DELETE("/groups/:id", groupHandler.Delete, requireEditor)

	contacts.POST("/:contact_id/groups", groupHandler.AddContactToGroup, requireEditor)
	contacts.DELETE("/:contact_id/groups/:id", groupHandler.RemoveContactFromGroup, requireEditor)

	journalRoutes := vaultScoped.Group("/journals")
	journalRoutes.GET("", journalHandler.List)
	journalRoutes.POST("", journalHandler.Create, requireEditor)
	journalRoutes.GET("/:id", journalHandler.Get)
	journalRoutes.PUT("/:id", journalHandler.Update, requireEditor)
	journalRoutes.DELETE("/:id", journalHandler.Delete, requireEditor)

	postRoutes := vaultScoped.Group("/journals/:journal_id/posts")
	postRoutes.GET("", postHandler.List)
	postRoutes.POST("", postHandler.Create, requireEditor)
	postRoutes.GET("/:id", postHandler.Get)
	postRoutes.PUT("/:id", postHandler.Update, requireEditor)
	postRoutes.DELETE("/:id", postHandler.Delete, requireEditor)

	vaultScoped.GET("/tasks", vaultTaskHandler.List)

	vaultScoped.GET("/files", vaultFileHandler.List)
	vaultScoped.POST("/files", vaultFileHandler.Upload, requireEditor)
	vaultScoped.GET("/files/:id/download", vaultFileHandler.Serve)
	vaultScoped.DELETE("/files/:id", vaultFileHandler.Delete, requireEditor)

	vaultScoped.GET("/companies", companyHandler.List)
	vaultScoped.GET("/companies/:id", companyHandler.Get)

	vaultScoped.GET("/calendar", calendarHandler.Get)

	vaultScoped.GET("/reports/addresses", reportHandler.Addresses)
	vaultScoped.GET("/reports/importantDates", reportHandler.ImportantDates)
	vaultScoped.GET("/reports/moodTrackingEvents", reportHandler.MoodTrackingEvents)

	vaultScoped.GET("/feed", feedHandler.Get)
	vaultScoped.GET("/search", searchHandler.Search)

	settingsGroup := protected.Group("/settings")

	prefsGroup := settingsGroup.Group("/preferences")
	prefsGroup.GET("", preferenceHandler.Get)
	prefsGroup.PUT("", preferenceHandler.UpdateAll)
	prefsGroup.POST("/name", preferenceHandler.UpdateNameOrder)
	prefsGroup.POST("/date", preferenceHandler.UpdateDateFormat)
	prefsGroup.POST("/timezone", preferenceHandler.UpdateTimezone)
	prefsGroup.POST("/locale", preferenceHandler.UpdateLocale)

	notifGroup := settingsGroup.Group("/notifications")
	notifGroup.GET("", notificationHandler.List)
	notifGroup.POST("", notificationHandler.Create)
	notifGroup.PUT("/:id/toggle", notificationHandler.Toggle)
	notifGroup.DELETE("/:id", notificationHandler.Delete)

	personalizeGroup := settingsGroup.Group("/personalize", authMiddleware.RequireAdmin)
	personalizeGroup.GET("/:entity", personalizeHandler.List)
	personalizeGroup.POST("/:entity", personalizeHandler.Create)
	personalizeGroup.PUT("/:entity/:id", personalizeHandler.Update)
	personalizeGroup.DELETE("/:entity/:id", personalizeHandler.Delete)

	if webauthnService != nil {
		webauthnHandler := NewWebAuthnHandler(webauthnService, authService)
		webauthnGroup := settingsGroup.Group("/webauthn")
		webauthnGroup.POST("/register/begin", webauthnHandler.BeginRegistration)
		webauthnGroup.POST("/register/finish", webauthnHandler.FinishRegistration)
		webauthnGroup.GET("/credentials", webauthnHandler.ListCredentials)
		webauthnGroup.DELETE("/credentials/:id", webauthnHandler.DeleteCredential)
	}

	inviteGroup := settingsGroup.Group("/invitations", authMiddleware.RequireAdmin)
	inviteGroup.GET("", invitationHandler.List)
	inviteGroup.POST("", invitationHandler.Create)
	inviteGroup.DELETE("/:id", invitationHandler.Delete)

	twoFactorGroup := settingsGroup.Group("/2fa")
	twoFactorGroup.POST("/enable", twoFactorHandler.Enable)
	twoFactorGroup.POST("/confirm", twoFactorHandler.Confirm)
	twoFactorGroup.POST("/disable", twoFactorHandler.Disable)
	twoFactorGroup.GET("/status", twoFactorHandler.Status)
}
