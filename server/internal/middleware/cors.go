package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
)

var corsAllowOrigins = []string{"http://localhost:5173", "http://localhost:3000"}

var corsAllowMethods = []string{
	http.MethodGet,
	http.MethodPost,
	http.MethodPut,
	http.MethodDelete,
	http.MethodPatch,
	http.MethodOptions,
}

var corsAllowHeaders = []string{
	"Accept",
	"Authorization",
	"Content-Type",
	"X-Requested-With",
}

var davCORSAllowMethods = []string{
	http.MethodGet,
	http.MethodHead,
	http.MethodPut,
	http.MethodDelete,
	http.MethodOptions,
	"PROPFIND",
	"PROPPATCH",
	"REPORT",
	"MKCOL",
	"COPY",
	"MOVE",
}

var davCORSAllowHeaders = []string{
	"Accept",
	"Authorization",
	"Content-Type",
	"Depth",
	"Destination",
	"If",
	"Lock-Token",
	"Overwrite",
	"Timeout",
	"X-Requested-With",
}

func CORS() echo.MiddlewareFunc {
	defaultCORS := echoMiddleware.CORSWithConfig(echoMiddleware.CORSConfig{
		AllowOrigins:     corsAllowOrigins,
		AllowMethods:     corsAllowMethods,
		AllowHeaders:     corsAllowHeaders,
		AllowCredentials: true,
		MaxAge:           86400,
	})

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		defaultHandler := defaultCORS(next)
		return func(c echo.Context) error {
			req := c.Request()
			if req.Method == http.MethodOptions && strings.HasPrefix(req.URL.Path, "/dav") {
				// Let DAV OPTIONS reach go-webdav so discovery keeps DAV/Allow headers while still adding CORS metadata.
				applyDAVCORSHeaders(c)
				return next(c)
			}
			return defaultHandler(c)
		}
	}
}

func applyDAVCORSHeaders(c echo.Context) {
	req := c.Request()
	res := c.Response()
	origin := req.Header.Get(echo.HeaderOrigin)

	res.Header().Add(echo.HeaderVary, echo.HeaderOrigin)
	if origin == "" || !isCORSOriginAllowed(origin) {
		return
	}

	res.Header().Set(echo.HeaderAccessControlAllowOrigin, origin)
	res.Header().Set(echo.HeaderAccessControlAllowCredentials, "true")
	res.Header().Set(echo.HeaderAccessControlAllowMethods, strings.Join(davCORSAllowMethods, ","))
	res.Header().Set(echo.HeaderAccessControlAllowHeaders, strings.Join(davCORSAllowHeaders, ","))
	res.Header().Set(echo.HeaderAccessControlMaxAge, "86400")
	res.Header().Add(echo.HeaderVary, echo.HeaderAccessControlRequestMethod)
	res.Header().Add(echo.HeaderVary, echo.HeaderAccessControlRequestHeaders)
}

func isCORSOriginAllowed(origin string) bool {
	for _, allowedOrigin := range corsAllowOrigins {
		if origin == allowedOrigin {
			return true
		}
	}
	return false
}
