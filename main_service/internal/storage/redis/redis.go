package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"main_service/internal/storage"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	redisScript = `
		-- KEYS[1] = userKey
		-- KEYS[2] = tableKey
		-- ARGV[1] = value
		-- ARGV[2] = ttl (ms)

		-- Проверяем, есть ли уже бронь у пользователя
		if redis.call("EXISTS", KEYS[1]) == 1 then
			return redis.error_reply("USER_ALREADY_BOOKED")
		end

		-- Проверяем, занят ли стол
		if redis.call("EXISTS", KEYS[2]) == 1 then
			return redis.error_reply("TABLE_ALREADY_BOOKED")
		end

		-- Добавляем бронь
		redis.call("SET", KEYS[1], ARGV[1], "PX", ARGV[2])
		redis.call("SET", KEYS[2], ARGV[1], "PX", ARGV[2])
		return "OK"
	`
)

type RedisRepo struct {
	client *redis.Client
}

type Booking struct {
	TableID int64     `json:"table_id"`
	UserID  int64     `json:"user_id"`
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

// SaveBooking сохраняет бронь, ключом является время:столик:userID
func (r *RedisRepo) SaveBooking(ctx context.Context, booking Booking) error {
	const op = "storage.redis.SaveBooking"

	userKey := fmt.Sprintf("booking:user:%d", booking.UserID)
	tableKey := fmt.Sprintf("booking:%s:table%d:uid:%d",
		booking.Time.Format("2006-01-02"),
		booking.TableID,
		booking.UserID,
	)

	// TTL до конца дня
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

	data, err := json.Marshal(booking)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	ttlMs := fmt.Sprintf("%d", ttl.Milliseconds())
	_, err = r.client.Eval(ctx, redisScript,
		[]string{userKey, tableKey},
		string(data), ttlMs,
	).Result()

	if err != nil {
		if strings.Contains(err.Error(), "USER_ALREADY_BOOKED") {
			return fmt.Errorf("%s: %w", op, storage.ErrUserAlreadyBooked)
		}
		if strings.Contains(err.Error(), "TABLE_ALREADY_BOOKED") {
			return fmt.Errorf("%s: %w", op, storage.ErrTableIsBooked)
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// DeleteBooking удаляет бронь по времени и id стола
func (r *RedisRepo) DeleteBooking(ctx context.Context, booking Booking) error {
	userKey := fmt.Sprintf("booking:user:%d", booking.UserID)
	tableKey := fmt.Sprintf("booking:%s:table%d:uid:%d",
		booking.Time.Format("2006-01-02"),
		booking.TableID,
		booking.UserID,
	)

	return r.client.Del(ctx, userKey, tableKey).Err()
}

// Close закрывает соединение с базой данных.
func (r *RedisRepo) Close() {
	r.client.Close()
}
