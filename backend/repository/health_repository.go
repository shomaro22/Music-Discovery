package repository

import (
	"context"
	"database/sql"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type healthCheckRepository struct {
	postgres *sql.DB
	redis    *redis.Client
}

func NewHealthCheckRepository(postgres *gorm.DB, redisClient *redis.Client) (*healthCheckRepository, error) {
	sqlDB, err := postgres.DB()
	if err != nil {
		return nil, err
	}
	return &healthCheckRepository{postgres: sqlDB, redis: redisClient}, nil
}

func (r *healthCheckRepository) PingPostgres(ctx context.Context) error {
	return r.postgres.PingContext(ctx)
}
func (r *healthCheckRepository) PingRedis(ctx context.Context) error { return r.redis.Ping(ctx).Err() }
func (r *healthCheckRepository) DatabaseStats() sql.DBStats          { return r.postgres.Stats() }
