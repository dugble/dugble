package team

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
	teams := router.Group("/teams")
	teams.Use(authMiddleware, csrfMiddleware)

	teams.GET("", handler.List)
	teams.POST("", handler.Create)
	teams.GET("/invitations/:token", handler.GetInvitation)
	teams.POST("/invitations/:token/accept", handler.AcceptInvitation)
	teams.POST("/invitations/:token/decline", handler.DeclineInvitation)

	teamRoutes := teams.Group("/:team_id")
	teamRoutes.GET("", handler.Get, tenantMiddleware(authz.PermissionTeamRead))
	teamRoutes.PATCH("", handler.Update, tenantMiddleware(authz.PermissionTeamUpdate))
	teamRoutes.DELETE("", handler.Delete, tenantMiddleware(authz.PermissionTeamDelete))

	teamRoutes.GET(
		"/members",
		handler.ListMembers,
		tenantMiddleware(authz.PermissionTeamMembersRead),
	)
	teamRoutes.POST(
		"/members/invite",
		handler.InviteMember,
		tenantMiddleware(authz.PermissionTeamMemberInvite),
	)
	teamRoutes.DELETE(
		"/members/leave",
		handler.Leave,
		tenantMiddleware(authz.PermissionTeamMemberLeave),
	)
	teamRoutes.DELETE(
		"/members/:user_id",
		handler.RemoveMember,
		tenantMiddleware(authz.PermissionTeamMemberRemove),
	)
	teamRoutes.PATCH(
		"/members/:user_id",
		handler.UpdateMemberRole,
		tenantMiddleware(authz.PermissionTeamMemberRole),
	)
}
