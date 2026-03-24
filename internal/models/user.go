package models

type User struct {
	ID           uint64
	Email        string
	PasswordHash []byte
}
