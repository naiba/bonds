package handlers

import (
	"errors"

	"github.com/labstack/echo/v4"
	"github.com/naiba/bonds/internal/services"
	"github.com/naiba/bonds/pkg/response"
)

type TelegramWebhookHandler struct {
	webhookService *services.TelegramWebhookService
}

func NewTelegramWebhookHandler(webhookService *services.TelegramWebhookService) *TelegramWebhookHandler {
	return &TelegramWebhookHandler{webhookService: webhookService}
}

// HandleWebhook godoc
//
//	@Summary		Handle Telegram webhook
//	@Description	Process incoming Telegram webhook updates
//	@Tags			telegram
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	response.APIResponse
//	@Failure		400	{object}	response.APIResponse
//	@Failure		404	{object}	response.APIResponse
//	@Failure		500	{object}	response.APIResponse
//	@Router			/telegram/webhook [post]
func (h *TelegramWebhookHandler) HandleWebhook(c echo.Context) error {
	var update services.TelegramUpdate
	if err := c.Bind(&update); err != nil {
		return response.BadRequest(c, "err.invalid_request_body", nil)
	}
	if err := h.webhookService.HandleUpdate(update); err != nil {
		if errors.Is(err, services.ErrInvalidTelegramUpdate) {
			return response.BadRequest(c, "err.invalid_telegram_update", nil)
		}
		if errors.Is(err, services.ErrTelegramChannelNotFound) {
			return response.NotFound(c, "err.telegram_channel_not_found")
		}
		return response.InternalError(c, "err.failed_to_handle_telegram_webhook")
	}
	return response.OK(c, map[string]string{"status": "ok"})
}
