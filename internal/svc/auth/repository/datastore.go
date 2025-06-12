package repository

import (
	"context"
	"database/sql"

	"github.com/redis/go-redis/v9"
)

type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

type DataStore interface {
	Atomic(ctx context.Context, fn func(DataStore) error) error
	AuthRepository() AuthRepository
	TokenRepository() TokenRepository
}

type dataStore struct {
	conn *sql.DB
	db   DBTX
	rdb  *redis.ClusterClient
}

func NewDataStore(db *sql.DB, rdb *redis.ClusterClient) DataStore {
	return &dataStore{
		conn: db,
		db:   db,
		rdb:  rdb,
	}
}

func (s *dataStore) Atomic(ctx context.Context, fn func(DataStore) error) error {
	tx, err := s.conn.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}

	err = fn(&dataStore{conn: s.conn, db: tx, rdb: s.rdb})
	if err != nil {
		if errRollback := tx.Rollback(); errRollback != nil {
			return err
		}
		return err
	}

	return tx.Commit()
}

func (s *dataStore) AuthRepository() AuthRepository {
	return NewAuthRepository(s.db)
}

func (s *dataStore) TokenRepository() TokenRepository {
	return NewTokenRepository(s.rdb)
}
