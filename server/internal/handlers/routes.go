package handlers

import (
	"log"
	"net/http"

	"github.com/labstack/echo/v5"
	swaggerFiles "github.com/swaggo/files/v2"
	"github.com/swaggo/swag"

	"github.com/naiba/bonds/internal/config"
	internalmcp "github.com/naiba/bonds/internal/mcp"
	"github.com/naiba/bonds/internal/middleware"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/search"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
	"gorm.io/gorm"

	_ "github.com/naiba/bonds/docs"
)

func RegisterRoutes(e *echo.Echo, db *gorm.DB, cfg *config.Config, version string, backupReloader func()) {
	authMiddleware := middleware.NewAuthMiddleware(cfg.JWT.Secret, db)

	systemSettingService := services.NewSystemSettingServiceWithCipher(db, cfg.Security.SettingsEncKey)
	if migrated, err := systemSettingService.MigratePlaintextSecrets(); err != nil {
		log.Printf("WARNING: failed to encrypt legacy plaintext secrets: %v", err)
	} else if migrated > 0 {
		log.Printf("Encrypted %d previously-plaintext secret system settings", migrated)
	}

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
	giftService := services.NewGiftService(db)
	relationshipService := services.NewRelationshipService(db)
	goalService := services.NewGoalService(db)
	activityService := services.NewActivityService(db)
	moodTrackingService := services.NewMoodTrackingService(db)
	groupService := services.NewGroupService(db)
	quickFactService := services.NewQuickFactService(db)
	journalService := services.NewJournalService(db)
	postService := services.NewPostService(db)
	journalService.SetUploadDir(cfg.Storage.UploadDir)
	postService.SetUploadDir(cfg.Storage.UploadDir)
	vaultTaskService := services.NewVaultTaskService(db)
	vaultFileService := services.NewVaultFileService(db, cfg.Storage.UploadDir)
	companyService := services.NewCompanyService(db)
	calendarService := services.NewCalendarService(db)
	calendarICSService := services.NewCalendarICSService(db)
	reportService := services.NewReportService(db)
	feedService := services.NewFeedService(db)
	preferenceService := services.NewPreferenceService(db)
	notificationSender := services.NewShoutrrrSender()
	notificationService := services.NewNotificationService(db)
	personalizeService := services.NewPersonalizeService(db)
	twoFactorService := services.NewTwoFactorService(db)
	vcardService := services.NewVCardService(db)
	contactLabelService := services.NewContactLabelService(db)
	contactReligionService := services.NewContactReligionService(db)
	contactJobService := services.NewContactJobService(db)
	contactMoveService := services.NewContactMoveService(db)
	contactTemplateService := services.NewContactTemplateService(db)
	contactTabService := services.NewContactTabService(db)
	contactLayoutService := services.NewContactLayoutService(db)
	journalMetricService := services.NewJournalMetricService(db)
	postMetricService := services.NewPostMetricService(db)
	postTagService := services.NewPostTagService(db)
	sliceOfLifeService := services.NewSliceOfLifeService(db)
	lifeMetricService := services.NewLifeMetricService(db)
	vaultReminderService := services.NewVaultReminderService(db)
	mostConsultedService := services.NewMostConsultedService(db)
	postTemplateSectionService := services.NewPostTemplateSectionService(db)
	groupTypeRoleService := services.NewGroupTypeRoleService(db)
	relationshipTypeService := services.NewRelationshipTypeService(db)
	callReasonService := services.NewCallReasonService(db)
	vaultSettingsService := services.NewVaultSettingsService(db)
	vaultUsersService := services.NewVaultUsersService(db)
	vaultLabelService := services.NewVaultLabelService(db)
	vaultTagService := services.NewVaultTagService(db)
	vaultDateTypeService := services.NewVaultImportantDateTypeService(db)
	vaultMoodParamService := services.NewVaultMoodParamService(db)
	vaultActivitySettingsService := services.NewVaultActivityService(db)
	vaultQuickFactTplService := services.NewVaultQuickFactTemplateService(db)
	vaultQuickFactTplService.SetUploadDir(cfg.Storage.UploadDir)
	userManagementService := services.NewUserManagementService(db)
	accountCancelService := services.NewAccountCancelService(db)
	storageInfoService := services.NewStorageInfoService(db, systemSettingService)
	backupService := services.NewBackupService(db, cfg)
	currencyService := services.NewCurrencyService(db)
	davClientService := services.NewDavClientService(db, cfg.JWT.Secret)
	davSyncService := services.NewDavSyncService(db, davClientService, vcardService)
	davPushService := services.NewDavPushService(db, davClientService, vcardService)
	monicaImportService := services.NewMonicaImportService(db, cfg.Storage.UploadDir)
	csvImportService := services.NewCSVImportService(db)
	adminService := services.NewAdminService(db, cfg.Storage.UploadDir)

	patService := services.NewPersonalAccessTokenService(db)

	mailer := services.NewDynamicMailer(systemSettingService)
	authService.SetMailer(mailer)
	authService.SetSystemSettings(systemSettingService)
	invitationService := services.NewInvitationService(db, mailer, cfg.App.URL)
	invitationService.SetSystemSettings(systemSettingService)
	notificationService.SetMailer(mailer)
	notificationService.SetSender(notificationSender)
	notificationService.SetSystemSettings(systemSettingService)

	geocodingRegistry := services.NewGeocodingProviderRegistry()
	geocodingConfigService := services.NewGeocodingProviderConfigService(db, cfg.Security.SettingsEncKey, geocodingRegistry)
	geocodingManager := services.NewGeocodingManager(
		systemSettingService,
		geocodingConfigService,
		geocodingRegistry,
		addressService,
		cfg.Geocoding.Provider,
		cfg.Geocoding.APIKey,
		cfg.Geocoding.Precision,
	)
	if migrated, err := geocodingManager.Initialize(); err != nil {
		log.Printf("WARNING: failed to initialize geocoding providers: %v", err)
	} else if migrated > 0 {
		log.Printf("Encrypted %d previously-plaintext geocoding provider configurations", migrated)
	}

	oauthProviderService := services.NewOAuthProviderServiceWithCipher(db, cfg.Security.SettingsEncKey)
	oauthProviderService.SetSystemSettings(systemSettingService)
	if migrated, err := oauthProviderService.MigratePlaintextSecrets(); err != nil {
		log.Printf("WARNING: failed to encrypt legacy OAuth client_secret values: %v", err)
	} else if migrated > 0 {
		log.Printf("Encrypted %d previously-plaintext OAuth client_secret values", migrated)
	}

	oauthService := services.NewOAuthService(db, &cfg.JWT)
	webauthnService, err := services.NewWebAuthnService(db, &cfg.WebAuthn)
	if err != nil {
		log.Printf("WARNING: Failed to initialize WebAuthn: %v — WebAuthn disabled", err)
	}
	webauthnService.SetSystemSettings(systemSettingService)
	backupService.SetSystemSettings(systemSettingService)

	var searchEngine search.Engine
	if cfg.Bleve.IndexPath != "" {
		bleveEngine, err := search.NewBleveEngine(cfg.Bleve.IndexPath)
		if err != nil {
			log.Printf("WARNING: Failed to initialize Bleve search: %v \u2014 search disabled", err)
			searchEngine = &search.NoopEngine{}
		} else {
			searchEngine = bleveEngine
		}
	} else {
		searchEngine = &search.NoopEngine{}
	}
	searchService := services.NewSearchServiceWithDB(db, searchEngine)

	// Wire FeedRecorder into services
	contactService.SetFeedRecorder(feedRecorder)
	noteService.SetFeedRecorder(feedRecorder)
	reminderService.SetFeedRecorder(feedRecorder)
	callService.SetFeedRecorder(feedRecorder)
	taskService.SetFeedRecorder(feedRecorder)
	// VaultTaskService records assignee Feed entries only when this shared recorder is wired.
	vaultTaskService.SetFeedRecorder(feedRecorder)
	addressService.SetFeedRecorder(feedRecorder)
	activityService.SetFeedRecorder(feedRecorder)
	loanService.SetFeedRecorder(feedRecorder)
	relationshipService.SetFeedRecorder(feedRecorder)
	vaultFileService.SetFeedRecorder(feedRecorder)
	quickFactService.SetFileService(vaultFileService)

	contactService.SetSearchService(searchService)
	contactService.SetDavPushService(davPushService)
	contactMoveService.SetSearchService(searchService)
	contactMoveService.SetDavPushService(davPushService)
	contactMoveService.SetFileService(vaultFileService)
	noteService.SetSearchService(searchService)
	monicaImportService.SetFeedRecorder(feedRecorder)
	monicaImportService.SetSearchEngine(searchEngine)
	csvImportService.SetFeedRecorder(feedRecorder)
	csvImportService.SetSearchService(searchService)
	csvImportService.SetDavPushService(davPushService)

	postPhotoHandler := NewPostPhotoHandler(vaultFileService, storageInfoService, systemSettingService)
	contactPhotoHandler := NewContactPhotoHandler(vaultFileService)
	contactDocumentHandler := NewContactDocumentHandler(vaultFileService)

	authHandler := NewAuthHandler(authService, systemSettingService)
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
	giftHandler := NewGiftHandler(giftService)
	relationshipHandler := NewRelationshipHandler(relationshipService)
	goalHandler := NewGoalHandler(goalService)
	activityHandler := NewActivityHandler(activityService)
	moodTrackingHandler := NewMoodTrackingHandler(moodTrackingService)
	groupHandler := NewGroupHandler(groupService)
	quickFactHandler := NewQuickFactHandler(quickFactService, storageInfoService, systemSettingService)
	journalHandler := NewJournalHandler(journalService)
	postHandler := NewPostHandler(postService)
	vaultTaskHandler := NewVaultTaskHandler(vaultTaskService)
	vaultFileHandler := NewVaultFileHandler(vaultFileService, storageInfoService, systemSettingService)
	avatarHandler := NewAvatarHandler(db, vaultFileService)
	companyHandler := NewCompanyHandler(companyService, contactJobService)
	calendarHandler := NewCalendarHandler(calendarService, calendarICSService)
	reportHandler := NewReportHandler(reportService, addressService)
	feedHandler := NewFeedHandler(feedService)
	preferenceHandler := NewPreferenceHandler(preferenceService)
	notificationHandler := NewNotificationHandler(notificationService)
	personalizeHandler := NewPersonalizeHandler(personalizeService)
	twoFactorHandler := NewTwoFactorHandler(twoFactorService)
	searchHandler := NewSearchHandler(searchService)
	oauthHandler := NewOAuthHandler(oauthService, systemSettingService, cfg.JWT.Secret)
	vcardHandler := NewVCardHandler(vcardService)
	monicaImportHandler := NewMonicaImportHandler(monicaImportService)
	csvImportHandler := NewCSVImportHandler(csvImportService)
	invitationHandler := NewInvitationHandler(invitationService)
	contactLabelHandler := NewContactLabelHandler(contactLabelService)
	contactReligionHandler := NewContactReligionHandler(contactReligionService)
	contactJobHandler := NewContactJobHandler(contactJobService)
	contactMoveHandler := NewContactMoveHandler(contactMoveService)
	contactTemplateHandler := NewContactTemplateHandler(contactTemplateService)
	contactTabHandler := NewContactTabHandler(contactTabService)
	contactLayoutHandler := NewContactLayoutHandler(contactLayoutService)
	journalMetricHandler := NewJournalMetricHandler(journalMetricService)
	postMetricHandler := NewPostMetricHandler(postMetricService)
	postTagHandler := NewPostTagHandler(postTagService)
	sliceOfLifeHandler := NewSliceOfLifeHandler(sliceOfLifeService)
	lifeMetricHandler := NewLifeMetricHandler(lifeMetricService)
	vaultReminderHandler := NewVaultReminderHandler(vaultReminderService)
	mostConsultedHandler := NewMostConsultedHandler(mostConsultedService)
	postTemplateSectionHandler := NewPostTemplateSectionHandler(postTemplateSectionService)
	groupTypeRoleHandler := NewGroupTypeRoleHandler(groupTypeRoleService)
	relationshipTypeHandler := NewRelationshipTypeHandler(relationshipTypeService)
	callReasonHandler := NewCallReasonHandler(callReasonService)
	vaultSettingsHandler := NewVaultSettingsHandler(
		vaultSettingsService, vaultUsersService, vaultLabelService, vaultTagService,
		vaultDateTypeService, vaultMoodParamService, vaultActivitySettingsService, vaultQuickFactTplService,
	)
	userManagementHandler := NewUserManagementHandler(userManagementService)
	accountCancelHandler := NewAccountCancelHandler(accountCancelService)
	storageInfoHandler := NewStorageInfoHandler(storageInfoService)
	backupHandler := NewBackupHandler(backupService)
	currencyHandler := NewCurrencyHandler(currencyService)
	davClientHandler := NewDavClientHandler(davClientService, davSyncService)
	adminHandler := NewAdminHandler(adminService, systemSettingService, searchService, db)
	adminHandler.RegisterReloader(func() {
		if err := geocodingManager.Reload(); err != nil {
			log.Printf("WARNING: geocoding reload failed: %v", err)
		}
	})
	adminHandler.RegisterReloader(func() {
		oauthProviderService.ReloadProviders()
	})
	adminHandler.RegisterReloader(func() {
		if err := webauthnService.ReloadConfig(); err != nil {
			log.Printf("WARNING: WebAuthn reload failed: %v", err)
		}
	})
	if backupReloader != nil {
		adminHandler.RegisterReloader(backupReloader)
	}
	oauthProviderHandler := NewOAuthProviderHandler(oauthProviderService)
	geocodingAdminHandler := NewGeocodingAdminHandler(geocodingManager)
	instanceHandler := NewInstanceHandler(systemSettingService, oauthService, webauthnService, version)

	patHandler := NewPersonalAccessTokenHandler(patService)

	e.Use(middleware.CORS())

	swaggerAssets := echo.WrapHandler(http.StripPrefix("/swagger/", http.FileServer(http.FS(swaggerFiles.FS))))
	e.GET("/swagger/*", func(c *echo.Context) error {
		if !systemSettingService.GetBool("swagger.enabled", cfg.Debug) {
			return c.NoContent(http.StatusNotFound)
		}
		if c.Param("*") == "doc.json" {
			doc, err := swag.ReadDoc()
			if err != nil {
				return err
			}
			return c.Blob(http.StatusOK, "application/json", []byte(doc))
		}
		if c.Param("*") == "swagger-initializer.js" {
			return c.Blob(http.StatusOK, "application/javascript", []byte(`window.onload = function() {
  window.ui = SwaggerUIBundle({
    url: "/swagger/doc.json",
    dom_id: "#swagger-ui",
    deepLinking: true,
    presets: [SwaggerUIBundle.presets.apis, SwaggerUIStandalonePreset],
    plugins: [SwaggerUIBundle.plugins.DownloadUrl],
    layout: "StandaloneLayout"
  });
};`))
		}
		return swaggerAssets(c)
	})

	api := e.Group("/api")

	api.GET("/announcement", func(c *echo.Context) error {
		content := systemSettingService.GetWithDefault("announcement", "")
		return response.OK(c, map[string]string{"content": content})
	})

	auth := api.Group("/auth")
	auth.POST("/register", authHandler.Register)
	auth.POST("/login", authHandler.Login)
	auth.POST("/refresh", authHandler.Refresh, authMiddleware.Authenticate)
	auth.GET("/me", authHandler.Me, authMiddleware.Authenticate)
	auth.GET("/providers", oauthHandler.AvailableProviders)
	auth.POST("/oauth/link", oauthHandler.LinkProvider, authMiddleware.Authenticate)
	auth.POST("/oauth/link-register", oauthHandler.LinkRegister)
	auth.GET("/:provider", oauthHandler.BeginAuth)
	auth.GET("/:provider/callback", oauthHandler.Callback)

	webauthnHandler := NewWebAuthnHandler(webauthnService, authService)
	auth.POST("/verify-email", authHandler.VerifyEmail)
	auth.POST("/resend-verification", authHandler.ResendVerification, authMiddleware.Authenticate)

	auth.POST("/webauthn/login/begin", webauthnHandler.BeginLogin)
	auth.POST("/webauthn/login/finish", webauthnHandler.FinishLogin)
	auth.POST("/2fa/verify", authHandler.VerifyTwoFactor)

	api.POST("/invitations/accept", invitationHandler.Accept)

	api.GET("/instance/info", instanceHandler.GetInfo)

	emailVerificationRequired := func() bool {
		if !systemSettingService.GetBool("auth.require_email_verification", true) {
			return false
		}
		return systemSettingService.GetWithDefault("smtp.host", "") != ""
	}

	adminGroup := api.Group("/admin", authMiddleware.Authenticate, middleware.RequireEmailVerification(emailVerificationRequired), middleware.DenyScopedPAT, authMiddleware.RequireInstanceAdmin)
	adminGroup.GET("/users", adminHandler.ListUsers)
	adminGroup.PUT("/users/:id/toggle", adminHandler.ToggleUser)
	adminGroup.PUT("/users/:id/admin", adminHandler.SetAdmin)
	adminGroup.DELETE("/users/:id", adminHandler.DeleteUser)
	adminGroup.PUT("/users/:id/storage-limit", adminHandler.SetStorageLimit)
	adminGroup.GET("/settings", adminHandler.GetSettings)
	adminGroup.PUT("/settings", adminHandler.UpdateSettings)
	adminGroup.GET("/geocoding", geocodingAdminHandler.Get)
	adminGroup.PUT("/geocoding", geocodingAdminHandler.UpdateSettings)
	adminGroup.PUT("/geocoding/providers/:provider", geocodingAdminHandler.UpdateProvider)
	adminGroup.DELETE("/geocoding/providers/:provider", geocodingAdminHandler.DeleteProvider)
	adminGroup.GET("/oauth-providers", oauthProviderHandler.List)
	adminGroup.POST("/oauth-providers", oauthProviderHandler.Create)
	adminGroup.PUT("/oauth-providers/:id", oauthProviderHandler.Update)
	adminGroup.DELETE("/oauth-providers/:id", oauthProviderHandler.Delete)

	adminGroup.POST("/search/rebuild", adminHandler.RebuildSearchIndex)
	backupGroup := adminGroup.Group("/backups")
	backupGroup.GET("", backupHandler.List)
	backupGroup.POST("", backupHandler.Create)
	backupGroup.GET("/config", backupHandler.GetConfig)
	backupGroup.GET("/:filename/download", backupHandler.Download)
	backupGroup.DELETE("/:filename", backupHandler.Delete)
	backupGroup.POST("/:filename/restore", backupHandler.Restore)

	protected := api.Group("", authMiddleware.Authenticate, middleware.RequireEmailVerification(emailVerificationRequired), middleware.DenyScopedPAT)

	protected.GET("/account", accountHandler.GetAccount)

	// Cross-vault relationship contacts (no vault scope — returns contacts from all accessible vaults)
	protected.GET("/relationships/contacts", relationshipHandler.ListContactsAcrossVaults)

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
	contacts.GET("/selectable", contactHandler.ListSelectable)
	contacts.GET("/labels/:labelId", contactHandler.ListByLabel)
	contacts.POST("/move", contactMoveHandler.MoveMany, requireEditor)
	contacts.POST("", contactHandler.Create, requireEditor)
	contacts.DELETE("", contactHandler.DeleteMany, requireEditor)
	contacts.GET("/:id", contactHandler.Get)
	contacts.PUT("/:id", contactHandler.Update, requireEditor)
	contacts.DELETE("/:id", contactHandler.Delete, requireEditor)
	contacts.PUT("/:id/archive", contactHandler.ToggleArchive, requireEditor)
	contacts.PUT("/:id/favorite", contactHandler.ToggleFavorite)
	contacts.GET("/export", vcardHandler.ExportVault)
	contacts.POST("/import", vcardHandler.ImportVCard, requireEditor)

	contactSub := protected.Group("/vaults/:vault_id/contacts/:contact_id", VaultPermissionMiddleware(vaultService, models.PermissionViewer))
	contactSub.GET("/vcard", vcardHandler.ExportContact)
	contactSub.GET("/labels", contactLabelHandler.List)
	contactSub.POST("/labels", contactLabelHandler.Add, requireEditor)
	contactSub.PUT("/labels/:id", contactLabelHandler.Update, requireEditor)
	contactSub.DELETE("/labels/:id", contactLabelHandler.Remove, requireEditor)
	contactSub.PUT("/religion", contactReligionHandler.Update, requireEditor)
	// Legacy endpoints for backward compatibility — work with new ContactCompany table
	contactSub.PUT("/jobInformation", contactJobHandler.LegacyUpdate, requireEditor)
	contactSub.DELETE("/jobInformation", contactJobHandler.LegacyDelete, requireEditor)

	// New many-to-many job CRUD endpoints
	jobRoutes := contactSub.Group("/jobs")
	jobRoutes.GET("", contactJobHandler.List)
	jobRoutes.POST("", contactJobHandler.Create, requireEditor)
	jobRoutes.PUT("/:job_id", contactJobHandler.UpdateJob, requireEditor)
	jobRoutes.DELETE("/:job_id", contactJobHandler.DeleteJob, requireEditor)
	contactSub.GET("/feed", feedHandler.GetContactFeed)
	contactSub.POST("/catchUp", contactHandler.MarkCaughtUp, requireEditor)
	contactSub.POST("/move", contactMoveHandler.Move, requireEditor)
	contactSub.PUT("/template", contactTemplateHandler.Update, requireEditor)
	contactSub.GET("/tabs", contactTabHandler.GetTabs)
	contactSub.PUT("/avatar", avatarHandler.UpdateAvatar, requireEditor)
	contactSub.DELETE("/avatar", avatarHandler.DeleteAvatar, requireEditor)

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
	tasks.GET("/completed", taskHandler.ListCompleted)
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
	addresses.GET("/:id/image/:width/:height", addressHandler.GetMapImage)

	vaultContactInfo := protected.Group("/vaults/:vault_id/contactInformation", VaultPermissionMiddleware(vaultService, models.PermissionViewer))
	vaultContactInfo.GET("/by-identity", contactInformationHandler.FindByIdentity)

	vaultRelationships := protected.Group("/vaults/:vault_id/relationships", VaultPermissionMiddleware(vaultService, models.PermissionViewer))
	vaultRelationships.GET("/graph", relationshipHandler.GetVaultGraph)

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

	giftRoutes := contactSub.Group("/gifts")
	giftRoutes.GET("", giftHandler.List)
	giftRoutes.POST("", giftHandler.Create, requireEditor)
	giftRoutes.PUT("/:id", giftHandler.Update, requireEditor)
	giftRoutes.DELETE("/:id", giftHandler.Delete, requireEditor)

	relationshipRoutes := contactSub.Group("/relationships")
	relationshipRoutes.GET("", relationshipHandler.List)
	relationshipRoutes.GET("/graph", relationshipHandler.GetContactGraph)
	relationshipRoutes.GET("/kinship/:related_contact_id", relationshipHandler.CalculateKinship)
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

	contactSub.POST("/photos", vaultFileHandler.UploadContactFile, requireEditor)
	contactSub.GET("/photos", contactPhotoHandler.List)
	contactSub.GET("/photos/:photoId", contactPhotoHandler.Get)
	contactSub.DELETE("/photos/:photoId", contactPhotoHandler.Delete, requireEditor)
	contactSub.POST("/documents", vaultFileHandler.UploadContactFile, requireEditor)
	contactSub.GET("/documents", contactDocumentHandler.List)
	contactSub.DELETE("/documents/:id", contactDocumentHandler.Delete, requireEditor)
	contactSub.GET("/avatar", avatarHandler.GetAvatar)
	contactSub.GET("/companies/list", companyHandler.ListForContact)
	contactSub.PUT("/quickFacts/toggle", quickFactHandler.Toggle, requireEditor)

	quickFactRoutes := contactSub.Group("/quickFacts")
	quickFactRoutes.GET("", quickFactHandler.ListAll)
	quickFactRoutes.GET("/:templateId", quickFactHandler.List)
	quickFactRoutes.POST("/:templateId", quickFactHandler.Create, requireEditor)
	quickFactRoutes.POST("/:templateId/file", quickFactHandler.UploadFile, requireEditor)
	quickFactRoutes.PUT("/:templateId/:id", quickFactHandler.Update, requireEditor)
	quickFactRoutes.PUT("/:templateId/:id/file", quickFactHandler.ReplaceFile, requireEditor)
	quickFactRoutes.DELETE("/:templateId/:id", quickFactHandler.Delete, requireEditor)

	vaultScoped := protected.Group("/vaults/:vault_id", VaultPermissionMiddleware(vaultService, models.PermissionViewer))
	contactLayouts := vaultScoped.Group("/contact-layout")
	contactLayouts.GET("/modules", contactLayoutHandler.Modules)
	contactLayouts.GET("/templates", contactLayoutHandler.List)
	contactLayouts.GET("/templates/:template_id", contactLayoutHandler.Get)
	contactLayoutManagers := contactLayouts.Group("", VaultPermissionMiddleware(vaultService, models.PermissionManager))
	contactLayoutManagers.POST("/templates", contactLayoutHandler.Create)
	contactLayoutManagers.PUT("/templates/:template_id", contactLayoutHandler.Rename)
	contactLayoutManagers.PUT("/templates/:template_id/layout", contactLayoutHandler.Save)
	contactLayoutManagers.PUT("/templates/:template_id/default", contactLayoutHandler.SetDefault)
	contactLayoutManagers.DELETE("/templates/:template_id", contactLayoutHandler.Delete)
	vaultScoped.POST("/groups", groupHandler.Create, requireEditor)
	vaultScoped.GET("/groups", groupHandler.List)
	vaultScoped.GET("/groups/:id", groupHandler.Get)
	vaultScoped.PUT("/groups/:id", groupHandler.Update, requireEditor)
	vaultScoped.DELETE("/groups/:id", groupHandler.Delete, requireEditor)
	vaultScoped.POST("/groups/:id/members", groupHandler.AddMembers, requireEditor)
	vaultScoped.DELETE("/groups/:id/members", groupHandler.RemoveMembers, requireEditor)

	contacts.GET("/:contact_id/groups", groupHandler.ListContactGroups)
	contacts.POST("/:contact_id/groups", groupHandler.AddContactToGroup, requireEditor)
	contacts.DELETE("/:contact_id/groups/:id", groupHandler.RemoveContactFromGroup, requireEditor)

	journalRoutes := vaultScoped.Group("/journals")
	journalRoutes.GET("", journalHandler.List)
	journalRoutes.POST("", journalHandler.Create, requireEditor)
	journalRoutes.GET("/:id", journalHandler.Get)
	journalRoutes.PUT("/:id", journalHandler.Update, requireEditor)
	journalRoutes.DELETE("/:id", journalHandler.Delete, requireEditor)

	journalRoutes.GET("/:id/photos", journalHandler.GetPhotos)
	journalRoutes.GET("/:id/years/:year", journalHandler.GetByYear)

	journalMetricRoutes := vaultScoped.Group("/journals/:journal_id/metrics")
	journalMetricRoutes.GET("", journalMetricHandler.List)
	journalMetricRoutes.POST("", journalMetricHandler.Create, requireEditor)
	journalMetricRoutes.DELETE("/:id", journalMetricHandler.Delete, requireEditor)

	sliceRoutes := vaultScoped.Group("/journals/:journal_id/slices")
	sliceRoutes.GET("", sliceOfLifeHandler.List)
	sliceRoutes.POST("", sliceOfLifeHandler.Create, requireEditor)
	sliceRoutes.GET("/:id", sliceOfLifeHandler.Get)
	sliceRoutes.PUT("/:id", sliceOfLifeHandler.Update, requireEditor)
	sliceRoutes.DELETE("/:id", sliceOfLifeHandler.Delete, requireEditor)
	sliceRoutes.PUT("/:id/cover", sliceOfLifeHandler.UpdateCover, requireEditor)
	sliceRoutes.DELETE("/:id/cover", sliceOfLifeHandler.RemoveCover, requireEditor)

	postRoutes := vaultScoped.Group("/journals/:journal_id/posts")
	postRoutes.GET("", postHandler.List)
	postRoutes.POST("", postHandler.Create, requireEditor)
	postRoutes.GET("/:id", postHandler.Get)
	postRoutes.PUT("/:id", postHandler.Update, requireEditor)
	postRoutes.DELETE("/:id", postHandler.Delete, requireEditor)
	postRoutes.GET("/:id/metrics", postMetricHandler.List)
	postRoutes.POST("/:id/metrics", postMetricHandler.Create, requireEditor)
	postRoutes.DELETE("/:id/metrics/:metricId", postMetricHandler.Delete, requireEditor)
	postRoutes.GET("/:id/tags", postTagHandler.List)
	postRoutes.POST("/:id/tags", postTagHandler.Add, requireEditor)
	postRoutes.PUT("/:id/tags/:tagId", postTagHandler.Update, requireEditor)
	postRoutes.DELETE("/:id/tags/:tagId", postTagHandler.Remove, requireEditor)
	postRoutes.PUT("/:id/slices", postHandler.SetSlice, requireEditor)
	postRoutes.DELETE("/:id/slices", postHandler.ClearSlice, requireEditor)
	postRoutes.GET("/:id/photos", postPhotoHandler.List)
	postRoutes.POST("/:id/photos", postPhotoHandler.Upload, requireEditor)
	postRoutes.DELETE("/:id/photos/:photoId", postPhotoHandler.Delete, requireEditor)

	vaultScoped.GET("/tasks", vaultTaskHandler.List)
	vaultScoped.POST("/tasks", vaultTaskHandler.Create, requireEditor)
	vaultScoped.PATCH("/tasks/:id", vaultTaskHandler.Update, requireEditor)
	vaultScoped.DELETE("/tasks/:id", vaultTaskHandler.Delete, requireEditor)
	vaultScoped.PATCH("/tasks/:id/status", vaultTaskHandler.UpdateStatus, requireEditor)
	vaultScoped.PATCH("/tasks/:id/position", vaultTaskHandler.UpdatePosition, requireEditor)

	vaultScoped.GET("/files", vaultFileHandler.List)
	vaultScoped.POST("/files", vaultFileHandler.Upload, requireEditor)
	vaultScoped.GET("/files/:id/download", vaultFileHandler.Serve)
	vaultScoped.DELETE("/files/:id", vaultFileHandler.Delete, requireEditor)

	vaultScoped.GET("/companies", companyHandler.List)
	vaultScoped.POST("/companies", companyHandler.Create, requireEditor)
	vaultScoped.GET("/companies/:id", companyHandler.Get)
	vaultScoped.PUT("/companies/:id", companyHandler.Update, requireEditor)
	vaultScoped.DELETE("/companies/:id", companyHandler.Delete, requireEditor)
	vaultScoped.POST("/companies/:id/employees", companyHandler.AddEmployee, requireEditor)
	vaultScoped.DELETE("/companies/:id/employees/:contact_id", companyHandler.RemoveEmployee, requireEditor)

	vaultScoped.GET("/files/photos", vaultFileHandler.ListPhotos)
	vaultScoped.GET("/files/documents", vaultFileHandler.ListDocuments)
	vaultScoped.GET("/files/avatars", vaultFileHandler.ListAvatars)

	vaultScoped.GET("/calendar", calendarHandler.Get)
	vaultScoped.GET("/calendar/years/:year/months/:month", calendarHandler.GetMonth)
	vaultScoped.GET("/calendar/years/:year/months/:month/days/:day", calendarHandler.GetDay)

	// Registered on api, not protected: protected's DenyScopedPAT would reject
	// scoped tokens, but this feed must be reachable by a calendar:read PAT.
	icsCalendar := api.Group("/vaults/:vault_id/calendar.ics",
		authMiddleware.Authenticate,
		middleware.RequireEmailVerification(emailVerificationRequired),
		middleware.RequireScope(middleware.ScopeCalendarRead),
		VaultPermissionMiddleware(vaultService, models.PermissionViewer),
	)
	icsCalendar.GET("", calendarHandler.GetICS)

	// Address lookup is vault-scoped rather than per-contact: it reads nothing
	// from a contact. It is gated on Editor even though it reads nothing,
	// because every lookup spends a request against the instance's geocoding
	// quota — a viewer who cannot save an address has no use for it.
	vaultScoped.GET("/addresses/suggest", addressHandler.Suggest, requireEditor)

	vaultScoped.GET("/reports", reportHandler.Index)
	vaultScoped.GET("/reports/overview", reportHandler.Overview)
	vaultScoped.GET("/reports/addresses", reportHandler.Addresses)
	vaultScoped.GET("/reports/addresses/city/:city", reportHandler.AddressesByCity)
	vaultScoped.GET("/reports/addresses/country/:country", reportHandler.AddressesByCountry)
	vaultScoped.GET("/reports/importantDates", reportHandler.ImportantDates)
	vaultScoped.GET("/reports/moodTrackingEvents", reportHandler.MoodTrackingEvents)
	vaultScoped.GET("/reports/demographics", reportHandler.Demographics)
	vaultScoped.GET("/reports/map", reportHandler.Map)
	vaultScoped.GET("/reports/interactions", reportHandler.Interactions)
	vaultScoped.POST("/moodTrackingEvents", moodTrackingHandler.Create)
	vaultScoped.GET("/moodTrackingEvents", moodTrackingHandler.List)

	vaultScoped.GET("/reminders", vaultReminderHandler.List)

	vaultScoped.GET("/lifeMetrics", lifeMetricHandler.List)
	vaultScoped.POST("/lifeMetrics", lifeMetricHandler.Create, requireEditor)
	vaultScoped.PUT("/lifeMetrics/:id", lifeMetricHandler.Update, requireEditor)
	vaultScoped.DELETE("/lifeMetrics/:id", lifeMetricHandler.Delete, requireEditor)
	vaultScoped.POST("/lifeMetrics/:id/increment", lifeMetricHandler.Increment, requireEditor)
	vaultScoped.GET("/lifeMetrics/:id/detail", lifeMetricHandler.GetDetail)

	vaultScoped.GET("/activities", activityHandler.List)
	vaultScoped.GET("/activities/:id", activityHandler.Get)
	vaultScoped.POST("/activities", activityHandler.Create, requireEditor)
	vaultScoped.PUT("/activities/:id", activityHandler.Update, requireEditor)
	vaultScoped.DELETE("/activities/:id", activityHandler.Delete, requireEditor)
	vaultScoped.GET("/dashboard/catchUp", contactHandler.ListCatchUpPrompts)

	davSubs := vaultScoped.Group("/dav/subscriptions", VaultPermissionMiddleware(vaultService, models.PermissionManager))
	davSubs.GET("", davClientHandler.List)
	davSubs.POST("", davClientHandler.Create)
	davSubs.POST("/test", davClientHandler.TestConnection)
	davSubs.GET("/:sub_id", davClientHandler.Get)
	davSubs.PUT("/:sub_id", davClientHandler.Update)
	davSubs.DELETE("/:sub_id", davClientHandler.Delete)
	davSubs.POST("/:sub_id/sync", davClientHandler.TriggerSync)
	davSubs.GET("/:sub_id/logs", davClientHandler.GetSyncLogs)

	vaultScoped.GET("/feed", feedHandler.Get)
	vaultScoped.GET("/search", searchHandler.Search)
	vaultScoped.GET("/search/mostConsulted", mostConsultedHandler.List)
	vaultScoped.POST("/search/contacts", contactHandler.QuickSearch)

	settingsGroup := protected.Group("/settings")

	prefsGroup := settingsGroup.Group("/preferences")
	prefsGroup.GET("", preferenceHandler.Get)
	prefsGroup.PUT("", preferenceHandler.UpdateAll)
	prefsGroup.POST("/name", preferenceHandler.UpdateNameOrder)
	prefsGroup.POST("/date", preferenceHandler.UpdateDateFormat)
	prefsGroup.POST("/timezone", preferenceHandler.UpdateTimezone)
	prefsGroup.POST("/locale", preferenceHandler.UpdateLocale)
	prefsGroup.POST("/number", preferenceHandler.UpdateNumberFormat)
	prefsGroup.POST("/distance", preferenceHandler.UpdateDistanceFormat)
	prefsGroup.POST("/maps", preferenceHandler.UpdateMapsPreference)
	prefsGroup.POST("/help", preferenceHandler.UpdateHelpShown)

	notifGroup := settingsGroup.Group("/notifications")
	notifGroup.GET("", notificationHandler.List)
	notifGroup.POST("", notificationHandler.Create)
	notifGroup.PUT("/:id", notificationHandler.Update)
	notifGroup.PUT("/:id/toggle", notificationHandler.Toggle)
	notifGroup.DELETE("/:id", notificationHandler.Delete)
	notifGroup.GET("/:id/verify/:token", notificationHandler.Verify)
	notifGroup.POST("/:id/test", notificationHandler.SendTest)
	notifGroup.GET("/:id/logs", notificationHandler.ListLogs)

	// Account-level reference data is read by every account member throughout
	// contact and activity screens. Only account administrators may mutate it.
	personalizeGroup := settingsGroup.Group("/personalize")
	personalizeGroup.GET("/:entity", personalizeHandler.List)

	ptSectionGroup := personalizeGroup.Group("/post-templates/:id/sections")
	ptSectionGroup.GET("", postTemplateSectionHandler.List)

	gtRoleGroup := personalizeGroup.Group("/group-types/:id/roles")
	gtRoleGroup.GET("", groupTypeRoleHandler.List)

	// Static route must be registered before parameterized /:id route to avoid
	// "all" being captured as an :id parameter value.
	personalizeGroup.GET("/relationship-types/all", relationshipTypeHandler.ListAll)

	rtTypeGroup := personalizeGroup.Group("/relationship-types/:id/types")
	rtTypeGroup.GET("", relationshipTypeHandler.List)

	crGroup := personalizeGroup.Group("/call-reasons/:id/reasons")
	crGroup.GET("", callReasonHandler.List)

	personalizeAdminGroup := settingsGroup.Group("/personalize", authMiddleware.RequireAdmin)
	personalizeAdminGroup.POST("/sync", personalizeHandler.SyncTranslations)
	personalizeAdminGroup.PUT("/currencies/:currencyId/toggle", currencyHandler.Toggle)
	personalizeAdminGroup.POST("/currencies/enable-all", currencyHandler.EnableAll)
	personalizeAdminGroup.DELETE("/currencies/disable-all", currencyHandler.DisableAll)
	personalizeAdminGroup.POST("/:entity", personalizeHandler.Create)
	personalizeAdminGroup.PUT("/:entity/:id", personalizeHandler.Update)
	personalizeAdminGroup.DELETE("/:entity/:id", personalizeHandler.Delete)
	personalizeAdminGroup.POST("/:entity/:id/position", personalizeHandler.UpdatePosition)

	ptSectionAdminGroup := personalizeAdminGroup.Group("/post-templates/:id/sections")
	ptSectionAdminGroup.POST("", postTemplateSectionHandler.Create)
	ptSectionAdminGroup.PUT("/:sectionId", postTemplateSectionHandler.Update)
	ptSectionAdminGroup.DELETE("/:sectionId", postTemplateSectionHandler.Delete)
	ptSectionAdminGroup.POST("/:sectionId/position", postTemplateSectionHandler.UpdatePosition)

	gtRoleAdminGroup := personalizeAdminGroup.Group("/group-types/:id/roles")
	gtRoleAdminGroup.POST("", groupTypeRoleHandler.Create)
	gtRoleAdminGroup.PUT("/:roleId", groupTypeRoleHandler.Update)
	gtRoleAdminGroup.DELETE("/:roleId", groupTypeRoleHandler.Delete)
	gtRoleAdminGroup.POST("/:roleId/position", groupTypeRoleHandler.UpdatePosition)

	rtTypeAdminGroup := personalizeAdminGroup.Group("/relationship-types/:id/types")
	rtTypeAdminGroup.POST("", relationshipTypeHandler.Create)
	rtTypeAdminGroup.PUT("/:typeId", relationshipTypeHandler.Update)
	rtTypeAdminGroup.DELETE("/:typeId", relationshipTypeHandler.Delete)

	crAdminGroup := personalizeAdminGroup.Group("/call-reasons/:id/reasons")
	crAdminGroup.POST("", callReasonHandler.Create)
	crAdminGroup.PUT("/:reasonId", callReasonHandler.Update)
	crAdminGroup.DELETE("/:reasonId", callReasonHandler.Delete)

	webauthnGroup := settingsGroup.Group("/webauthn")
	webauthnGroup.POST("/register/begin", webauthnHandler.BeginRegistration)
	webauthnGroup.POST("/register/finish", webauthnHandler.FinishRegistration)
	webauthnGroup.GET("/credentials", webauthnHandler.ListCredentials)
	webauthnGroup.DELETE("/credentials/:id", webauthnHandler.DeleteCredential)

	inviteGroup := settingsGroup.Group("/invitations", authMiddleware.RequireAdmin)
	inviteGroup.GET("", invitationHandler.List)
	inviteGroup.POST("", invitationHandler.Create)
	inviteGroup.DELETE("/:id", invitationHandler.Delete)

	twoFactorGroup := settingsGroup.Group("/2fa")
	twoFactorGroup.POST("/enable", twoFactorHandler.Enable)
	twoFactorGroup.POST("/confirm", twoFactorHandler.Confirm)
	twoFactorGroup.POST("/disable", twoFactorHandler.Disable)
	twoFactorGroup.GET("/status", twoFactorHandler.Status)

	tokenGroup := settingsGroup.Group("/tokens")
	tokenGroup.GET("", patHandler.List)
	tokenGroup.POST("", patHandler.Create)
	tokenGroup.DELETE("/:id", patHandler.Delete)

	usersGroup := settingsGroup.Group("/users", authMiddleware.RequireAdmin)
	usersGroup.GET("", userManagementHandler.List)
	usersGroup.GET("/:id", userManagementHandler.Get)
	usersGroup.PUT("/:id", userManagementHandler.Update)
	usersGroup.DELETE("/:id", userManagementHandler.Delete)

	oauthGroup := settingsGroup.Group("/oauth")
	oauthGroup.GET("", oauthHandler.ListProviders)
	oauthGroup.DELETE("/:driver", oauthHandler.UnlinkProvider)

	settingsGroup.DELETE("/account", accountCancelHandler.Cancel, authMiddleware.RequireAdmin)
	settingsGroup.GET("/storage", storageInfoHandler.Get)

	protected.GET("/currencies", currencyHandler.List)
	protected.GET("/pet-categories", petHandler.ListCategories)

	vaultSettings := vaultScoped.Group("/settings", VaultPermissionMiddleware(vaultService, models.PermissionManager))
	vaultSettings.GET("", vaultSettingsHandler.Get)
	vaultSettings.PUT("", vaultSettingsHandler.Update)
	vaultSettings.PUT("/visibility", vaultSettingsHandler.UpdateVisibility)

	vaultSettings.GET("/users", vaultSettingsHandler.ListUsers)
	vaultSettings.POST("/users", vaultSettingsHandler.AddUser)
	vaultSettings.PUT("/users/:id", vaultSettingsHandler.UpdateUserPermission)
	vaultSettings.DELETE("/users/:id", vaultSettingsHandler.RemoveUser)

	vaultSettings.GET("/labels", vaultSettingsHandler.ListLabels)
	vaultSettings.POST("/labels", vaultSettingsHandler.CreateLabel)
	vaultSettings.PUT("/labels/:id", vaultSettingsHandler.UpdateLabel)
	vaultSettings.DELETE("/labels/:id", vaultSettingsHandler.DeleteLabel)

	vaultSettings.GET("/tags", vaultSettingsHandler.ListTags)
	vaultSettings.POST("/tags", vaultSettingsHandler.CreateTag)
	vaultSettings.PUT("/tags/:id", vaultSettingsHandler.UpdateTag)
	vaultSettings.DELETE("/tags/:id", vaultSettingsHandler.DeleteTag)

	vaultSettings.GET("/dateTypes", vaultSettingsHandler.ListDateTypes)
	vaultSettings.POST("/dateTypes", vaultSettingsHandler.CreateDateType)
	vaultSettings.PUT("/dateTypes/:id", vaultSettingsHandler.UpdateDateType)
	vaultSettings.DELETE("/dateTypes/:id", vaultSettingsHandler.DeleteDateType)

	vaultSettings.GET("/moodParams", vaultSettingsHandler.ListMoodParams)
	vaultSettings.POST("/moodParams", vaultSettingsHandler.CreateMoodParam)
	vaultSettings.PUT("/moodParams/:id", vaultSettingsHandler.UpdateMoodParam)
	vaultSettings.POST("/moodParams/:id/position", vaultSettingsHandler.UpdateMoodParamOrder)
	vaultSettings.DELETE("/moodParams/:id", vaultSettingsHandler.DeleteMoodParam)

	vaultSettings.GET("/activityCategories", vaultSettingsHandler.ListActivityCategories)
	vaultSettings.POST("/activityCategories", vaultSettingsHandler.CreateActivityCategory)
	vaultSettings.PUT("/activityCategories/:id", vaultSettingsHandler.UpdateActivityCategory)
	vaultSettings.POST("/activityCategories/:id/position", vaultSettingsHandler.UpdateActivityCategoryOrder)
	vaultSettings.DELETE("/activityCategories/:id", vaultSettingsHandler.DeleteActivityCategory)
	vaultSettings.POST("/activityCategories/:categoryId/types", vaultSettingsHandler.CreateActivityType)
	vaultSettings.PUT("/activityCategories/:categoryId/types/:typeId", vaultSettingsHandler.UpdateActivityType)
	vaultSettings.DELETE("/activityCategories/:categoryId/types/:typeId", vaultSettingsHandler.DeleteActivityType)
	vaultSettings.POST("/activityCategories/:categoryId/activityTypes", vaultSettingsHandler.CreateActivityType)
	vaultSettings.PUT("/activityCategories/:categoryId/activityTypes/:typeId", vaultSettingsHandler.UpdateActivityType)
	vaultSettings.POST("/activityCategories/:categoryId/activityTypes/:typeId/position", vaultSettingsHandler.UpdateActivityTypeOrder)
	vaultSettings.DELETE("/activityCategories/:categoryId/activityTypes/:typeId", vaultSettingsHandler.DeleteActivityType)

	vaultSettings.GET("/quickFactTemplates", vaultSettingsHandler.ListQuickFactTemplates)
	vaultSettings.POST("/quickFactTemplates", vaultSettingsHandler.CreateQuickFactTemplate)
	vaultSettings.PUT("/quickFactTemplates/:id", vaultSettingsHandler.UpdateQuickFactTemplate)
	vaultSettings.POST("/quickFactTemplates/:id/position", vaultSettingsHandler.UpdateQuickFactTemplateOrder)
	vaultSettings.DELETE("/quickFactTemplates/:id", vaultSettingsHandler.DeleteQuickFactTemplate)

	vaultSettings.POST("/import/monica", monicaImportHandler.Import)
	vaultSettings.POST("/import/csv", csvImportHandler.Import)

	mcpRegistry := internalmcp.NewActionRegistry(e)
	mcpExecutor := internalmcp.NewActionExecutor(e, mcpRegistry)
	mcpSearcher := internalmcp.NewBondsSearcher(db, searchService, vaultService)
	mcpFetcher := internalmcp.NewResourceFetcher(db, vaultService)
	mcpHandler := internalmcp.NewHandler(db, mcpRegistry, mcpExecutor, mcpSearcher, mcpFetcher)
	mcpMiddleware := []echo.MiddlewareFunc{internalmcp.RequireAllowedOrigin(cfg.App.URL, "http://localhost:5173", "http://localhost:3000"), authMiddleware.Authenticate, middleware.RequireEmailVerification(emailVerificationRequired)}
	e.POST("/mcp", mcpHandler.Handle, mcpMiddleware...)
	e.GET("/mcp", mcpHandler.MethodNotAllowed, mcpMiddleware...)
	e.DELETE("/mcp", mcpHandler.MethodNotAllowed, mcpMiddleware...)
}
