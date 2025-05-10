package repository

import (
	"context"
	"database/sql"
	
	"github.com/hailsayan/achilles/internal/pkg/logger"
)

type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

type DataStore interface {
	Atomic(ctx context.Context, fn func(DataStore) error) error
	UserRepository() UserRepository
}

type dataStore struct {
	conn   *sql.DB
	db     DBTX
	cache  CacheRepository
	logger logger.Logger
}

// NewDataStore creates a new datastore with DB connection and optional Redis cache
func NewDataStore(db *sql.DB, cache CacheRepository, logger logger.Logger) DataStore {
	return &dataStore{
		conn:   db,
		db:     db,
		cache:  cache,
		logger: logger,
	}
}

// Atomic executes a function within a transaction
func (s *dataStore) Atomic(ctx context.Context, fn func(DataStore) error) error {
	tx, err := s.conn.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	
	// Create a new datastore with the transaction
	txStore := &dataStore{
		conn:   s.conn,
		db:     tx,
		cache:  s.cache, // Share the same cache
		logger: s.logger,
	}
	
	err = fn(txStore)
	if err != nil {
		if errRollback := tx.Rollback(); errRollback != nil {
			s.logger.Error("Failed to rollback transaction", "error", errRollback)
		}
		return err
	}
	
	return tx.Commit()
}

// UserRepository returns a repository for user operations
func (s *dataStore) UserRepository() UserRepository {
	return NewUserRepository(s.db, s.cache, s.logger)
}