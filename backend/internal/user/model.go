package user

import "time"

type Role string

const (
	RoleEmployee Role = "employee"
	RoleManager  Role = "manager"
)

type User struct {
	ID           int64
	Name         string
	Email        string
	PasswordHash string
	Role         Role
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type CreateParams struct {
	Name         string
	Email        string
	PasswordHash string
	Role         Role
}

type RegisterInput struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type Response struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  Role   `json:"role"`
}
