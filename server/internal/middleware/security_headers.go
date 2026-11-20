package middleware

import "github.com/labstack/echo/v5"

// SecurityHeaders 给所有响应加上不会影响现有用法的基础安全头。
//
//   - Referrer-Policy: same-origin —— 文件预览、下载链接的 URL 里带着会话令牌
//     （?token=），没有这个头时，用户从应用跳到任何外部站点，浏览器都会把完整 URL
//     放进 Referer 发给对方。
//   - X-Content-Type-Options: nosniff —— 禁止浏览器把用户上传的内容猜成可执行类型。
//
// 这里刻意不加 X-Frame-Options / CSP：有人把 Bonds 嵌在自建面板的 iframe 里，
// 前端也依赖内联样式，这两个头会让现有部署直接不可用。
func SecurityHeaders() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			header := c.Response().Header()
			header.Set("Referrer-Policy", "same-origin")
			header.Set("X-Content-Type-Options", "nosniff")
			return next(c)
		}
	}
}
