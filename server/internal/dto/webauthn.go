package dto

import "time"

type WebAuthnLoginBeginRequest struct {
	// 这里不加 email 格式规则：早期注册不校验邮箱格式，库里可能有不是标准邮箱的登录标识，
	// 对“按邮箱查找已有账号”的请求做格式校验会把这些老账号挡在外面。
	Email string `json:"email" validate:"required" example:"user@example.com"`
}

type WebAuthnCredentialResponse struct {
	ID        uint      `json:"id" example:"1"`
	Name      string    `json:"name" example:"YubiKey 5"`
	CreatedAt time.Time `json:"created_at" example:"2026-01-15T10:30:00Z"`
}
