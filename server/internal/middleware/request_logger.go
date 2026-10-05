package middleware

import (
	"github.com/labstack/echo/v5"
	echomiddleware "github.com/labstack/echo/v5/middleware"
)

// RequestLogger excludes query strings, headers and raw errors. OAuth codes,
// login tokens and PKCE material must not be copied into operational logs.
func RequestLogger() echo.MiddlewareFunc {
	return echomiddleware.RequestLoggerWithConfig(echomiddleware.RequestLoggerConfig{
		LogMethod: true, LogRoutePath: true, LogStatus: true,
		LogLatency: true, LogRequestID: true, HandleError: true,
		LogValuesFunc: func(c *echo.Context, v echomiddleware.RequestLoggerValues) error {
			c.Logger().Info("REQUEST", "method", v.Method, "route", v.RoutePath,
				"status", v.Status, "latency", v.Latency, "request_id", v.RequestID)
			return nil
		},
	})
}
