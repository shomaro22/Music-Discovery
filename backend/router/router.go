package router

import (
	"log/slog"
	"net/http"
	"os"
	"strings"

	"music-discovery/backend/controller"
	"music-discovery/backend/entity"
	appMiddleware "music-discovery/backend/middleware"

	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
)

const signupBodyLimit = "16K"
const loginBodyLimit = "8K"
const refreshBodyLimit = "1K"
const withdrawBodyLimit = "8K"

func NewRouter(
	logger *slog.Logger,
	userController controller.IUserController,
	adminController controller.IAdminController,
	rateLimit appMiddleware.IRateLimitMiddleware,
	authMiddleware appMiddleware.IAuthMiddleware,
) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HTTPErrorHandler = appMiddleware.NewHTTPErrorHandler(logger)
	e.Use(echoMiddleware.RequestID(), echoMiddleware.Recover(), echoMiddleware.Logger(), echoMiddleware.Secure())
	e.Use(echoMiddleware.CORSWithConfig(echoMiddleware.CORSConfig{
		AllowOrigins:     loadAllowedOrigins(),
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization, echo.HeaderXRequestID, "X-CSRF-Token"},
		AllowCredentials: true,
	}))

	allAuthenticated := mustRoleMiddleware(logger, entity.RoleUser, entity.RoleContentAdmin, entity.RoleSystemAdmin)
	userOnly := mustRoleMiddleware(logger, entity.RoleUser)
	contentAdminOnly := mustRoleMiddleware(logger, entity.RoleContentAdmin)
	systemAdminOnly := mustRoleMiddleware(logger, entity.RoleSystemAdmin)

	publicAuth := e.Group("/api/auth")
	publicAuth.POST("/signup", userController.SignUp, echoMiddleware.BodyLimit(signupBodyLimit), rateLimit.SignUp, appMiddleware.RequireJSON)
	publicAuth.POST("/login", userController.Login, echoMiddleware.BodyLimit(loginBodyLimit), rateLimit.Login, appMiddleware.RequireJSON)
	publicAuth.POST("/refresh", userController.Refresh, echoMiddleware.BodyLimit(refreshBodyLimit), rateLimit.Refresh, appMiddleware.CSRF)

	authenticated := e.Group("/api", authMiddleware.Authenticate)
	authenticated.GET("/users/me", userController.CurrentUser, allAuthenticated.Authorize)
	authenticated.POST("/auth/logout", userController.Logout, echoMiddleware.BodyLimit(refreshBodyLimit), appMiddleware.CSRF, allAuthenticated.Authorize)
	authenticated.DELETE("/users/me", userController.Withdraw, echoMiddleware.BodyLimit(withdrawBodyLimit), rateLimit.Withdraw, appMiddleware.CSRF, appMiddleware.RequireJSON, userOnly.Authorize)

	content := e.Group("/api/admin", authMiddleware.Authenticate, contentAdminOnly.Authorize)
	registerContentAdminRoutes(content, adminController)

	system := e.Group("/api/system", authMiddleware.Authenticate, systemAdminOnly.Authorize)
	registerSystemAdminRoutes(system, adminController)

	return e
}

func mustRoleMiddleware(logger *slog.Logger, roles ...entity.Role) appMiddleware.IRoleMiddleware {
	middleware, err := appMiddleware.NewRoleMiddleware(logger, roles...)
	if err != nil {
		panic(err)
	}
	return middleware
}

func registerContentAdminRoutes(group *echo.Group, adminController controller.IAdminController) {
	resources := []struct {
		path    string
		paramID string
	}{
		{path: "artists", paramID: "artistId"},
		{path: "songs", paramID: "songId"},
		{path: "tags", paramID: "tagId"},
	}
	for _, resource := range resources {
		group.GET("/"+resource.path, adminController.ProtectedPlaceholder)
		group.POST("/"+resource.path, adminController.ProtectedPlaceholder)
		group.PATCH("/"+resource.path+"/:"+resource.paramID, adminController.ProtectedPlaceholder)
		group.DELETE("/"+resource.path+"/:"+resource.paramID, adminController.ProtectedPlaceholder)
	}
}

func registerSystemAdminRoutes(group *echo.Group, adminController controller.IAdminController) {
	group.GET("/users", adminController.ProtectedPlaceholder)
	group.PATCH("/users/:userId/status", adminController.ProtectedPlaceholder)
	group.PATCH("/users/:userId/role", adminController.ChangeRole, appMiddleware.RequireJSON)
	group.GET("/audit-logs", adminController.ProtectedPlaceholder)
	group.GET("/logs", adminController.ProtectedPlaceholder)
}

func loadAllowedOrigins() []string {
	rawOrigins := strings.TrimSpace(os.Getenv("CORS_ALLOW_ORIGIN"))
	if rawOrigins == "" {
		return []string{}
	}
	values := strings.Split(rawOrigins, ",")
	origins := make([]string, 0, len(values))
	for _, value := range values {
		if origin := strings.TrimSpace(value); origin != "" {
			origins = append(origins, origin)
		}
	}
	return origins
}
