package entity

import (
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	admin Role = "admin"
	user  Role = "user"
)

type User struct {
	Id        uuid.UUID  `json:"id"`
	Email     string     `json:"email"`
	Role      Role       `json:"role"`
	CreatedAt *time.Time `json:"createdAt"`
}
