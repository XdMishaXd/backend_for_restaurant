package postgres

import (
	"context"
	"fmt"
	"main_service/internal/config"
	"main_service/internal/models"
	"main_service/internal/storage"
	"time"

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

func (r *PostgresRepo) SaveBooking(ctx context.Context, booking models.Booking) error {
	const op = "storage.postgres.SaveBooking"

	var id int64
	err := r.pool.QueryRow(
		ctx,
		`INSERT INTO bookings (user_id, table_id, booking_time) VALUES ($1, $2, $3) RETURNING id;`,
		booking.UserID,
		booking.TableID,
		booking.BookingTime,
	).Scan(&id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// GetBookings возвращает либо все брони, либо только активные в зависимости от переменной mode
func (r *PostgresRepo) GetBookings(ctx context.Context, mode string) ([]models.BookingInfo, error) {
	const op = "storage.postgres.GetBookings"

	rows, err := r.pool.Query(
		ctx,
		`SELECT b.booking_time, b.table_id, u.email, u.first_name, u.last_name
		FROM bookings b
		JOIN users u ON u.id = b.user_id
		WHERE ($1 = 'all' OR (b.is_active = TRUE AND $1 = 'active'))
		ORDER BY b.booking_time`,
		mode,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var bookings []models.BookingInfo
	for rows.Next() {
		var b models.BookingInfo
		if err := rows.Scan(&b.BookingTime, &b.TableID, &b.Email, &b.FirstName, &b.LastName); err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		bookings = append(bookings, b)
	}

	return bookings, nil
}

func (r *PostgresRepo) DeleteBooking(ctx context.Context, tableId int16, bookingTime time.Time) error {
	const op = "storage.postgres.DeleteBooking"

	cmdTag, err := r.pool.Exec(
		ctx,
		`UPDATE bookings 
		SET is_active = FALSE 
		WHERE table_id = $1 AND booking_time = $2 AND is_active = $3`,
		tableId,
		bookingTime,
		true,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("%s: %w", op, storage.ErrBookingNotFound)
	}

	return nil
}

func (r *PostgresRepo) IsBookingOwner(ctx context.Context, tableID int16, bookingTime time.Time, userID int64) (bool, error) {
	const op = "storage.postgres.IsBookingOwner"

	var exists bool
	err := r.pool.QueryRow(
		ctx,
		`SELECT EXISTS(
			SELECT 1 
			FROM bookings 
			WHERE table_id = $1 AND booking_time = $2 AND user_id = $3
		)`,
		tableID,
		bookingTime,
		userID,
	).Scan(&exists)

	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	return exists, nil
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
