package router

import (
	"log/slog"
	"net/http"
	"os"
	"strings"

	"music-discovery/backend/controller"
	"music-discovery/backend/entity"
	"music-discovery/backend/middleware"
	appMiddleware "music-discovery/backend/middleware"

	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
)

const signupBodyLimit = "16K"
const loginBodyLimit = "8K"
const refreshBodyLimit = "1K"
const withdrawBodyLimit = "8K"
const questionAnswersBodyLimit = "32K"
const feedbackBodyLimit = "4K"

func NewRouter(
	logger *slog.Logger,
	userController controller.IUserController,
	questionController controller.IQuestionController,
	recommendationController controller.IRecommendationController,
	dailyDiscoveryController controller.IDailyDiscoveryController,
	recommendationHistoryController controller.IRecommendationHistoryController,
	feedbackController controller.IFeedbackController,
	spotifyAdminController controller.ISpotifyAdminController,
	artistAdminController controller.IArtistAdminController,
	songAdminController controller.ISongAdminController,
	tagAdminController controller.ITagAdminController,
	userAdminController controller.IUserAdminController,
	roleAdminController controller.IRoleAdminController,
	logAdminController controller.ILogAdminController,
	rateLimit middleware.IRateLimitMiddleware,
	authMiddleware middleware.IAuthMiddleware,
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
	csrf := appMiddleware.NewCSRFMiddleware(os.Getenv("CSRF_COOKIE_NAME"))
	requireJSON := appMiddleware.NewRequireJSONMiddleware()

	publicAuth := e.Group("/api/auth")
	publicAuth.POST("/signup", userController.SignUp, echoMiddleware.BodyLimit(signupBodyLimit), rateLimit.SignUp, requireJSON)
	publicAuth.POST("/login", userController.Login, echoMiddleware.BodyLimit(loginBodyLimit), rateLimit.Login, requireJSON)
	publicAuth.POST("/refresh", userController.Refresh, echoMiddleware.BodyLimit(refreshBodyLimit), rateLimit.Refresh, csrf)

	authenticated := e.Group("/api", authMiddleware.Authenticate)
	authenticated.GET("/users/me", userController.CurrentUser, allAuthenticated.Authorize)
	authenticated.PATCH("/users/me", userController.UpdateProfile, echoMiddleware.BodyLimit(signupBodyLimit), csrf, requireJSON, allAuthenticated.Authorize)
	authenticated.GET("/questions", questionController.ListQuestions, allAuthenticated.Authorize)
	authenticated.GET("/user-answers/latest", questionController.GetLatestAnswers, allAuthenticated.Authorize)
	authenticated.POST("/user-answers", questionController.SaveAnswers, echoMiddleware.BodyLimit(questionAnswersBodyLimit), rateLimit.QuestionAnswers, requireJSON, allAuthenticated.Authorize)
	authenticated.POST("/recommendations", recommendationController.Generate, rateLimit.Recommendations, allAuthenticated.Authorize)
	authenticated.GET("/recommendations/latest", recommendationController.GetLatest, allAuthenticated.Authorize)
	authenticated.GET("/recommendations/:recommendationId", recommendationController.GetByID, allAuthenticated.Authorize)
	authenticated.GET("/daily-discovery", dailyDiscoveryController.GetToday, rateLimit.DailyDiscovery, userOnly.Authorize)
	authenticated.GET("/daily-discoveries", dailyDiscoveryController.List, userOnly.Authorize)
	authenticated.GET("/daily-discoveries/:recommendationId", dailyDiscoveryController.GetByID, userOnly.Authorize)
	authenticated.GET("/recommendation-history", recommendationHistoryController.List, userOnly.Authorize)
	authenticated.GET("/recommendation-history/:recommendationId", recommendationHistoryController.GetByID, userOnly.Authorize)
	authenticated.PUT("/recommendation-details/:recommendationDetailId/feedback", feedbackController.Save, echoMiddleware.BodyLimit(feedbackBodyLimit), rateLimit.Feedback, requireJSON, userOnly.Authorize)
	authenticated.GET("/recommendation-details/:recommendationDetailId/feedback", feedbackController.Get, userOnly.Authorize)
	authenticated.POST("/auth/logout", userController.Logout, echoMiddleware.BodyLimit(refreshBodyLimit), csrf, allAuthenticated.Authorize)
	authenticated.DELETE("/users/me", userController.Withdraw, echoMiddleware.BodyLimit(withdrawBodyLimit), rateLimit.Withdraw, csrf, requireJSON, userOnly.Authorize)

	content := e.Group("/api/admin", authMiddleware.Authenticate, contentAdminOnly.Authorize)
	registerContentAdminRoutes(content, artistAdminController, songAdminController, tagAdminController, spotifyAdminController)

	system := e.Group("/api/system", authMiddleware.Authenticate, systemAdminOnly.Authorize)
	registerSystemAdminRoutes(system, userAdminController, roleAdminController, logAdminController)

	return e
}

func mustRoleMiddleware(logger *slog.Logger, roles ...entity.Role) appMiddleware.IRoleMiddleware {
	middleware, err := appMiddleware.NewRoleMiddleware(logger, roles...)
	if err != nil {
		panic(err)
	}
	return middleware
}

func registerContentAdminRoutes(
	group *echo.Group,
	artistController controller.IArtistAdminController,
	songController controller.ISongAdminController,
	tagController controller.ITagAdminController,
	spotifyController controller.ISpotifyAdminController,
) {
	group.GET("/artists", artistController.ListArtists)
	group.GET("/artists/:artistId", artistController.GetArtist)
	group.POST("/artists", artistController.CreateArtist, appMiddleware.RequireJSON)
	group.PATCH("/artists/:artistId", artistController.UpdateArtist, appMiddleware.RequireJSON)
	group.DELETE("/artists/:artistId", artistController.DeleteArtist)

	group.GET("/songs", songController.ListSongs)
	group.GET("/songs/:songId", songController.GetSong)
	group.POST("/songs", songController.CreateSong, appMiddleware.RequireJSON)
	group.PATCH("/songs/:songId", songController.UpdateSong, appMiddleware.RequireJSON)
	group.DELETE("/songs/:songId", songController.DeleteSong)

	group.GET("/tags", tagController.ListTags)
	group.GET("/tags/options", tagController.ListTagOptions)
	group.GET("/tags/:tagId", tagController.GetTag)
	group.POST("/tags", tagController.CreateTag, appMiddleware.RequireJSON)
	group.PATCH("/tags/:tagId", tagController.UpdateTag, appMiddleware.RequireJSON)
	group.DELETE("/tags/:tagId", tagController.DeleteTag)

	group.GET("/spotify/artists", spotifyController.SearchArtists)
	group.GET("/spotify/tracks", spotifyController.SearchTracks)
}

func registerSystemAdminRoutes(
	group *echo.Group,
	userController controller.IUserAdminController,
	roleController controller.IRoleAdminController,
	logController controller.ILogAdminController,
) {
	group.GET("/users", userController.ListUsers)
	group.GET("/users/:userId", userController.GetUser)
	group.PATCH("/users/:userId/status", userController.SetUserStatus, appMiddleware.RequireJSON)

	group.GET("/roles", roleController.ListRoles)
	group.PATCH("/users/:userId/role", roleController.ChangeRole, appMiddleware.RequireJSON)
	group.GET("/audit-logs", logController.ListAuditLogs)
	group.GET("/audit-logs/:auditLogId", logController.GetAuditLog)
	group.GET("/logs", logController.ListSystemLogs)
	group.GET("/logs/:logId", logController.GetSystemLog)
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
