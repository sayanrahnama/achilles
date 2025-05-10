package app

import (
	"database/sql"

	"github.com/hailsayan/achilles/internal/pkg/logger"
	"github.com/hailsayan/achilles/internal/svc/user/handler"
	"github.com/hailsayan/achilles/internal/svc/user/repository"
	"github.com/hailsayan/achilles/internal/svc/user/usecase"
	"github.com/redis/go-redis/v9"
)

type Factory struct {
	db     *sql.DB
	redis  *redis.ClusterClient
	logger logger.Logger
}

func NewFactory(db *sql.DB, redis *redis.ClusterClient, logger logger.Logger) *Factory {
	return &Factory{
		db:     db,
		redis:  redis,
		logger: logger,
	}
}

func (f *Factory) NewDataStore() repository.DataStore {
	var cache repository.CacheRepository

	if f.redis != nil {
		redisRepo := repository.NewRedisClusterRepository(f.redis)
		cache = repository.NewCacheRepository(redisRepo)
	}

	return repository.NewDataStore(f.db, cache, f.logger)
}

func (f *Factory) NewUserUseCase() usecase.UserUseCase {
	ds := f.NewDataStore()
	return usecase.NewUserUseCase(ds.UserRepository(), f.logger)
}

func (f *Factory) NewUserHandler() *handler.UserHandler {
	userUseCase := f.NewUserUseCase()
	return handler.NewUserHandler(userUseCase, f.logger)
}
