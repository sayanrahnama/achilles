package redis

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisClusterOptions struct {
	Addrs           []string
	Password        string
	DialTimeout     int
	ReadTimeout     int
	WriteTimeout    int
	MinIdleConns    int
	MaxIdleConns     int
	MaxActiveConns   int
	ConnMaxLifetime int
}

func NewCluster(opt *RedisClusterOptions) (*redis.ClusterClient, error) {
	cluster := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:           opt.Addrs,
		Password:        opt.Password,
		DialTimeout:     time.Duration(opt.DialTimeout) * time.Second,
		ReadTimeout:     time.Duration(opt.ReadTimeout) * time.Second,
		WriteTimeout:    time.Duration(opt.WriteTimeout) * time.Second,
		MinIdleConns:    opt.MinIdleConns,
		MaxIdleConns:    opt.MaxIdleConns,
		MaxActiveConns:  opt.MaxActiveConns,
		ConnMaxLifetime: time.Duration(opt.ConnMaxLifetime) * time.Minute,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	status, err := cluster.Ping(ctx).Result()
	if err != nil {
		log.Printf("Failed to connect to Redis Cluster: %v", err)
		return nil, err
	}
	
	log.Printf("Connected to Redis Cluster: status=%s, addrs=%s, maxActiveConn=%d, maxIdleConn=%d",
		status, strings.Join(opt.Addrs, ","), opt.MaxActiveConns, opt.MaxIdleConns,
	)

	return cluster, nil
}