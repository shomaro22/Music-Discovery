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
	"music-discovery/backend/usecase"
	"music-discovery/backend/validator"
)

func start(ctx context.Context) error {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	postgres, err := db.NewPostgres()
	if err != nil {
		return fmt.Errorf("postgres: %w", err)
	}

	if err := migrate.CreateUsers(postgres); err != nil {
		return fmt.Errorf("migrate users: %w", err)
	}
	if err := migrate.CreateRefreshTokens(postgres); err != nil {
		return fmt.Errorf("migrate refresh tokens: %w", err)
	}
	if err := migrate.CreateAdminTables(postgres); err != nil {
		return fmt.Errorf("migrate admin tables: %w", err)
	}
	if err := migrate.CreateQuestionTables(postgres); err != nil {
		return fmt.Errorf("migrate question tables: %w", err)
	}
	if err := migrate.CreateRecommendationTables(postgres); err != nil {
		return fmt.Errorf("migrate recommendation tables: %w", err)
	}
	if os.Getenv("SEED_DEVELOPMENT_USERS") == "true" {
		if err := migrate.SeedDevelopmentUsers(postgres); err != nil {
			return fmt.Errorf("seed development users: %w", err)
		}
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
	questionUsecase := usecase.NewQuestionUsecase(questionRepository, questionAnswerRepository, userProfileRepository, authTransactionManager, userRepository, clock)
	questionController := controller.NewQuestionController(questionUsecase)
	recommendationRepository := repository.NewRecommendationRepository(postgres)
	recommendationLockRepository := repository.NewRecommendationLockRepository(redisClient)
	recommendationConfig := loadRecommendationConfig()
	recommendationCalculator := usecase.NewRecommendationCalculator(recommendationConfig.ScoreWeights)
	recommendationUsecase := usecase.NewRecommendationUsecase(userRepository, questionRepository, questionAnswerRepository, userProfileRepository, recommendationRepository, recommendationLockRepository, authTransactionManager, recommendationCalculator, clock, recommendationConfig)
	recommendationController := controller.NewRecommendationController(recommendationUsecase)
	adminRepository := repository.NewAdminRepository(postgres)
	adminManagementUsecase := usecase.NewAdminManagementUsecase(adminRepository, clock)
	systemLogUsecase := usecase.NewSystemLogUsecase(adminRepository)
	artistAdminController := controller.NewArtistAdminController(adminManagementUsecase)
	songAdminController := controller.NewSongAdminController(adminManagementUsecase)
	tagAdminController := controller.NewTagAdminController(adminManagementUsecase)
	userAdminController := controller.NewUserAdminController(adminManagementUsecase)
	roleAdminController := controller.NewRoleAdminController(adminManagementUsecase)
	logAdminController := controller.NewLogAdminController(adminManagementUsecase)

	rateLimitRepository := repository.NewRateLimitRepository(redisClient)
	rateLimitUsecase := usecase.NewRateLimitUsecase(rateLimitRepository, clock)
	rateLimit := appMiddleware.NewRateLimitMiddleware(rateLimitUsecase)
	authMiddleware := appMiddleware.NewAuthMiddleware(tokenService, userUsecase)

	e := router.NewRouter(
		logger,
		userController,
		questionController,
		recommendationController,
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
	port := os.Getenv("PORT")
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
		GenerationWindow:  envPositiveInt("RECOMMENDATION_GENERATION_WINDOW_SECONDS", 10),
		LockTTLSeconds:    envPositiveInt("RECOMMENDATION_LOCK_TTL_SECONDS", 120),
		AlgorithmVersion:  "recommendation-v1",
		MinimumMatchScore: 2,
		ScoreWeights: usecase.RecommendationScoreWeights{
			YourMatch:       usecase.RecommendationTypeScoreWeights{Mood: 3, Element: 1.5, Atmosphere: 1.5, Genre: 3, Artist: 1, Song: 1, Discovery: .25},
			NewFavorite:     usecase.RecommendationTypeScoreWeights{Mood: 1.5, Element: 1, Atmosphere: 1, Genre: 2, Artist: 1, Song: 1, Discovery: 2},
			TrySomethingNew: usecase.RecommendationTypeScoreWeights{Mood: 1, Element: 1, Atmosphere: 1, Genre: .5, Artist: .5, Song: .5, Discovery: 4},
			ArtistRecency:   .15, SongRecency: .3,
		},
	}
}
