package app

import (
	grpcapp "SSO/internal/app/grpc"
	"SSO/internal/services/auth"
	"SSO/internal/storage/postgres"

	"log/slog"
	"time"
)

type App struct {
	GRPCSrv *grpcapp.App
}

func New(
	log *slog.Logger,
	storage *postgres.PostgresRepo,
	grpcPort int,
	tokenTTL time.Duration,
) *App {
	authService := auth.New(log, storage, storage, storage, tokenTTL)

	grpcApp := grpcapp.New(log, authService, grpcPort)

	return &App{
		GRPCSrv: grpcApp,
	}
}
