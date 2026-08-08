package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

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
	appEnvironment := envString("APP_ENV", "local")
	serviceName := envString("SERVICE_NAME", "music-discovery-api")
	appVersion := envString("APP_VERSION", "development")
	logger := newLogger(appEnvironment, serviceName, appVersion)
	logger.Info("application starting")

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
		Market:       envString("SPOTIFY_MARKET", "JP"),
	})
	if err != nil {
		return fmt.Errorf("spotify client: %w", err)
	}
	spotifyCandidateRepository := repository.NewSpotifyCandidateRepository(postgres)
	if os.Getenv("SEED_CURATED_CATALOG") == "true" {
		curatedCatalogUsecase := usecase.NewCuratedCatalogImportUsecase(
			spotifyClient,
			repository.NewCuratedCatalogRepository(postgres),
		)
		result, importErr := curatedCatalogUsecase.Import(ctx, usecase.DefaultCuratedCatalogSeeds())
		if importErr != nil {
			return fmt.Errorf("seed curated catalog: %w", importErr)
		}
		logger.Info(
			"curated catalog seed completed",
			"defined", result.Defined,
			"existing", result.Existing,
			"imported", result.Imported,
			"not_found", result.NotFound,
		)
	}
	spotifyCandidateImportUsecase := usecase.NewSpotifyCandidateImportUsecase(
		spotifyClient,
		spotifyCandidateRepository,
		usecase.SpotifyCandidateImportConfig{
			SearchLimit:        envPositiveInt("SPOTIFY_CANDIDATE_SEARCH_LIMIT", 10),
			MaxQueries:         envPositiveInt("SPOTIFY_CANDIDATE_MAX_QUERIES", 3),
			MinimumPopularity:  envNonNegativeInt("SPOTIFY_CANDIDATE_MIN_POPULARITY", 35),
			MaxTracksPerArtist: envPositiveInt("SPOTIFY_CANDIDATE_MAX_TRACKS_PER_ARTIST", 2),
			MaxPagesPerQuery:   envPositiveInt("SPOTIFY_CANDIDATE_MAX_PAGES_PER_QUERY", 3),
		},
	)

	recommendationRepository := repository.NewRecommendationRepository(postgres)
	recommendationLockRepository := repository.NewRecommendationLockRepository(redisClient)
	recommendationConfig := loadRecommendationConfig()
	if err := usecase.ValidateRecommendationScoreWeights(recommendationConfig.ScoreWeights); err != nil {
		return fmt.Errorf("recommendation weight config: %w", err)
	}
	if err := usecase.ValidateGenreRelations(); err != nil {
		return fmt.Errorf("recommendation genre relation config: %w", err)
	}
	recommendationCalculator := usecase.NewRecommendationCalculator(recommendationConfig.ScoreWeights)
	recommendationConditionBuilder := usecase.NewRecommendationConditionBuilder(questionRepository, questionAnswerRepository, userProfileRepository, recommendationRepository, authTransactionManager)
	recommendationUsecase := usecase.NewRecommendationUsecase(
		userRepository,
		recommendationRepository,
		recommendationLockRepository,
		authTransactionManager,
		recommendationCalculator,
		recommendationConditionBuilder,
		spotifyCandidateImportUsecase,
		clock,
		recommendationConfig,
	)
	recommendationController := controller.NewRecommendationController(recommendationUsecase)
	dailyDiscoveryConfig := usecase.DailyDiscoveryV1Config(recommendationConfig)
	dailyDiscoveryUsecase := usecase.NewDailyDiscoveryUsecase(userRepository, userProfileRepository, recommendationRepository, recommendationLockRepository, authTransactionManager, recommendationCalculator, clock, recommendationConditionBuilder, dailyDiscoveryConfig)
	dailyDiscoveryController := controller.NewDailyDiscoveryController(dailyDiscoveryUsecase)
	recommendationHistoryUsecase := usecase.NewRecommendationHistoryUsecase(userRepository, userProfileRepository, recommendationRepository)
	recommendationHistoryController := controller.NewRecommendationHistoryController(recommendationHistoryUsecase)
	feedbackRepository := repository.NewFeedbackRepository(postgres)
	feedbackUsecase := usecase.NewFeedbackUsecase(userRepository, feedbackRepository, authTransactionManager, clock, feedbackConfig)
	feedbackController := controller.NewFeedbackController(feedbackUsecase)

	adminRepository := repository.NewAdminRepository(postgres)
	adminManagementUsecase := usecase.NewAdminManagementUsecase(adminRepository, clock)
	systemLogUsecase := usecase.NewSystemLogUsecase(adminRepository)
	logRetentionUsecase := usecase.NewLogRetentionUsecase(repository.NewLogRetentionRepository(postgres), usecase.LogRetentionConfig{SystemDays: envPositiveInt("SYSTEM_LOG_RETENTION_DAYS", 90), SensitiveDays: envPositiveInt("ERROR_SECURITY_LOG_RETENTION_DAYS", 180), AuditDays: envPositiveInt("AUDIT_LOG_RETENTION_DAYS", 365)})
	go runLogRetention(ctx, logger, logRetentionUsecase)
	artistAdminController := controller.NewArtistAdminController(adminManagementUsecase)
	songAdminController := controller.NewSongAdminController(adminManagementUsecase)
	tagAdminController := controller.NewTagAdminController(adminManagementUsecase)
	userAdminController := controller.NewUserAdminController(adminManagementUsecase)
	roleAdminController := controller.NewRoleAdminController(adminManagementUsecase)
	logAdminController := controller.NewLogAdminController(adminManagementUsecase)
	metricsUsecase := usecase.NewMetricsUsecase(time.Duration(envPositiveInt("SLOW_REQUEST_THRESHOLD_MS", 1000))*time.Millisecond, appVersion)
	healthRepository, err := repository.NewHealthCheckRepository(postgres, redisClient)
	if err != nil {
		return fmt.Errorf("health repository: %w", err)
	}
	metricsUsecase.SetDatabaseStatsProvider(func() usecase.DatabaseMetrics {
		stats := healthRepository.DatabaseStats()
		return usecase.DatabaseMetrics{OpenConnections: stats.OpenConnections, InUseConnections: stats.InUse, IdleConnections: stats.Idle, WaitCount: stats.WaitCount}
	})
	healthUsecase := usecase.NewHealthUsecase(healthRepository, appVersion, time.Duration(envPositiveInt("HEALTH_CHECK_TIMEOUT_MS", 2000))*time.Millisecond, logger)
	alertUsecase := usecase.NewAlertUsecase(metricsUsecase, envFloat("ALERT_ERROR_RATE_THRESHOLD", 0.05), envFloat("ALERT_P95_THRESHOLD_MS", 1000))
	alertNotifier := usecase.NewWebhookAlertNotifier(os.Getenv("ALERT_WEBHOOK_URL"), time.Duration(envPositiveInt("ALERT_NOTIFICATION_TIMEOUT_MS", 5000))*time.Millisecond)
	alertStateRepository := repository.NewAlertStateRepository(postgres)
	alertMonitor := usecase.NewAlertMonitor(alertUsecase, healthUsecase, alertNotifier, alertStateRepository)
	go runAlertEvaluation(ctx, logger, alertMonitor, time.Duration(envPositiveInt("ALERT_EVALUATION_INTERVAL_SECONDS", 60))*time.Second)
	monitoringController := controller.NewMonitoringController(metricsUsecase, appVersion, controller.WithHealthUsecase(healthUsecase), controller.WithAlertUsecase(alertUsecase), controller.WithAlertMonitor(alertMonitor))

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
		recommendationHistoryController,
		feedbackController,
		spotifyAdminController,
		artistAdminController,
		songAdminController,
		tagAdminController,
		userAdminController,
		roleAdminController,
		logAdminController,
		monitoringController,
		metricsUsecase,
		rateLimit,
		authMiddleware,
	)
	e.Pre(appMiddleware.NewSystemLogMiddleware(systemLogUsecase, appMiddleware.SystemLogConfig{Environment: appEnvironment, Service: serviceName, Version: appVersion, Logger: logger, Metrics: metricsUsecase, SaveTimeout: 2 * time.Second, SlowThreshold: time.Duration(envPositiveInt("SLOW_REQUEST_THRESHOLD_MS", 1000)) * time.Millisecond}))
	port := os.Getenv("PORT")
	if port == "" {
		port = os.Getenv("APP_PORT")
	}
	if port == "" {
		port = "8080"
	}

	return e.Start("0.0.0.0:" + port)
}

func runAlertEvaluation(ctx context.Context, logger *slog.Logger, monitor usecase.IAlertMonitor, interval time.Duration) {
	if interval <= 0 {
		interval = time.Minute
	}
	evaluate := func() {
		evalCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := monitor.EvaluateAndPublish(evalCtx); err != nil {
			logger.Error("alert evaluation or notification failed", "error", err)
		}
	}

	// 起動直後にも一度評価し、Alert一覧が初回Tickerまで空になるのを防ぐ。
	evaluate()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			evaluate()
		}
	}
}

func runLogRetention(ctx context.Context, logger *slog.Logger, retention usecase.ILogRetentionUsecase) {
	run := func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		result, err := retention.Cleanup(cleanupCtx)
		if err != nil {
			logger.Error("log retention cleanup failed", "error", err)
			return
		}
		logger.Info("log retention cleanup completed", "system_deleted", result.SystemDeleted, "audit_deleted", result.AuditDeleted)
	}
	run()
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			run()
		}
	}
}

func newLogger(environment, service, version string) *slog.Logger {
	level := new(slog.LevelVar)
	switch strings.ToLower(strings.TrimSpace(os.Getenv("LOG_LEVEL"))) {
	case "debug":
		level.Set(slog.LevelDebug)
	case "warn":
		level.Set(slog.LevelWarn)
	case "error":
		level.Set(slog.LevelError)
	default:
		level.Set(slog.LevelInfo)
	}
	options := &slog.HandlerOptions{Level: level}
	var handler slog.Handler
	if environment == "local" || environment == "development" {
		handler = slog.NewTextHandler(os.Stdout, options)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, options)
	}
	return slog.New(handler).With("environment", environment, "service", service, "version", version)
}

func envPositiveInt(name string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(name))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func envNonNegativeInt(name string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(name))
	if err != nil || value < 0 {
		return fallback
	}
	return value
}

func envString(name, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return value
}

func loadRecommendationConfig() usecase.RecommendationConfig {
	return usecase.RecommendationConfig{
		CandidateLimit:      envPositiveInt("RECOMMENDATION_CANDIDATE_LIMIT", 100),
		DuplicateDays:       envPositiveInt("RECOMMENDATION_DUPLICATE_DAYS", 30),
		RecencyDays:         envPositiveInt("RECOMMENDATION_RECENCY_DAYS", 30),
		GenerationWindow:    envPositiveInt("RECOMMENDATION_GENERATION_WINDOW_SECONDS", 10),
		LockTTLSeconds:      envPositiveInt("RECOMMENDATION_LOCK_TTL_SECONDS", 120),
		AlgorithmVersion:    usecase.RecommendationAlgorithmVersion,
		WeightConfigVersion: usecase.RecommendationWeightConfigVersion,
		MinimumMatchScore:   usecase.RecommendationMinimumMatchScore,
		MinimumContactScore: usecase.RecommendationMinimumContactScore,
		ScoreWeights:        usecase.DefaultRecommendationScoreWeights(),
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
