package model

import "time"

// User model
type User struct {
	UserID       int       `json:"user_id"`
	Email        string    `json:"email"`
	FirstName    string    `json:"firstname"`
	LastName     string    `json:"lastname"`
	UserType     string    `json:"user_type"`
	CreationDate time.Time `json:"creation_date"`
}
