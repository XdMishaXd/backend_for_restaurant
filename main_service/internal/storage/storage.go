package storage

import "errors"

var (
	ErrTableIsBooked     = errors.New("table is already booked")
	ErrBookingNotFound   = errors.New("booking is not found")
	ErrTableIsEmpty      = errors.New("bookings table is empty")
	ErrPastDate          = errors.New("cannot create booking for a past date")
	ErrUserAlreadyBooked = errors.New("user has already booked a table")
)
