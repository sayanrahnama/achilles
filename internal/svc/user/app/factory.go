package factory

import (
	"database/sql"

	"github.com/hailsayan/achilles/internal/svc/user/handler"
	"github.com/hailsayan/achilles/internal/svc/user/repository"
	"github.com/hailsayan/achilles/internal/svc/user/usecase"
)

type UserServiceFactory struct {
	db        *sql.DB
	redisRepo repository.RedisRepository
	
	userRepo  repository.UserRepository
	dataStore repository.DataStore
	
	userUseCase usecase.UserUseCase
	
	userHandler *handler.UserHandler
}

func NewUserServiceFactory(db *sql.DB, redisRepo repository.RedisRepository) *UserServiceFactory {
	factory := &UserServiceFactory{
		db:        db,
		redisRepo: redisRepo,
	}
	
	factory.initRepositories()
	factory.initUseCases()
	factory.initHandlers()
	
	return factory
}

func (f *UserServiceFactory) initRepositories() {
	f.userRepo = repository.NewUserRepository(f.db)
	f.dataStore = repository.NewDataStore(f.db)
}

func (f *UserServiceFactory) initUseCases() {
	f.userUseCase = usecase.NewUserUseCase(f.dataStore, f.redisRepo)
}

func (f *UserServiceFactory) initHandlers() {
	f.userHandler = handler.NewUserHandler(f.userUseCase)
}

func (f *UserServiceFactory) GetUserRepository() repository.UserRepository {
	return f.userRepo
}

func (f *UserServiceFactory) GetDataStore() repository.DataStore {
	return f.dataStore
}

func (f *UserServiceFactory) GetUserUseCase() usecase.UserUseCase {
	return f.userUseCase
}

func (f *UserServiceFactory) GetUserHandler() *handler.UserHandler {
	return f.userHandler
}

func (f *UserServiceFactory) Close() error {
	if f.db != nil {
		return f.db.Close()
	}
	return nil
}

func (f *UserServiceFactory) HealthCheck() error {
	if err := f.db.Ping(); err != nil {
		return err
	}
	return nil
}