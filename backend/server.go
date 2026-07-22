package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

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

	rateLimitRepository := repository.NewRateLimitRepository(redisClient)
	rateLimitUsecase := usecase.NewRateLimitUsecase(rateLimitRepository, clock)
	rateLimit := appMiddleware.NewRateLimitMiddleware(rateLimitUsecase)
	authMiddleware := appMiddleware.NewAuthMiddleware(tokenService, userUsecase)

	e := router.NewRouter(logger, userController, rateLimit, authMiddleware)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	return e.Start(":" + port)
}
