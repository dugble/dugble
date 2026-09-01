package middleware

import (
	"strings"

	"github.com/labstack/echo/v5"
)

const DefaultCacheControl = "private, no-store"

// CacheControl prevents accidental caching of API and authenticated responses.
// Handlers remain free to replace the default with an explicit cache policy.
func CacheControl(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		headers := c.Response().Header()
		if strings.TrimSpace(headers.Get(echo.HeaderCacheControl)) == "" {
			headers.Set(echo.HeaderCacheControl, DefaultCacheControl)
		}
		// Vary prevents a shared cache from reusing a representation across
		// requests with different credentials, even if an upstream overrides
		// the default policy.
		headers.Add("Vary", echo.HeaderAuthorization)
		headers.Add("Vary", "Cookie")
		return next(c)
	}
}
