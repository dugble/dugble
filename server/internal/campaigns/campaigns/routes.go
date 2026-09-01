package smscampaign

import (
	"github.com/dugble/dugble/server/internal/security/authz"
	"github.com/labstack/echo/v5"
)

type AccessMiddleware func(authz.Permission) echo.MiddlewareFunc

func RegisterRoutes(router *echo.Echo, handler *Handler, access AccessMiddleware) {
	campaigns := router.Group("/campaigns")
	campaigns.POST("", handler.Create, access(authz.PermissionSMSSend))
	campaigns.GET("", handler.List, access(authz.PermissionSMSRead))
	router.POST("/sms/opt-outs", handler.RecordOptOut, access(authz.PermissionSMSSend))
	campaigns.GET("/:campaign", handler.Get, access(authz.PermissionSMSRead))
	campaigns.PATCH("/:campaign", handler.Update, access(authz.PermissionSMSSend))
	campaigns.DELETE("/:campaign", handler.Delete, access(authz.PermissionSMSSend))
	campaigns.POST("/:campaign/duplicate", handler.Duplicate, access(authz.PermissionSMSSend))
	campaigns.POST("/:campaign/preview", handler.Preview, access(authz.PermissionSMSRead))
	campaigns.POST("/:campaign/send", handler.Send, access(authz.PermissionSMSSend))
	campaigns.POST("/:campaign/cancel", handler.Cancel, access(authz.PermissionSMSSend))
	campaigns.GET("/:campaign/recipients", handler.ListRecipients, access(authz.PermissionSMSRead))
	campaigns.GET("/:campaign/cost-estimate", handler.GetCostEstimate, access(authz.PermissionSMSRead))
	campaigns.GET("/:campaign/exclusions", handler.GetExclusionSummary, access(authz.PermissionSMSRead))
	campaigns.GET("/:campaign/analytics", handler.GetAnalytics, access(authz.PermissionSMSRead))
}
