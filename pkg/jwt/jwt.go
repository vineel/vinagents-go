package jwt

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token has expired")
)

// Claims represents the JWT claims
type Claims struct {
	UserID string `json:"userId"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// Manager handles JWT operations
type Manager struct {
	accessSecret        []byte
	accessExpiresIn     time.Duration
	refreshSecret       []byte
	refreshExpiresIn    time.Duration
}

// NewManager creates a new JWT manager
func NewManager(accessSecret string, accessExpiresIn time.Duration, refreshSecret string, refreshExpiresIn time.Duration) *Manager {
	return &Manager{
		accessSecret:     []byte(accessSecret),
		accessExpiresIn:  accessExpiresIn,
		refreshSecret:    []byte(refreshSecret),
		refreshExpiresIn: refreshExpiresIn,
	}
}

// GenerateAccessToken creates a new access token for a user
func (m *Manager) GenerateAccessToken(userID, email string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(m.accessExpiresIn)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.accessSecret)
}

// ValidateAccessToken validates an access token and returns the claims
func (m *Manager) ValidateAccessToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return m.accessSecret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// GenerateRefreshToken creates a random refresh token string
func (m *Manager) GenerateRefreshToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// RefreshExpiresIn returns the refresh token expiration duration
func (m *Manager) RefreshExpiresIn() time.Duration {
	return m.refreshExpiresIn
}

// AccessExpiresIn returns the access token expiration duration
func (m *Manager) AccessExpiresIn() time.Duration {
	return m.accessExpiresIn
}
