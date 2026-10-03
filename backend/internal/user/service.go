package user

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

const (
	minimumPasswordBytes = 8
	maximumPasswordBytes = 72
	maximumNameRunes     = 255
)

var (
	ErrNameRequired       = errors.New("name is required")
	ErrNameTooLong        = errors.New("name must not exceed 255 characters")
	ErrInvalidEmail       = errors.New("email is invalid")
	ErrPasswordTooShort   = errors.New("password must contain at least 8 bytes")
	ErrPasswordTooLong    = errors.New("password must not exceed 72 bytes")
	ErrNotFoundUser       = errors.New("User not found")
	ErrInvalidCredentials = errors.New("invalid email or password")
)

type UserService struct {
	repository Repository
}

func NewService(repository Repository) *UserService {
	return &UserService{repository: repository}
}

func (service *UserService) Register(
	ctx context.Context,
	input RegisterInput,
) (Response, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return Response{}, ErrNameRequired
	}
	if utf8.RuneCountInString(name) > maximumNameRunes {
		return Response{}, ErrNameTooLong
	}

	email := normalizeEmail(input.Email)
	parsedEmail, err := mail.ParseAddress(email)
	if err != nil || parsedEmail.Address != email {
		return Response{}, ErrInvalidEmail
	}

	passwordLength := len([]byte(input.Password))
	if passwordLength < minimumPasswordBytes {
		return Response{}, ErrPasswordTooShort
	}
	if passwordLength > maximumPasswordBytes {
		return Response{}, ErrPasswordTooLong
	}

	passwordHash, err := bcrypt.GenerateFromPassword(
		[]byte(input.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return Response{}, fmt.Errorf("hash password: %w", err)
	}

	createdUser, err := service.repository.Create(ctx, CreateParams{
		Name:         name,
		Email:        email,
		PasswordHash: string(passwordHash),
		Role:         RoleEmployee,
	})
	if err != nil {
		return Response{}, fmt.Errorf("register user: %w", err)
	}

	return Response{
		ID:    createdUser.ID,
		Name:  createdUser.Name,
		Email: createdUser.Email,
		Role:  createdUser.Role,
	}, nil
}

func (service *UserService) Login(
	ctx context.Context,
	input LoginInput,
) (Response, error) {
	email := normalizeEmail(input.Email)
	foundUser, err := service.repository.FindByEmail(ctx, email)
	if err != nil {
		return Response{}, ErrNotFoundUser
	}

	err = bcrypt.CompareHashAndPassword(
		[]byte(foundUser.PasswordHash),
		[]byte(input.Password),
	) 
	if err != nil {
		return Response{}, ErrInvalidCredentials
	}

	return Response{
		ID:    foundUser.ID,
		Name:  foundUser.Name,
		Email: foundUser.Email,
		Role:  foundUser.Role,
	}, nil
}
