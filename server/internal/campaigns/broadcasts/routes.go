package broadcast

import (
	"github.com/labstack/echo/v5"

	"github.com/dugble/dugble/server/internal/security/authz"
)

type AccessMiddleware func(permission authz.Permission) echo.MiddlewareFunc

func RegisterRoutes(router *echo.Echo, handler *Handler, access AccessMiddleware) {
	broadcasts := router.Group("/broadcasts")
	broadcasts.POST("", handler.Create, access(authz.PermissionBroadcastsWrite))
	broadcasts.GET("", handler.List, access(authz.PermissionBroadcastsRead))
	broadcasts.GET("/:broadcast", handler.Get, access(authz.PermissionBroadcastsRead))
	broadcasts.PATCH("/:broadcast", handler.Update, access(authz.PermissionBroadcastsWrite))
	broadcasts.DELETE("/:broadcast", handler.Delete, access(authz.PermissionBroadcastsWrite))
	broadcasts.POST("/:broadcast/send", handler.Send, access(authz.PermissionBroadcastsSend))
	broadcasts.POST("/:broadcast/cancel", handler.Cancel, access(authz.PermissionBroadcastsSend))
	broadcasts.POST("/:broadcast/duplicate", handler.Duplicate, access(authz.PermissionBroadcastsWrite))
	broadcasts.POST("/:broadcast/preview", handler.Preview, access(authz.PermissionBroadcastsRead))
	broadcasts.GET("/:broadcast/recipients", handler.ListRecipients, access(authz.PermissionBroadcastsRead))
	broadcasts.GET("/:broadcast/exclusions", handler.GetExclusionSummary, access(authz.PermissionBroadcastsRead))
	broadcasts.GET("/:broadcast/analytics", handler.GetAnalytics, access(authz.PermissionBroadcastsRead))
}
