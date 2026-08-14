package webhooks

import (
	"github.com/labstack/echo/v5"

	"github.com/dugble/dugble/server/internal/authz"
)

type TenantMiddleware func(permission authz.Permission) echo.MiddlewareFunc

func RegisterRoutes(
	router *echo.Echo,
	handler *Handler,
	authMiddleware echo.MiddlewareFunc,
	csrfMiddleware echo.MiddlewareFunc,
	tenantMiddleware TenantMiddleware,
) {
	endpoints := router.Group("/webhook-endpoints")
	endpoints.Use(authMiddleware, csrfMiddleware)
	endpoints.POST("", handler.CreateEndpoint, tenantMiddleware(authz.PermissionWebhooksWrite))
	endpoints.GET("", handler.ListEndpoints, tenantMiddleware(authz.PermissionWebhooksRead))
	endpoints.GET("/:endpoint_id", handler.GetEndpoint, tenantMiddleware(authz.PermissionWebhooksRead))
	endpoints.PATCH("/:endpoint_id", handler.UpdateEndpoint, tenantMiddleware(authz.PermissionWebhooksWrite))
	endpoints.DELETE("/:endpoint_id", handler.DeleteEndpoint, tenantMiddleware(authz.PermissionWebhooksWrite))
	endpoints.POST("/:endpoint_id/test", handler.TestEndpoint, tenantMiddleware(authz.PermissionWebhooksWrite))
	endpoints.POST("/:endpoint_id/rotate-secret", handler.RotateSecret, tenantMiddleware(authz.PermissionWebhooksWrite))

	events := router.Group("/webhook-events")
	events.Use(authMiddleware, csrfMiddleware)
	events.GET("", handler.ListEvents, tenantMiddleware(authz.PermissionWebhooksRead))
	events.GET("/:event_id", handler.GetEvent, tenantMiddleware(authz.PermissionWebhooksRead))

	deliveries := router.Group("/webhook-deliveries")
	deliveries.Use(authMiddleware, csrfMiddleware)
	deliveries.GET("/:delivery_id", handler.GetDelivery, tenantMiddleware(authz.PermissionWebhooksRead))
	deliveries.POST("/:delivery_id/retry", handler.RetryDelivery, tenantMiddleware(authz.PermissionWebhooksWrite))
}
