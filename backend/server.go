package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"music-discovery/backend/controller"
	"music-discovery/backend/db"
	appMiddleware "music-discovery/backend/middleware"
	"music-discovery/backend/migrate"
	"music-discovery/backend/repository"
	"music-discovery/backend/router"
	"music-discovery/backend/spotify"
	"music-discovery/backend/usecase"
	"music-discovery/backend/validator"
)

func start(ctx context.Context) error {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	postgres, err := db.NewPostgres()
	if err != nil {
		return fmt.Errorf("postgres: %w", err)
	}

	if err := migrate.Run(postgres, migrate.Config{
		AppEnvironment:         os.Getenv("APP_ENV"),
		SeedDevelopmentUsers:   os.Getenv("SEED_DEVELOPMENT_USERS") == "true",
		SeedDevelopmentCatalog: os.Getenv("SEED_DEVELOPMENT_CATALOG") == "true",
	}); err != nil {
		return fmt.Errorf("migration: %w", err)
	}

	redisClient, err := db.NewRedis(ctx)
	if err != nil {
		return fmt.Errorf("redis: %w", err)
	}
	defer redisClient.Close()

	userRepository := repository.NewUserRepository(postgres)
	refreshRepository := repository.NewRefreshTokenRepository(postgres)
	authTransactionManager := repository.NewAuthTransactionManager(postgres)

	tokenService, err := usecase.NewTokenServiceFromEnv()
	if err != nil {
		return fmt.Errorf("token service: %w", err)
	}

	userValidator := validator.NewUserValidator()
	clock := usecase.NewClock()
	userUsecase := usecase.NewUserUsecase(
		userRepository,
		userValidator,
		refreshRepository,
		authTransactionManager,
		usecase.NewPasswordHasher(),
		tokenService,
		clock,
	)
	userController := controller.NewUserController(userUsecase)
	questionRepository := repository.NewQuestionRepository(postgres)
	questionAnswerRepository := repository.NewQuestionAnswerRepository(postgres)
	userProfileRepository := repository.NewUserProfileRepository(postgres)
	feedbackConfig := loadFeedbackConfig()
	questionUsecase := usecase.NewQuestionUsecase(
		questionRepository, questionAnswerRepository, userProfileRepository,
		authTransactionManager, userRepository, clock,
		usecase.FeedbackWeightRange{Minimum: feedbackConfig.MinimumWeight, Maximum: feedbackConfig.MaximumWeight},
	)
	questionController := controller.NewQuestionController(questionUsecase)

	spotifyClient, err := spotify.NewClient(spotify.Config{
		ClientID:     os.Getenv("SPOTIFY_CLIENT_ID"),
		ClientSecret: os.Getenv("SPOTIFY_CLIENT_SECRET"),
	})
	if err != nil {
		return fmt.Errorf("spotify client: %w", err)
	}
	spotifyCandidateRepository := repository.NewSpotifyCandidateRepository(postgres)
	spotifyCandidateImportUsecase := usecase.NewSpotifyCandidateImportUsecase(
		spotifyClient,
		spotifyCandidateRepository,
		usecase.SpotifyCandidateImportConfig{
			SearchLimit: envPositiveInt("SPOTIFY_CANDIDATE_SEARCH_LIMIT", 10),
			MaxQueries:  envPositiveInt("SPOTIFY_CANDIDATE_MAX_QUERIES", 3),
		},
	)

	recommendationRepository := repository.NewRecommendationRepository(postgres)
	recommendationLockRepository := repository.NewRecommendationLockRepository(redisClient)
	recommendationConfig := loadRecommendationConfig()
	recommendationCalculator := usecase.NewRecommendationCalculator(recommendationConfig.ScoreWeights)
	recommendationUsecase := usecase.NewRecommendationUsecase(
		userRepository,
		questionRepository,
		questionAnswerRepository,
		userProfileRepository,
		recommendationRepository,
		recommendationLockRepository,
		authTransactionManager,
		recommendationCalculator,
		spotifyCandidateImportUsecase,
		clock,
		recommendationConfig,
	)
	recommendationController := controller.NewRecommendationController(recommendationUsecase)
	dailyDiscoveryUsecase := usecase.NewDailyDiscoveryUsecase(userRepository, questionRepository, questionAnswerRepository, userProfileRepository, recommendationRepository, authTransactionManager, recommendationCalculator, clock, recommendationConfig)
	dailyDiscoveryController := controller.NewDailyDiscoveryController(dailyDiscoveryUsecase)
	feedbackRepository := repository.NewFeedbackRepository(postgres)
	feedbackUsecase := usecase.NewFeedbackUsecase(userRepository, feedbackRepository, authTransactionManager, clock, feedbackConfig)
	feedbackController := controller.NewFeedbackController(feedbackUsecase)

	adminRepository := repository.NewAdminRepository(postgres)
	adminManagementUsecase := usecase.NewAdminManagementUsecase(adminRepository, clock)
	systemLogUsecase := usecase.NewSystemLogUsecase(adminRepository)
	artistAdminController := controller.NewArtistAdminController(adminManagementUsecase)
	songAdminController := controller.NewSongAdminController(adminManagementUsecase)
	tagAdminController := controller.NewTagAdminController(adminManagementUsecase)
	userAdminController := controller.NewUserAdminController(adminManagementUsecase)
	roleAdminController := controller.NewRoleAdminController(adminManagementUsecase)
	logAdminController := controller.NewLogAdminController(adminManagementUsecase)

	spotifyAdminUsecase := usecase.NewSpotifyAdminUsecase(spotifyClient)
	spotifyAdminController := controller.NewSpotifyAdminController(spotifyAdminUsecase)

	rateLimitRepository := repository.NewRateLimitRepository(redisClient)
	rateLimitUsecase := usecase.NewRateLimitUsecase(rateLimitRepository, clock)
	rateLimit := appMiddleware.NewRateLimitMiddleware(rateLimitUsecase)
	authMiddleware := appMiddleware.NewAuthMiddleware(tokenService, userUsecase)

	e := router.NewRouter(
		logger,
		userController,
		questionController,
		recommendationController,
		dailyDiscoveryController,
		feedbackController,
		spotifyAdminController,
		artistAdminController,
		songAdminController,
		tagAdminController,
		userAdminController,
		roleAdminController,
		logAdminController,
		rateLimit,
		authMiddleware,
	)
	e.Use(appMiddleware.NewSystemLogMiddleware(systemLogUsecase))
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	return e.Start(":" + port)
}

func envPositiveInt(name string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(name))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func loadRecommendationConfig() usecase.RecommendationConfig {
	return usecase.RecommendationConfig{
		CandidateLimit:    envPositiveInt("RECOMMENDATION_CANDIDATE_LIMIT", 100),
		DuplicateDays:     envPositiveInt("RECOMMENDATION_DUPLICATE_DAYS", 30),
		RecencyDays:       envPositiveInt("RECOMMENDATION_RECENCY_DAYS", 30),
		GenerationWindow:  envPositiveInt("RECOMMENDATION_GENERATION_WINDOW_SECONDS", 10),
		LockTTLSeconds:    envPositiveInt("RECOMMENDATION_LOCK_TTL_SECONDS", 120),
		AlgorithmVersion:  usecase.RecommendationAlgorithmVersion,
		MinimumMatchScore: usecase.RecommendationMinimumMatchScore,
		ScoreWeights:      usecase.RecommendationV1ScoreWeights(),
	}
}

func envFloat(name string, fallback float64) float64 {
	value, err := strconv.ParseFloat(os.Getenv(name), 64)
	if err != nil {
		return fallback
	}
	return value
}

func loadFeedbackConfig() usecase.FeedbackConfig {
	return usecase.FeedbackConfig{
		Like:          usecase.FeedbackDelta{Genre: envFloat("FEEDBACK_LIKE_GENRE_DELTA", 0.5), Artist: envFloat("FEEDBACK_LIKE_ARTIST_DELTA", 2), Song: envFloat("FEEDBACK_LIKE_SONG_DELTA", 3), Profile: envFloat("FEEDBACK_LIKE_PROFILE_DELTA", 0.5), Discovery: envFloat("FEEDBACK_LIKE_DISCOVERY_DELTA", 0.5)},
		Maybe:         usecase.FeedbackDelta{Genre: envFloat("FEEDBACK_MAYBE_GENRE_DELTA", 0.2), Artist: envFloat("FEEDBACK_MAYBE_ARTIST_DELTA", 0.5), Song: envFloat("FEEDBACK_MAYBE_SONG_DELTA", 0.5), Profile: envFloat("FEEDBACK_MAYBE_PROFILE_DELTA", 0.2), Discovery: envFloat("FEEDBACK_MAYBE_DISCOVERY_DELTA", 0.2)},
		Skip:          usecase.FeedbackDelta{Genre: envFloat("FEEDBACK_SKIP_GENRE_DELTA", -0.2), Artist: envFloat("FEEDBACK_SKIP_ARTIST_DELTA", -1), Song: envFloat("FEEDBACK_SKIP_SONG_DELTA", -3), Profile: envFloat("FEEDBACK_SKIP_PROFILE_DELTA", -0.2), Discovery: envFloat("FEEDBACK_SKIP_DISCOVERY_DELTA", -0.5)},
		MinimumWeight: envFloat("FEEDBACK_WEIGHT_MIN", -10), MaximumWeight: envFloat("FEEDBACK_WEIGHT_MAX", 10),
		MinimumDiscoveryAdjustment: envFloat("DISCOVERY_ADJUSTMENT_MIN", -3), MaximumDiscoveryAdjustment: envFloat("DISCOVERY_ADJUSTMENT_MAX", 3),
	}
}
