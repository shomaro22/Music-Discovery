package router

import (
	"log/slog"
	"net/http"
	"os"
	"strings"

	"music-discovery/backend/controller"
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
	rateLimit appMiddleware.IRateLimitMiddleware,
	authMiddleware appMiddleware.IAuthMiddleware,
) *echo.Echo {
	e := echo.New()

	e.HideBanner = true
	e.HTTPErrorHandler = appMiddleware.NewHTTPErrorHandler(logger)

	// 実効順序:
	// Request ID
	// Recover
	// Logger
	// Security Header
	// CORS
	e.Use(echoMiddleware.RequestID())
	e.Use(echoMiddleware.Recover())
	e.Use(echoMiddleware.Logger())
	e.Use(echoMiddleware.Secure())
	e.Use(echoMiddleware.CORSWithConfig(
		echoMiddleware.CORSConfig{
			AllowOrigins: loadAllowedOrigins(),
			AllowMethods: []string{
				http.MethodGet,
				http.MethodPost,
				http.MethodPut,
				http.MethodPatch,
				http.MethodDelete,
				http.MethodOptions,
			},
			AllowHeaders: []string{
				echo.HeaderOrigin,
				echo.HeaderContentType,
				echo.HeaderAccept,
				echo.HeaderAuthorization,
				echo.HeaderXRequestID,
				"X-CSRF-Token",
			},
			AllowCredentials: true,
		},
	))

	auth := e.Group("/api/auth")

	auth.POST(
		"/signup",
		userController.SignUp,

		// Route Middlewareの実効順序:
		// Body Limit
		// Rate Limit
		// Content-Type
		// Controller
		echoMiddleware.BodyLimit(signupBodyLimit),
		rateLimit.SignUp,
		appMiddleware.RequireJSON,
	)

	auth.POST(
		"/login",
		userController.Login,
		echoMiddleware.BodyLimit(loginBodyLimit),
		rateLimit.Login,
		appMiddleware.RequireJSON,
	)

	auth.POST(
		"/logout",
		userController.Logout,
		echoMiddleware.BodyLimit(refreshBodyLimit),
		appMiddleware.CSRF,
	)

	auth.POST(
		"/refresh",
		userController.Refresh,
		echoMiddleware.BodyLimit(refreshBodyLimit),
		rateLimit.Refresh,
		appMiddleware.CSRF,
	)

	e.GET("/api/users/me", userController.CurrentUser, authMiddleware.Authenticate)
	e.DELETE("/api/users/me", userController.Withdraw, echoMiddleware.BodyLimit(withdrawBodyLimit), rateLimit.Withdraw, authMiddleware.Authenticate, appMiddleware.CSRF, appMiddleware.RequireJSON)

	return e
}

func loadAllowedOrigins() []string {
	rawOrigins := strings.TrimSpace(
		os.Getenv("CORS_ALLOW_ORIGIN"),
	)
	if rawOrigins == "" {
		return []string{}
	}

	values := strings.Split(rawOrigins, ",")
	origins := make([]string, 0, len(values))

	for _, value := range values {
		origin := strings.TrimSpace(value)
		if origin == "" {
			continue
		}

		origins = append(origins, origin)
	}

	return origins
}
