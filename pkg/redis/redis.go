package redis

import (
	"context"
	"time"

	"github.com/hailsayan/achilles/pkg/logger"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type RedisOptions struct {
	Addr            string
	DialTimeout     int
	ReadTimeout     int
	WriteTimeout    int
	MinIdleConn     int
	MaxIdleConn     int
	MaxActiveConn   int
	MaxConnLifetime int
}

func New(opt *RedisOptions, log logger.Logger) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:            opt.Addr,
		DialTimeout:     time.Duration(opt.DialTimeout) * time.Second,
		ReadTimeout:     time.Duration(opt.ReadTimeout) * time.Second,
		WriteTimeout:    time.Duration(opt.WriteTimeout) * time.Second,
		MinIdleConns:    opt.MinIdleConn,
		MaxIdleConns:    opt.MaxIdleConn,
		MaxActiveConns:  opt.MaxActiveConn,
		ConnMaxLifetime: time.Duration(opt.MaxConnLifetime) * time.Minute,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	status, err := client.Ping(ctx).Result()
	if err != nil {
		log.Error("Failed to connect to Redis", zap.Error(err))
		return nil, err
	}
	
	log.Info("Connected to Redis",
		zap.String("status", status),
		zap.String("addr", opt.Addr),
		zap.Int("maxActiveConn", opt.MaxActiveConn),
		zap.Int("maxIdleConn", opt.MaxIdleConn),
	)

	return client, nil
}

func Close(client *redis.Client, log logger.Logger) {
	if err := client.Close(); err != nil {
		log.Error("Error closing Redis connection", zap.Error(err))
		return
	}
	log.Info("Redis connection closed")
}