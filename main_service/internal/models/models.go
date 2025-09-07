package models

import "time"

type ContextKey string

type Booking struct {
	UserID      int64
	TableID     int16
	BookingTime time.Time
}

type User struct {
	ID         int64
	Email      string
	First_name string
	Last_name  string
}

type BookingInfo struct {
	BookingTime time.Time `json:"booking_time"`
	TableID     int16     `json:"table_id"`
	Email       string    `json:"email"`
	FirstName   string    `json:"first_name"`
	LastName    string    `json:"last_name"`
}
