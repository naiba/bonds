package handlers

import (
	"log"

	"github.com/labstack/echo/v4"
	echoSwagger "github.com/swaggo/echo-swagger"

	"github.com/naiba/bonds/internal/config"
	"github.com/naiba/bonds/internal/middleware"
	"github.com/naiba/bonds/internal/models"
	"github.com/naiba/bonds/internal/search"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
	"gorm.io/gorm"

	_ "github.com/naiba/bonds/docs"
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
	contactLabelService := services.NewContactLabelService(db)
	contactReligionService := services.NewContactReligionService(db)
	contactJobService := services.NewContactJobService(db)
	contactMoveService := services.NewContactMoveService(db)
	contactTemplateService := services.NewContactTemplateService(db)
	contactTabService := services.NewContactTabService(db)
	contactSortService := services.NewContactSortService(db)
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
	vaultLifeEventSettingsService := services.NewVaultLifeEventService(db)
	vaultQuickFactTplService := services.NewVaultQuickFactTemplateService(db)
	userManagementService := services.NewUserManagementService(db)
	accountCancelService := services.NewAccountCancelService(db)
	storageInfoService := services.NewStorageInfoService(db)
	currencyService := services.NewCurrencyService(db)
	templatePageService := services.NewTemplatePageService(db)

	mailer, mErr := services.NewSMTPMailer(&cfg.SMTP)
	if mErr != nil {
		log.Printf("WARNING: Failed to initialize mailer: %v", mErr)
		mailer = &services.NoopMailer{}
	}
	invitationService := services.NewInvitationService(db, mailer, cfg.App.URL)
	notificationService.SetMailer(mailer, cfg.App.URL)

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

	postPhotoHandler := NewPostPhotoHandler(vaultFileService)
	contactPhotoHandler := NewContactPhotoHandler(vaultFileService)
	contactDocumentHandler := NewContactDocumentHandler(vaultFileService)

	telegramWebhookService := services.NewTelegramWebhookService(db)

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
	contactLabelHandler := NewContactLabelHandler(contactLabelService)
	contactReligionHandler := NewContactReligionHandler(contactReligionService)
	contactJobHandler := NewContactJobHandler(contactJobService)
	contactMoveHandler := NewContactMoveHandler(contactMoveService)
	contactTemplateHandler := NewContactTemplateHandler(contactTemplateService)
	contactTabHandler := NewContactTabHandler(contactTabService)
	contactSortHandler := NewContactSortHandler(contactSortService)
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
		vaultDateTypeService, vaultMoodParamService, vaultLifeEventSettingsService, vaultQuickFactTplService,
	)
	userManagementHandler := NewUserManagementHandler(userManagementService)
	accountCancelHandler := NewAccountCancelHandler(accountCancelService)
	storageInfoHandler := NewStorageInfoHandler(storageInfoService)
	currencyHandler := NewCurrencyHandler(currencyService)
	templatePageHandler := NewTemplatePageHandler(templatePageService)
	telegramWebhookHandler := NewTelegramWebhookHandler(telegramWebhookService)

	e.Use(middleware.CORS())

	if cfg.Debug {
		e.GET("/swagger/*", echoSwagger.WrapHandler)
	}

	api := e.Group("/api")

	api.GET("/announcement", func(c echo.Context) error {
		return response.OK(c, map[string]string{"content": cfg.Announcement})
	})

	if cfg.Telegram.BotToken != "" {
		api.POST("/telegram/webhook", telegramWebhookHandler.HandleWebhook)
	}

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
	contacts.GET("/labels/:labelId", contactHandler.ListByLabel)
	contacts.POST("", contactHandler.Create, requireEditor)
	contacts.GET("/:id", contactHandler.Get)
	contacts.PUT("/:id", contactHandler.Update, requireEditor)
	contacts.DELETE("/:id", contactHandler.Delete, requireEditor)
	contacts.PUT("/:id/archive", contactHandler.ToggleArchive, requireEditor)
	contacts.PUT("/:id/favorite", contactHandler.ToggleFavorite)
	contacts.GET("/export", vcardHandler.ExportVault)
	contacts.POST("/import", vcardHandler.ImportVCard, requireEditor)
	contacts.PUT("/sort", contactSortHandler.UpdateSort, requireEditor)

	contactSub := protected.Group("/vaults/:vault_id/contacts/:contact_id", VaultPermissionMiddleware(vaultService, models.PermissionViewer))
	contactSub.GET("/vcard", vcardHandler.ExportContact)
	contactSub.GET("/labels", contactLabelHandler.List)
	contactSub.POST("/labels", contactLabelHandler.Add, requireEditor)
	contactSub.PUT("/labels/:id", contactLabelHandler.Update, requireEditor)
	contactSub.DELETE("/labels/:id", contactLabelHandler.Remove, requireEditor)
	contactSub.PUT("/religion", contactReligionHandler.Update, requireEditor)
	contactSub.PUT("/jobInformation", contactJobHandler.Update, requireEditor)
	contactSub.DELETE("/jobInformation", contactJobHandler.Delete, requireEditor)
	contactSub.GET("/feed", feedHandler.GetContactFeed)
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
	timelineRoutes.PUT("/:id/toggle", lifeEventHandler.ToggleTimelineEvent, requireEditor)
	timelineRoutes.PUT("/:id/lifeEvents/:lifeEventId/toggle", lifeEventHandler.ToggleLifeEvent, requireEditor)
	timelineRoutes.DELETE("/:id", lifeEventHandler.DeleteTimelineEvent, requireEditor)
	timelineRoutes.DELETE("/:id/lifeEvents/:lifeEventId", lifeEventHandler.DeleteLifeEvent, requireEditor)

	moodRoutes := contactSub.Group("/moodTrackingEvents")
	moodRoutes.POST("", moodTrackingHandler.Create, requireEditor)
	moodRoutes.GET("", moodTrackingHandler.List)

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
	quickFactRoutes.GET("/:templateId", quickFactHandler.List)
	quickFactRoutes.POST("/:templateId", quickFactHandler.Create, requireEditor)
	quickFactRoutes.PUT("/:templateId/:id", quickFactHandler.Update, requireEditor)
	quickFactRoutes.DELETE("/:templateId/:id", quickFactHandler.Delete, requireEditor)

	vaultScoped := protected.Group("/vaults/:vault_id", VaultPermissionMiddleware(vaultService, models.PermissionViewer))
	vaultScoped.POST("/groups", groupHandler.Create, requireEditor)
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

	vaultScoped.GET("/files", vaultFileHandler.List)
	vaultScoped.POST("/files", vaultFileHandler.Upload, requireEditor)
	vaultScoped.GET("/files/:id/download", vaultFileHandler.Serve)
	vaultScoped.DELETE("/files/:id", vaultFileHandler.Delete, requireEditor)

	vaultScoped.GET("/companies", companyHandler.List)
	vaultScoped.GET("/companies/:id", companyHandler.Get)

	vaultScoped.GET("/files/photos", vaultFileHandler.ListPhotos)
	vaultScoped.GET("/files/documents", vaultFileHandler.ListDocuments)
	vaultScoped.GET("/files/avatars", vaultFileHandler.ListAvatars)

	vaultScoped.GET("/calendar", calendarHandler.Get)
	vaultScoped.GET("/calendar/years/:year/months/:month", calendarHandler.GetMonth)
	vaultScoped.GET("/calendar/years/:year/months/:month/days/:day", calendarHandler.GetDay)

	vaultScoped.GET("/reports", reportHandler.Index)
	vaultScoped.GET("/reports/addresses", reportHandler.Addresses)
	vaultScoped.GET("/reports/addresses/city/:city", reportHandler.AddressesByCity)
	vaultScoped.GET("/reports/addresses/country/:country", reportHandler.AddressesByCountry)
	vaultScoped.GET("/reports/importantDates", reportHandler.ImportantDates)
	vaultScoped.GET("/reports/moodTrackingEvents", reportHandler.MoodTrackingEvents)

	vaultScoped.GET("/reminders", vaultReminderHandler.List)

	vaultScoped.GET("/lifeMetrics", lifeMetricHandler.List)
	vaultScoped.POST("/lifeMetrics", lifeMetricHandler.Create, requireEditor)
	vaultScoped.PUT("/lifeMetrics/:id", lifeMetricHandler.Update, requireEditor)
	vaultScoped.DELETE("/lifeMetrics/:id", lifeMetricHandler.Delete, requireEditor)
	vaultScoped.POST("/lifeMetrics/:id/contacts", lifeMetricHandler.AddContact, requireEditor)

	vaultScoped.PUT("/defaultTab", vaultHandler.UpdateDefaultTab, requireEditor)

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
	notifGroup.PUT("/:id/toggle", notificationHandler.Toggle)
	notifGroup.DELETE("/:id", notificationHandler.Delete)
	notifGroup.GET("/:id/verify/:token", notificationHandler.Verify)
	notifGroup.POST("/:id/test", notificationHandler.SendTest)
	notifGroup.GET("/:id/logs", notificationHandler.ListLogs)

	personalizeGroup := settingsGroup.Group("/personalize", authMiddleware.RequireAdmin)
	personalizeGroup.PUT("/currencies/:currencyId/toggle", currencyHandler.Toggle)
	personalizeGroup.POST("/currencies/enable-all", currencyHandler.EnableAll)
	personalizeGroup.DELETE("/currencies/disable-all", currencyHandler.DisableAll)
	personalizeGroup.GET("/:entity", personalizeHandler.List)
	personalizeGroup.POST("/:entity", personalizeHandler.Create)
	personalizeGroup.PUT("/:entity/:id", personalizeHandler.Update)
	personalizeGroup.DELETE("/:entity/:id", personalizeHandler.Delete)
	personalizeGroup.POST("/:entity/:id/position", personalizeHandler.UpdatePosition)

	ptSectionGroup := personalizeGroup.Group("/post-templates/:id/sections")
	ptSectionGroup.GET("", postTemplateSectionHandler.List)
	ptSectionGroup.POST("", postTemplateSectionHandler.Create)
	ptSectionGroup.PUT("/:sectionId", postTemplateSectionHandler.Update)
	ptSectionGroup.DELETE("/:sectionId", postTemplateSectionHandler.Delete)
	ptSectionGroup.POST("/:sectionId/position", postTemplateSectionHandler.UpdatePosition)

	gtRoleGroup := personalizeGroup.Group("/group-types/:id/roles")
	gtRoleGroup.GET("", groupTypeRoleHandler.List)
	gtRoleGroup.POST("", groupTypeRoleHandler.Create)
	gtRoleGroup.PUT("/:roleId", groupTypeRoleHandler.Update)
	gtRoleGroup.DELETE("/:roleId", groupTypeRoleHandler.Delete)
	gtRoleGroup.POST("/:roleId/position", groupTypeRoleHandler.UpdatePosition)

	rtTypeGroup := personalizeGroup.Group("/relationship-types/:id/types")
	rtTypeGroup.GET("", relationshipTypeHandler.List)
	rtTypeGroup.POST("", relationshipTypeHandler.Create)
	rtTypeGroup.PUT("/:typeId", relationshipTypeHandler.Update)
	rtTypeGroup.DELETE("/:typeId", relationshipTypeHandler.Delete)

	crGroup := personalizeGroup.Group("/call-reasons/:id/reasons")
	crGroup.GET("", callReasonHandler.List)
	crGroup.POST("", callReasonHandler.Create)
	crGroup.PUT("/:reasonId", callReasonHandler.Update)
	crGroup.DELETE("/:reasonId", callReasonHandler.Delete)

	tpGroup := personalizeGroup.Group("/templates/:id/pages")
	tpGroup.GET("", templatePageHandler.List)
	tpGroup.POST("", templatePageHandler.Create)
	tpGroup.GET("/:pageId", templatePageHandler.Get)
	tpGroup.PUT("/:pageId", templatePageHandler.Update)
	tpGroup.DELETE("/:pageId", templatePageHandler.Delete)
	tpGroup.POST("/:pageId/position", templatePageHandler.UpdatePosition)
	tpGroup.GET("/:pageId/modules", templatePageHandler.ListModules)
	tpGroup.POST("/:pageId/modules", templatePageHandler.AddModule)
	tpGroup.DELETE("/:pageId/modules/:moduleId", templatePageHandler.RemoveModule)
	tpGroup.POST("/:pageId/modules/:moduleId/position", templatePageHandler.UpdateModulePosition)

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

	usersGroup := settingsGroup.Group("/users", authMiddleware.RequireAdmin)
	usersGroup.GET("", userManagementHandler.List)
	usersGroup.POST("", userManagementHandler.Create)
	usersGroup.GET("/:id", userManagementHandler.Get)
	usersGroup.PUT("/:id", userManagementHandler.Update)
	usersGroup.DELETE("/:id", userManagementHandler.Delete)

	oauthGroup := settingsGroup.Group("/oauth")
	oauthGroup.GET("", oauthHandler.ListProviders)
	oauthGroup.DELETE("/:driver", oauthHandler.UnlinkProvider)

	settingsGroup.DELETE("/account", accountCancelHandler.Cancel, authMiddleware.RequireAdmin)
	settingsGroup.GET("/storage", storageInfoHandler.Get)

	protected.GET("/currencies", currencyHandler.List)

	vaultSettings := vaultScoped.Group("/settings", VaultPermissionMiddleware(vaultService, models.PermissionManager))
	vaultSettings.GET("", vaultSettingsHandler.Get)
	vaultSettings.PUT("", vaultSettingsHandler.Update)
	vaultSettings.PUT("/template", vaultSettingsHandler.UpdateTemplate)
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

	vaultSettings.GET("/moodTrackingParameters", vaultSettingsHandler.ListMoodParams)
	vaultSettings.POST("/moodTrackingParameters", vaultSettingsHandler.CreateMoodParam)
	vaultSettings.PUT("/moodTrackingParameters/:id", vaultSettingsHandler.UpdateMoodParam)
	vaultSettings.PUT("/moodTrackingParameters/:id/order", vaultSettingsHandler.UpdateMoodParamOrder)
	vaultSettings.DELETE("/moodTrackingParameters/:id", vaultSettingsHandler.DeleteMoodParam)

	vaultSettings.GET("/lifeEventCategories", vaultSettingsHandler.ListLifeEventCategories)
	vaultSettings.POST("/lifeEventCategories", vaultSettingsHandler.CreateLifeEventCategory)
	vaultSettings.PUT("/lifeEventCategories/:id", vaultSettingsHandler.UpdateLifeEventCategory)
	vaultSettings.POST("/lifeEventCategories/:id/order", vaultSettingsHandler.UpdateLifeEventCategoryOrder)
	vaultSettings.DELETE("/lifeEventCategories/:id", vaultSettingsHandler.DeleteLifeEventCategory)
	vaultSettings.POST("/lifeEventCategories/:categoryId/lifeEventTypes", vaultSettingsHandler.CreateLifeEventType)
	vaultSettings.PUT("/lifeEventCategories/:categoryId/lifeEventTypes/:typeId", vaultSettingsHandler.UpdateLifeEventType)
	vaultSettings.POST("/lifeEventCategories/:categoryId/lifeEventTypes/:typeId/order", vaultSettingsHandler.UpdateLifeEventTypeOrder)
	vaultSettings.DELETE("/lifeEventCategories/:categoryId/lifeEventTypes/:typeId", vaultSettingsHandler.DeleteLifeEventType)

	vaultSettings.GET("/quickFactTemplates", vaultSettingsHandler.ListQuickFactTemplates)
	vaultSettings.POST("/quickFactTemplates", vaultSettingsHandler.CreateQuickFactTemplate)
	vaultSettings.PUT("/quickFactTemplates/:id", vaultSettingsHandler.UpdateQuickFactTemplate)
	vaultSettings.PUT("/quickFactTemplates/:id/order", vaultSettingsHandler.UpdateQuickFactTemplateOrder)
	vaultSettings.DELETE("/quickFactTemplates/:id", vaultSettingsHandler.DeleteQuickFactTemplate)
}
