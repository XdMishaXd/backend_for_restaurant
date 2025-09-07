package bookingsrv

import (
	"context"
	"time"

	"main_service/internal/models"
	"main_service/internal/storage/redis"
)

type Postgres interface {
	SaveBooking(ctx context.Context, booking models.Booking) error
	DeleteBooking(ctx context.Context, tableId int16, bookingTime time.Time) error
	IsBookingOwner(ctx context.Context, tableID int16, bookingTime time.Time, userID int64) (bool, error)
	GetBookings(ctx context.Context, mode string) ([]models.BookingInfo, error)
}

type Redis interface {
	SaveBooking(ctx context.Context, booking redis.Booking) error
	DeleteBooking(ctx context.Context, tableID int16, bookingTime time.Time) error
}

type RabbitMQ interface {
	SendNotification(ctx context.Context, booking models.Booking) error
}

type BookingService struct {
	postgres Postgres
	redis    Redis
	rabbitmq RabbitMQ
}

func NewBookingService(pg Postgres, r Redis, mq RabbitMQ) *BookingService {
	return &BookingService{
		postgres: pg,
		redis:    r,
		rabbitmq: mq,
	}
}

func (s *BookingService) BookTable(ctx context.Context, booking models.Booking) error {
	err := s.redis.SaveBooking(
		ctx,
		redis.Booking{
			TableID: int64(booking.TableID),
			Time:    booking.BookingTime,
		},
	)
	if err != nil {
		return err
	}

	if err := s.postgres.SaveBooking(ctx, booking); err != nil {
		return err
	}

	return s.rabbitmq.SendNotification(ctx, booking)
}

func (s *BookingService) CancelBooking(ctx context.Context, tableID int16, bookingTime time.Time) error {
	if err := s.postgres.DeleteBooking(ctx, tableID, bookingTime); err != nil {
		return err
	}

	if err := s.redis.DeleteBooking(ctx, tableID, bookingTime); err != nil {
		return err
	}

	return s.rabbitmq.SendNotification(
		ctx,
		models.Booking{
			UserID:      -1, // ! Если UserID == -1, то это отмена брони, в остальных случаях это новая бронь.
			TableID:     tableID,
			BookingTime: bookingTime,
		},
	)
}

func (s *BookingService) GetBookings(ctx context.Context, mode string) ([]models.BookingInfo, error) {
	return s.postgres.GetBookings(ctx, mode)
}
