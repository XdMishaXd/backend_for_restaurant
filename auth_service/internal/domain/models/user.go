package models

type User struct {
	ID         int64
	Email      string
	First_name string
	Last_name  string
	PassHash   []byte
}
