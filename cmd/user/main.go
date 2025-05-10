package main

import (
	"database/sql"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/hailsayan/achilles/pkg/logger"
	"github.com/hailsayan/achilles/pkg/redis"
	"github.com/hailsayan/achilles/proto/user/userpb"
	"github.com/hailsayan/achilles/user/handler"
	"github.com/hailsayan/achilles/user/repository"
	"github.com/hailsayan/achilles/user/service"
	_ "github.com/lib/pq"
	"google.golang.org/grpc"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	log, err := logger.NewZapLogger(cfg.LogLevel)
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Sync()

	// Connect to PostgreSQL
	dbConn, err := sql.Open("postgres", cfg.PostgresURI)
	if err != nil {
		log.Error("Failed to connect to PostgreSQL", "error", err)
		os.Exit(1)
	}
	defer dbConn.Close()

	// Test the database connection
	if err = dbConn.Ping(); err != nil {
		log.Error("Failed to ping PostgreSQL", "error", err)
		os.Exit(1)
	}

	// Initialize Redis client
	redisClient, err := redis.NewRedisClient(cfg.RedisURI)
	if err != nil {
		log.Error("Failed to connect to Redis", "error", err)
		os.Exit(1)
	}
	defer redisClient.Close()

	// Initialize the repositories
	userRepo := repository.NewPGUserRepository(dbConn, redisClient, log)

	// Initialize the services
	userService := service.NewUserService(userRepo, log)

	// Initialize the handlers
	userHandler := handler.NewUserHandler(userService, log)

	// Create a new gRPC server
	grpcServer := grpc.NewServer()
	userpb.RegisterUserServiceServer(grpcServer, userHandler)

	// Start listening
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
	if err != nil {
		log.Error("Failed to listen", "error", err)
		os.Exit(1)
	}

	// Graceful shutdown
	go func() {
		log.Info("User service started", "port", cfg.GRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Error("Failed to serve", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch

	// Stop the server gracefully
	log.Info("Shutting down server...")
	grpcServer.GracefulStop()
	log.Info("Server stopped")
}
