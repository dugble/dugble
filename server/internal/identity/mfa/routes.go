package mfa

import (
	"time"

	"github.com/labstack/echo/v5"

	httpmiddleware "github.com/dugble/dugble/server/internal/runtime/middleware"
	"github.com/dugble/dugble/server/internal/security/authn"
)

func RegisterRoutes(router *echo.Echo, handler *Handler, authMiddleware, csrfMiddleware echo.MiddlewareFunc) {
	group := router.Group("/auth/mfa")
	group.Use(authMiddleware)
	group.Use(csrfMiddleware)

	recentPassword := httpmiddleware.RequireRecentAuthentication(httpmiddleware.StepUpConfig{Assurance: authn.AssuranceLevelOne, MaxAge: 15 * time.Minute})
	recentMFA := httpmiddleware.RequireRecentAuthentication(httpmiddleware.StepUpConfig{Assurance: authn.AssuranceLevelTwo, MaxAge: 15 * time.Minute})
	group.GET("", handler.Status)
	group.POST("/totp/enroll", handler.Enroll, recentPassword)
	group.POST("/totp/confirm", handler.Confirm, recentPassword)
	group.POST("/verify", handler.Verify)
	group.POST("/recovery", handler.Recover)
	group.DELETE("", handler.Disable, recentMFA)
}
