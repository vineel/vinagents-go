package service

import (
	"context"
	"errors"
	"time"

	"github.com/vineel/vinagents-go/internal/middleware"
	"github.com/vineel/vinagents-go/internal/repository"
	"github.com/vineel/vinagents-go/pkg/jwt"
	"golang.org/x/crypto/bcrypt"
)

// AuthResponse represents the response from auth operations
type AuthResponse struct {
	User         *UserResponse `json:"user"`
	AccessToken  string        `json:"accessToken"`
	RefreshToken string        `json:"refreshToken"`
}

// UserResponse is the sanitized user (no password)
type UserResponse struct {
	UserID    string     `json:"userId"`
	Email     string     `json:"email"`
	FirstName *string    `json:"firstName"`
	LastName  *string    `json:"lastName"`
	IsActive  bool       `json:"isActive"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

type AuthService struct {
	userRepo         *repository.UserRepository
	refreshTokenRepo *repository.RefreshTokenRepository
	jwtManager       *jwt.Manager
}

func NewAuthService(
	userRepo *repository.UserRepository,
	refreshTokenRepo *repository.RefreshTokenRepository,
	jwtManager *jwt.Manager,
) *AuthService {
	return &AuthService{
		userRepo:         userRepo,
		refreshTokenRepo: refreshTokenRepo,
		jwtManager:       jwtManager,
	}
}

func (s *AuthService) Register(ctx context.Context, email, password string, firstName, lastName *string) (*AuthResponse, error) {
	// Check if user already exists
	exists, err := s.userRepo.Exists(ctx, email)
	if err != nil {
		return nil, middleware.NewInternalError("Failed to check user existence", err)
	}
	if exists {
		return nil, middleware.NewConflictError("User with this email already exists")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return nil, middleware.NewInternalError("Failed to hash password", err)
	}

	// Create user
	user, err := s.userRepo.Create(ctx, repository.CreateUserInput{
		Email:     email,
		Password:  string(hashedPassword),
		FirstName: firstName,
		LastName:  lastName,
	})
	if err != nil {
		return nil, middleware.NewInternalError("Failed to create user", err)
	}

	// Generate tokens
	accessToken, refreshToken, err := s.generateTokens(ctx, user.UserID, user.Email)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		User:         sanitizeUser(user),
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*AuthResponse, error) {
	// Find user
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, middleware.NewUnauthorizedError("Invalid credentials")
		}
		return nil, middleware.NewInternalError("Failed to find user", err)
	}

	// Check if active
	if !user.IsActive {
		return nil, middleware.NewUnauthorizedError("Account is disabled")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, middleware.NewUnauthorizedError("Invalid credentials")
	}

	// Generate tokens
	accessToken, refreshToken, err := s.generateTokens(ctx, user.UserID, user.Email)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		User:         sanitizeUser(user),
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*AuthResponse, error) {
	// Find stored token
	storedToken, err := s.refreshTokenRepo.FindByToken(ctx, refreshToken)
	if err != nil {
		if errors.Is(err, repository.ErrRefreshTokenNotFound) {
			return nil, middleware.NewUnauthorizedError("Refresh token not found")
		}
		return nil, middleware.NewInternalError("Failed to find refresh token", err)
	}

	// Check expiration
	if storedToken.ExpiresAt.Before(time.Now()) {
		_ = s.refreshTokenRepo.Delete(ctx, refreshToken)
		return nil, middleware.NewUnauthorizedError("Refresh token expired")
	}

	// Find user
	user, err := s.userRepo.FindByID(ctx, storedToken.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, middleware.NewUnauthorizedError("User not found")
		}
		return nil, middleware.NewInternalError("Failed to find user", err)
	}

	if !user.IsActive {
		return nil, middleware.NewUnauthorizedError("Account is disabled")
	}

	// Delete old refresh token
	_ = s.refreshTokenRepo.Delete(ctx, refreshToken)

	// Generate new tokens
	newAccessToken, newRefreshToken, err := s.generateTokens(ctx, user.UserID, user.Email)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		User:         sanitizeUser(user),
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	// Just delete the token - don't error if it doesn't exist
	_ = s.refreshTokenRepo.Delete(ctx, refreshToken)
	return nil
}

func (s *AuthService) generateTokens(ctx context.Context, userID, email string) (string, string, error) {
	// Generate access token
	accessToken, err := s.jwtManager.GenerateAccessToken(userID, email)
	if err != nil {
		return "", "", middleware.NewInternalError("Failed to generate access token", err)
	}

	// Generate refresh token
	refreshToken, err := s.jwtManager.GenerateRefreshToken()
	if err != nil {
		return "", "", middleware.NewInternalError("Failed to generate refresh token", err)
	}

	// Store refresh token
	expiresAt := time.Now().Add(s.jwtManager.RefreshExpiresIn())
	_, err = s.refreshTokenRepo.Create(ctx, repository.CreateRefreshTokenInput{
		Token:     refreshToken,
		UserID:    userID,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return "", "", middleware.NewInternalError("Failed to store refresh token", err)
	}

	return accessToken, refreshToken, nil
}

func sanitizeUser(user *repository.User) *UserResponse {
	return &UserResponse{
		UserID:    user.UserID,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		IsActive:  user.IsActive,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}
