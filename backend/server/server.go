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
	userUsecase := usecase.NewUserUsecase(userRepository, usecase.NewPasswordHasher(), usecase.NewClock())
	userController := controller.NewUserController(userUsecase)
	rateLimit := appMiddleware.NewRateLimitMiddleware(redisClient)
	e := router.NewRouter(logger, userController, rateLimit)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return e.Start(":" + port)
}
