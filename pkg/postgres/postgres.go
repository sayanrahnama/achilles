package postgres

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/hailsayan/achilles/pkg/logger"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

type PostgresOptions struct {
	Host            string
	DbName          string
	Username        string
	Password        string
	Sslmode         string
	Port            int
	MaxIdleConn     int
	MaxOpenConn     int
	MaxConnLifetime int
}

func New(opts PostgresOptions, log logger.Logger) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		opts.Host, opts.Port, opts.Username, opts.Password, opts.DbName, opts.Sslmode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Error("Failed to open PostgreSQL connection", zap.Error(err))
		return nil, err
	}

	// Connection pool configuration
	db.SetMaxOpenConns(opts.MaxOpenConn)
	db.SetMaxIdleConns(opts.MaxIdleConn)
	db.SetConnMaxLifetime(time.Duration(opts.MaxConnLifetime) * time.Second)

	if err := db.Ping(); err != nil {
		log.Error("Failed to ping PostgreSQL", zap.Error(err))
		return nil, err
	}

	log.Info("Connected to PostgreSQL",
		zap.String("host", opts.Host),
		zap.Int("port", opts.Port),
		zap.String("user", opts.Username),
		zap.String("dbname", opts.DbName),
		zap.Int("maxOpenConn", opts.MaxOpenConn),
		zap.Int("maxIdleConn", opts.MaxIdleConn),
	)

	return db, nil
}

func Close(db *sql.DB, log logger.Logger) {
	if err := db.Close(); err != nil {
		log.Error("Error closing PostgreSQL connection", zap.Error(err))
		return
	}
	log.Info("PostgreSQL connection closed")
}