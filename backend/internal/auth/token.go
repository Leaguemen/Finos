package auth

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	issuer              = "finos-api"
	minimumSecretLength = 32
)

var ErrInvalidToken = errors.New("invalid token")

type Claims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

type TokenManager struct {
	secret []byte
	ttl    time.Duration
}

func NewTokenManager(secret string, ttl time.Duration) (*TokenManager, error) {
	if len([]byte(secret)) < minimumSecretLength {
		return nil, fmt.Errorf("JWT secret must contain at least %d bytes", minimumSecretLength)
	}
	if ttl <= 0 {
		return nil, errors.New("JWT TTL must be greater than zero")
	}

	return &TokenManager{
		secret: []byte(secret),
		ttl:    ttl,
	}, nil
}

func (manager *TokenManager) Generate(
	userID int64,
	role string,
) (string, time.Time, error) {
	if userID <= 0 || role == "" {
		return "", time.Time{}, errors.New("user ID and role are required")
	}

	now := time.Now().UTC()
	expiresAt := now.Add(manager.ttl)
	claims := Claims{
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   strconv.FormatInt(userID, 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(manager.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign JWT: %w", err)
	}

	return signedToken, expiresAt, nil
}

func (manager *TokenManager) Parse(tokenString string) (Claims, error) {
	claims := Claims{}
	token, err := jwt.ParseWithClaims(
		tokenString,
		&claims,
		func(*jwt.Token) (any, error) {
			return manager.secret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(issuer),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
	)
	if err != nil || !token.Valid {
		return Claims{}, ErrInvalidToken
	}

	if _, err := strconv.ParseInt(claims.Subject, 10, 64); err != nil || claims.Role == "" {
		return Claims{}, ErrInvalidToken
	}

	return claims, nil
}
