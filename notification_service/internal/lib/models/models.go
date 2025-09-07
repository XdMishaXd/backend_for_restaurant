package emailmodel

import "time"

type EmailMessage struct {
	UserID      int
	TableID     int
	BookingTime time.Time
}
