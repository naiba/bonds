package dav

import (
	"net/http"
	"strings"

	"github.com/emersion/go-webdav"
	"github.com/emersion/go-webdav/caldav"
	"github.com/emersion/go-webdav/carddav"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// SetupDAVRoutes registers CardDAV and CalDAV routes on the Echo instance.
func SetupDAVRoutes(e *echo.Echo, db *gorm.DB) {
	cardBackend := NewCardDAVBackend(db)
	calBackend := NewCalDAVBackend(db)

	cardHandler := &carddav.Handler{Backend: cardBackend, Prefix: "/dav"}
	calHandler := &caldav.Handler{Backend: calBackend, Prefix: "/dav"}

	authMw := BasicAuthMiddleware(db)

	davHandler := authMw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.Contains(path, "/addressbooks/") {
			cardHandler.ServeHTTP(w, r)
		} else if strings.Contains(path, "/calendars/") {
			calHandler.ServeHTTP(w, r)
		} else if strings.Contains(path, "/principals/") {
			serveDAVPrincipal(w, r, cardBackend, calBackend)
		} else {
			// Default: serve CardDAV for discovery
			cardHandler.ServeHTTP(w, r)
		}
	}))

	// Mount under /dav/*
	davGroup := e.Group("/dav")
	davGroup.Any("/*", echo.WrapHandler(davHandler))
	davGroup.Any("", echo.WrapHandler(davHandler))

	// Well-known discovery endpoints
	e.Any("/.well-known/carddav", func(c echo.Context) error {
		return c.Redirect(http.StatusMovedPermanently, "/dav/")
	})
	e.Any("/.well-known/caldav", func(c echo.Context) error {
		return c.Redirect(http.StatusMovedPermanently, "/dav/")
	})
}

func serveDAVPrincipal(w http.ResponseWriter, r *http.Request, cardBackend *CardDAVBackend, calBackend *CalDAVBackend) {
	if r.Method == http.MethodOptions {
		w.Header().Add("DAV", "1, 3, addressbook, calendar-access")
		w.Header().Add("Allow", "OPTIONS, PROPFIND, REPORT, DELETE, MKCOL")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	principalPath, err := cardBackend.CurrentUserPrincipal(r.Context())
	if err != nil {
		http.Error(w, "dav: failed to determine current user principal", http.StatusInternalServerError)
		return
	}
	addressBookHomeSetPath, err := cardBackend.AddressBookHomeSetPath(r.Context())
	if err != nil {
		http.Error(w, "dav: failed to determine address book home set", http.StatusInternalServerError)
		return
	}
	calendarHomeSetPath, err := calBackend.CalendarHomeSetPath(r.Context())
	if err != nil {
		http.Error(w, "dav: failed to determine calendar home set", http.StatusInternalServerError)
		return
	}

	webdav.ServePrincipal(w, r, &webdav.ServePrincipalOptions{
		CurrentUserPrincipalPath: principalPath,
		HomeSets: []webdav.BackendSuppliedHomeSet{
			carddav.NewAddressBookHomeSet(addressBookHomeSetPath),
			caldav.NewCalendarHomeSet(calendarHomeSetPath),
		},
		Capabilities: []webdav.Capability{
			carddav.CapabilityAddressBook,
			caldav.CapabilityCalendar,
		},
	})
}
