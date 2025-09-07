package emailsender

import (
	"fmt"
	"time"

	"gopkg.in/gomail.v2"
)

type Mailer struct {
	Host     string
	Port     int
	Username string
	Password string
}

func (m *Mailer) Send(to, subject, body string) error {
	msg := gomail.NewMessage()
	msg.SetHeader("From", m.Username)
	msg.SetHeader("To", to)

	msg.SetBody("text/plain", body)

	dialer := gomail.NewDialer(m.Host, m.Port, m.Username, m.Password)
	return dialer.DialAndSend(msg)
}

func (m *Mailer) CreateMessege(userID, tableID int, bookingTime time.Time) (string, string) {
	var subject, messageText string

	formattedTime := bookingTime.Format("02-01-2006 15:04:05")

	if userID == -1 {
		subject = "Отмена брони"

		messageText = fmt.Sprintf("Бронь отменена! Столик номер %d. Дата и время: %s", tableID, formattedTime)
	} else {
		subject = "Новая бронь"

		messageText = fmt.Sprintf("Новая бронь! Столик номер %d. Дата и время: %s", tableID, formattedTime)
	}

	return subject, messageText
}
