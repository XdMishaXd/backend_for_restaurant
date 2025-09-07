package main

import (
	"context"
	"errors"
	"log/slog"
	"main_service/internal/config"
	booktable "main_service/internal/http-server/handlers/book_table"
	cancelbooking "main_service/internal/http-server/handlers/cancel_booking"
	getbookings "main_service/internal/http-server/handlers/get_bookings"
	bookingsrv "main_service/internal/http-server/handlers/middleware/booking"
	"main_service/internal/lib/jwt"
	"main_service/internal/lib/logger/sl"
	"main_service/internal/rabbitmq"
	"main_service/internal/storage/postgres"
	"main_service/internal/storage/redis"
	"net/http"
	"os"
	"time"

	ssogrpc "main_service/internal/clients/sso/grpc"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	cfg := config.MustLoad("./config/local.yaml")
	log := setupLogger(cfg.Env)

	log.Info("starting main service", slog.String("env", cfg.Env))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// * RabbitMQ
	rabbitMQClient, err := rabbitmq.New(cfg.RabbitMQ.URL, cfg.RabbitMQ.QueueName)
	if err != nil {
		log.Error("failed to init RabbitMQ", sl.Err(err))
		os.Exit(1)
	}
	defer rabbitMQClient.Close()

	// * Postgres
	postgresRepo, err := postgres.Connect(ctx, cfg)
	if err != nil {
		log.Error("failed to connect to postgres", sl.Err(err))
		os.Exit(1)
	}
	defer postgresRepo.Close()

	// * Redis
	redisRepo, err := redis.New(ctx, cfg.Redis.Host, cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		log.Error("failed to connect to redis", sl.Err(err))
		os.Exit(1)
	}
	defer redisRepo.Close()

	// * SSO grpc client
	ssoClient, err := ssogrpc.New(
		context.Background(),
		log,
		cfg.Clients.SSO.Address,
		cfg.Clients.SSO.Timeout,
		cfg.Clients.SSO.RetriesCount,
	)
	if err != nil {
		log.Error("failed to init grpc client", sl.Err(err))
		os.Exit(1)
	}

	bookingService := bookingsrv.NewBookingService(postgresRepo, redisRepo, rabbitMQClient)

	// * Routing
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(jwt.AuthMiddleware(cfg.AppSecret))

	// * Handlers
	r.Post("/book", booktable.New(log, ssoClient, bookingService))
	r.Post("/cancel", cancelbooking.New(log, ssoClient, bookingService, postgresRepo))
	r.Get("/bookings", getbookings.New(log, ssoClient, bookingService))

	srv := &http.Server{
		Addr:         cfg.HTTPServer.Address,
		Handler:      r,
		ReadTimeout:  cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout,
	}

	log.Info("HTTP server starting", slog.String("addr", cfg.HTTPServer.Address))
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error("server failed", sl.Err(err))
		os.Exit(1)
	}
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = slog.New(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envDev:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envProd:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}

	return log
}
