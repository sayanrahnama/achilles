package postgres

import (
	"database/sql"
	"fmt"
	"time"

	"log"
	_ "github.com/lib/pq"
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

func New(opts PostgresOptions) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		opts.Host, opts.Port, opts.Username, opts.Password, opts.DbName, opts.Sslmode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to open PostgreSQL connection: %v", err)
		return nil, err
	}

	db.SetMaxOpenConns(opts.MaxOpenConn)
	db.SetMaxIdleConns(opts.MaxIdleConn)
	db.SetConnMaxLifetime(time.Duration(opts.MaxConnLifetime) * time.Second)

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping PostgreSQL: %v", err)
		return nil, err
	}

	return db, nil
}
