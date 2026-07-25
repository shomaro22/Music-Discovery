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
	if err := migrate.CreateAdminTables(postgres); err != nil {
		return fmt.Errorf("migrate admin tables: %w", err)
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
