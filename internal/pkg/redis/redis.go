package redis

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
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

func NewRedis(opt *RedisOptions) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
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

	status, err := rdb.Ping(ctx).Result()
	if err != nil {
		log.Printf("Failed to connect to Redis: %v", err)
		return nil, err
	}

	log.Printf("Connected to Redis: status=%s, addr=%s, maxActiveConn=%d, maxIdleConn=%d",
		status, opt.Addr, opt.MaxActiveConn, opt.MaxIdleConn,
	)

	return rdb, nil
}
