package middleware

import (
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

const maxCorrelationIDLength = 128

// RequestCorrelation establishes one safe correlation identifier and exposes
// it on both the request and response for downstream logging and clients.
func RequestCorrelation(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		correlationID := validCorrelationID(c.Request().Header.Get(echo.HeaderXCorrelationID))
		if correlationID == "" {
			correlationID = requestID(c)
		}
		if correlationID == "" {
			correlationID = uuid.NewString()
		}
		c.Request().Header.Set(echo.HeaderXCorrelationID, correlationID)
		c.Response().Header().Set(echo.HeaderXCorrelationID, correlationID)
		return next(c)
	}
}

func validCorrelationID(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > maxCorrelationIDLength {
		return ""
	}
	for _, character := range value {
		if character < 0x21 || character > 0x7e {
			return ""
		}
	}
	return value
}

func requestID(c *echo.Context) string {
	if c == nil {
		return ""
	}
	if value := strings.TrimSpace(c.Request().Header.Get(echo.HeaderXRequestID)); value != "" {
		return value
	}
	return strings.TrimSpace(c.Response().Header().Get(echo.HeaderXRequestID))
}
