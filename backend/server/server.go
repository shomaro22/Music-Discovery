package server

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"music-discovery/backend/controller"
	"music-discovery/backend/db"
	appMiddleware "music-discovery/backend/middleware"
	"music-discovery/backend/repository"
	"music-discovery/backend/router"
	"music-discovery/backend/usecase"
)

func Start(ctx context.Context) error {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cookieConfig, err := controller.LoadCookieConfigFromEnv()
	if err != nil {
		return fmt.Errorf("cookie config: %w", err)
	}
	rateLimitConfig, err := appMiddleware.LoadRateLimitConfigFromEnv()
	if err != nil {
		return fmt.Errorf("rate limit config: %w", err)
	}
	postgres, err := db.NewPostgres()
	if err != nil {
		return fmt.Errorf("postgres: %w", err)
	}
	redisClient, err := db.NewRedis(ctx)
	if err != nil {
		return fmt.Errorf("redis: %w", err)
	}
	defer redisClient.Close()
	userRepository := repository.NewUserRepository(postgres)
	refreshRepository := repository.NewRefreshTokenRepository(postgres)
	tokenService, err := usecase.NewTokenServiceFromEnv()
	if err != nil {
		return fmt.Errorf("token service: %w", err)
	}
	userUsecase := usecase.NewUserUsecase(userRepository, usecase.NewPasswordHasher(), usecase.NewClock(), refreshRepository, tokenService)
	userController := controller.NewUserController(userUsecase, cookieConfig)
	rateLimit := appMiddleware.NewRateLimitMiddleware(redisClient, logger, rateLimitConfig)
	csrf := appMiddleware.NewCSRFMiddleware(cookieConfig.CSRFName)
	e := router.NewRouter(logger, userController, rateLimit, csrf)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return e.Start(":" + port)
}
