package main

import (
	"context"
	"fmt"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/naiba/bonds/internal/config"
	"github.com/naiba/bonds/internal/cron"
	"github.com/naiba/bonds/internal/database"
	"github.com/naiba/bonds/internal/dav"
	"github.com/naiba/bonds/internal/frontend"
	"github.com/naiba/bonds/internal/handlers"
	appMiddleware "github.com/naiba/bonds/internal/middleware"
)

//	@title			Bonds API
//	@version		1.0
//	@description	Personal relationship manager RESTful API.

//	@contact.name	Bonds Team
//	@contact.url	https://github.com/naiba/bonds

//	@license.name	AGPL-3.0
//	@license.url	https://www.gnu.org/licenses/agpl-3.0.html

//	@BasePath	/api

//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				JWT Bearer token. Format: "Bearer {token}"

func main() {
	cfg := config.Load()

	db, err := database.Connect(&cfg.Database, cfg.Debug)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	if err := database.AutoMigrate(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	scheduler := cron.NewScheduler(db)
	scheduler.Start()

	e := echo.New()
	e.HideBanner = true

	if cfg.Debug {
		e.Use(echoMiddleware.Logger())
	}
	e.Use(echoMiddleware.Recover())
	e.Use(appMiddleware.Locale())

	handlers.RegisterRoutes(e, db, cfg)

	dav.SetupDAVRoutes(e, db)

	if frontend.HasDistFiles() {
		frontend.RegisterSPARoutes(e)
		log.Println("Serving embedded frontend")
	} else {
		log.Println("No embedded frontend found, API-only mode")
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	go func() {
		log.Printf("Starting server on %s", addr)
		if err := e.Start(addr); err != nil {
			log.Printf("Server stopped: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down...")

	cronCtx := scheduler.Stop()
	select {
	case <-cronCtx.Done():
		log.Println("Cron scheduler stopped")
	case <-time.After(30 * time.Second):
		log.Println("Cron scheduler stop timed out")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Server exited")
}
