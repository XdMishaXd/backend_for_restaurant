package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"main_service/internal/storage"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisRepo struct {
	client *redis.Client
}

type Booking struct {
	TableID int64     `json:"table_id"`
	Time    time.Time `json:"booking_time"`
}

func New(ctx context.Context, address string, password string, db int) (*RedisRepo, error) {
	const op = "storage.redis.New"

	rdb := redis.NewClient(&redis.Options{
		Addr: address,
		// Password: password,
		DB: db,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &RedisRepo{client: rdb}, nil
}

// SaveBooking сохраняет бронь, ключом является время
func (r *RedisRepo) SaveBooking(ctx context.Context, booking Booking) error {
	const op = "storage.redis.SaveBooking"

	booked, err := r.isTableBooked(ctx, booking.Time, booking.TableID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	if booked {
		return storage.ErrTableIsBooked
	}

	date := booking.Time.Format("2006-01-02")
	key := fmt.Sprintf("booking:%s:table%d", date, booking.TableID)

	data, err := json.Marshal(booking)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	endOfBookingDay := time.Date(
		booking.Time.Year(),
		booking.Time.Month(),
		booking.Time.Day(),
		23, 59, 59, 0,
		booking.Time.Location(),
	)

	ttl := time.Until(endOfBookingDay)
	if ttl <= 0 {
		return storage.ErrPastDate
	}

	return r.client.Set(ctx, key, data, ttl).Err()
}

// IsTableBooked проверяет, можно ли бронировать
func (r *RedisRepo) isTableBooked(ctx context.Context, checkTime time.Time, tableID int64) (bool, error) {
	const op = "storage.redis.IsTableBooked"

	date := checkTime.Format("2006-01-02")
	key := fmt.Sprintf("booking:%s:table%d", date, tableID)

	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil // ключа нет — можно бронировать
	}
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	var booking Booking
	if err := json.Unmarshal([]byte(val), &booking); err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	startBlock := booking.Time.Add(-5 * time.Hour)
	endBlock := time.Date(
		booking.Time.Year(),
		booking.Time.Month(),
		booking.Time.Day(),
		23, 59, 59, 0,
		booking.Time.Location(),
	)

	if checkTime.After(startBlock) && checkTime.Before(endBlock.Add(time.Second)) {
		return true, nil
	}

	return false, nil
}

// DeleteBooking удаляет бронь по времени и id стола
func (r *RedisRepo) DeleteBooking(ctx context.Context, tableID int16, bookingTime time.Time) error {
	date := bookingTime.Format("2006-01-02")
	key := fmt.Sprintf("booking:%s:table%d", date, tableID)

	return r.client.Del(ctx, key).Err()
}

// Close закрывает соединение с базой данных.
func (r *RedisRepo) Close() {
	r.client.Close()
}
