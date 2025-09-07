package postgres

import (
	"SSO/internal/config"
	"SSO/internal/domain/models"
	"SSO/internal/storage"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepo struct {
	pool *pgxpool.Pool
}

// Connect создает подключение к базе данных и возвращает репозиторий.
func Connect(ctx context.Context, cfg *config.Config) (*PostgresRepo, error) {
	const op = "storage.postgres.Connect"

	dsn := dsn(cfg)

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to parse config: %w", op, err)
	}

	poolConfig.MaxConns = 10
	poolConfig.MinConns = 2
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = time.Minute * 30

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to create pool: %w", op, err)
	}

	return &PostgresRepo{pool: pool}, nil
}

func (r *PostgresRepo) SaveUser(ctx context.Context, first_name, last_name string, email string, passHash []byte) (int64, error) {
	const op = "storage.postgres.SaveUser"

	var id int64
	err := r.pool.QueryRow(
		ctx,
		`INSERT INTO users (first_name, last_name, email, pass_hash) VALUES ($1, $2, $3, $4) RETURNING id;`,
		first_name,
		last_name,
		email,
		passHash,
	).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == pgerrcode.UniqueViolation {
				return -1, fmt.Errorf("%s: %w", op, storage.ErrUserExists)
			}
		}

		return -1, fmt.Errorf("%s: %w", op, err)
	}

	return id, nil
}

func (r *PostgresRepo) User(ctx context.Context, email string) (models.User, error) {
	const op = "storage.postgres.User"

	var usr models.User

	err := r.pool.QueryRow(
		ctx,
		`SELECT id, email, pass_hash FROM users WHERE email = $1`,
		email,
	).Scan(&usr.ID, &usr.Email, &usr.PassHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, fmt.Errorf("%s: %w", op, storage.ErrUserNotFound)
		}

		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}

	return usr, nil
}

func (r *PostgresRepo) GetUser(ctx context.Context, userID int64) (models.User, error) {
	const op = "storage.postgres.Username"

	var usr models.User

	err := r.pool.QueryRow(
		ctx,
		`SELECT first_name, last_name, email FROM users WHERE id = $1`,
		userID,
	).Scan(&usr.First_name, &usr.Last_name, &usr.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, fmt.Errorf("%s: %w", op, storage.ErrUserNotFound)
		}

		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}

	return usr, nil
}

func (r *PostgresRepo) IsAdmin(ctx context.Context, userID int64) (bool, error) {
	const op = "storage.postgres.IsAdmin"

	var isAdmin bool

	err := r.pool.QueryRow(
		ctx,
		`SELECT is_admin FROM users WHERE id = $1`,
		userID,
	).Scan(&isAdmin)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, fmt.Errorf("%s: %w", op, storage.ErrUserNotFound)
		}

		return false, fmt.Errorf("%s: %w", op, err)
	}

	return isAdmin, nil
}

func (r *PostgresRepo) App(ctx context.Context, appID int) (models.App, error) {
	const op = "storage.postgres.App"

	var app models.App

	err := r.pool.QueryRow(
		ctx,
		`SELECT id, name, secret FROM apps WHERE id = $1`,
		appID,
	).Scan(&app.ID, &app.Name, &app.Secret)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.App{}, fmt.Errorf("%s: %w", op, storage.ErrAppNotFound)
		}

		return models.App{}, fmt.Errorf("%s: %w", op, err)
	}

	return app, nil
}

// Close закрывает соединение с базой данных.
func (r *PostgresRepo) Close() {
	r.pool.Close()
}

// dsn формирует конфигурацию базы данных.
func dsn(cfg *config.Config) string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s database=%s sslmode=%s",
		cfg.Postgres.Host,
		cfg.Postgres.Port,
		cfg.Postgres.User,
		cfg.Postgres.Password,
		cfg.Postgres.DBName,
		cfg.Postgres.SSLMode,
	)
}
