package response

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/i18n"
)

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

type APIError struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

type Meta struct {
	Page       int   `json:"page"`
	PerPage    int   `json:"per_page"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// localize resolves a message key using the locale from the echo context.
func localize(c echo.Context, key string) string {
	lang := "en"
	if locale, ok := c.Get("locale").(string); ok && locale != "" {
		lang = locale
	}
	return i18n.T(lang, key)
}

func OK(c echo.Context, data interface{}) error {
	return c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    data,
	})
}

func Created(c echo.Context, data interface{}) error {
	return c.JSON(http.StatusCreated, APIResponse{
		Success: true,
		Data:    data,
	})
}

func Paginated(c echo.Context, data interface{}, meta Meta) error {
	return c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    data,
		Meta:    &meta,
	})
}

func BadRequest(c echo.Context, message string, details map[string]string) error {
	return c.JSON(http.StatusBadRequest, APIResponse{
		Success: false,
		Error: &APIError{
			Code:    "BAD_REQUEST",
			Message: localize(c, message),
			Details: details,
		},
	})
}

func Unauthorized(c echo.Context, message string) error {
	return c.JSON(http.StatusUnauthorized, APIResponse{
		Success: false,
		Error: &APIError{
			Code:    "UNAUTHORIZED",
			Message: localize(c, message),
		},
	})
}

func Forbidden(c echo.Context, message string) error {
	return c.JSON(http.StatusForbidden, APIResponse{
		Success: false,
		Error: &APIError{
			Code:    "FORBIDDEN",
			Message: localize(c, message),
		},
	})
}

func NotFound(c echo.Context, message string) error {
	return c.JSON(http.StatusNotFound, APIResponse{
		Success: false,
		Error: &APIError{
			Code:    "NOT_FOUND",
			Message: localize(c, message),
		},
	})
}

func Conflict(c echo.Context, message string) error {
	return c.JSON(http.StatusConflict, APIResponse{
		Success: false,
		Error: &APIError{
			Code:    "CONFLICT",
			Message: localize(c, message),
		},
	})
}

func InternalError(c echo.Context, message string) error {
	return c.JSON(http.StatusInternalServerError, APIResponse{
		Success: false,
		Error: &APIError{
			Code:    "INTERNAL_ERROR",
			Message: localize(c, message),
		},
	})
}

func ValidationError(c echo.Context, details map[string]string) error {
	return c.JSON(http.StatusUnprocessableEntity, APIResponse{
		Success: false,
		Error: &APIError{
			Code:    "VALIDATION_ERROR",
			Message: localize(c, "err.validation_error"),
			Details: details,
		},
	})
}

func NoContent(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}
