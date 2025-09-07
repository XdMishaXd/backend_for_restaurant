package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"notification_service/internal/config"
	mailer "notification_service/internal/email_sender"
	sl "notification_service/internal/lib/logger"
	emailmodel "notification_service/internal/lib/models"
	"notification_service/internal/rabbitmq"
	"os"
	"os/signal"
	"syscall"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := config.MustLoad()
	log := setupLogger(cfg.Env)

	startServer(ctx, cfg, log)
}

func startServer(ctx context.Context, cfg *config.Config, log *slog.Logger) {
	log.Info("starting notification service", slog.String("env", cfg.Env))

	r, err := rabbitmq.New(cfg.RabbitMQURL)
	if err != nil {
		log.Error("failed to init rabbitmq", sl.Err(err))
		return
	}
	defer r.Close()

	m := &mailer.Mailer{
		Host:     cfg.Email.Host,
		Port:     cfg.Email.Port,
		Username: cfg.Email.Username,
		Password: cfg.Email.Password,
	}

	done := make(chan struct{})

	go func() {
		defer close(done)

		err := r.StartReading(ctx, cfg.QueueName, func(msg []byte) {
			var emailMsg emailmodel.EmailMessage
			if err := json.Unmarshal(msg, &emailMsg); err != nil {
				log.Error("failed to unmarshal message", sl.Err(err))
				return
			}

			subject, mesText := m.CreateMessege(emailMsg.UserID, emailMsg.TableID, emailMsg.BookingTime)

			err := m.Send(cfg.AdministratorEmail,
				subject,
				mesText,
			)
			if err != nil {
				log.Error("failed to send message", sl.Err(err))
				return
			}

			log.Info("message sent successfully")
		})
		if err != nil {
			log.Error("failed to start reading", sl.Err(err))
			return
		}
	}()

	log.Info("notification service successfully started")

	select {
	case <-ctx.Done():
		log.Info("shutting down consumer...")
	case <-done:
		log.Info("notification service finished the work")
	}

	log.Info("notification service gracefully stopped")
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
