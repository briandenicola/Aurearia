package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	AccessTokenDuration  = 15 * time.Minute
	RefreshTokenDuration = 30 * 24 * time.Hour
)

var (
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrUsernameExists      = errors.New("username already exists")
	ErrHashingFailed       = errors.New("failed to hash password")
	ErrTokenGeneration     = errors.New("failed to generate token")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrRefreshTokenExpired = errors.New("refresh token expired")
)

// AuthService handles authentication business logic.
type AuthService struct {
	repo      *repository.AuthRepository
	jwtSecret string
}

// NewAuthService creates a new AuthService.
func NewAuthService(repo *repository.AuthRepository, jwtSecret string) *AuthService {
	return &AuthService{repo: repo, jwtSecret: jwtSecret}
}

// RegisterUser creates a new user. The first user becomes admin.
func (s *AuthService) RegisterUser(username, email, password string) (*models.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, ErrHashingFailed
	}

	count := s.repo.CountUsers()
	role := models.RoleUser
	if count == 0 {
		role = models.RoleAdmin
	}

	user := models.User{
		Username:     username,
		Email:        email,
		PasswordHash: string(hash),
		Role:         role,
	}

	if err := s.repo.CreateUser(&user); err != nil {
		return nil, ErrUsernameExists
	}

	return &user, nil
}

// AuthenticateUser verifies credentials and returns the user on success.
func (s *AuthService) AuthenticateUser(username, password string) (*models.User, error) {
	user, err := s.repo.FindUserByUsername(username)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return user, nil
}

// GenerateAccessToken creates a JWT access token for the given user.
func (s *AuthService) GenerateAccessToken(user models.User) (string, error) {
	claims := jwt.MapClaims{
		"userId":   user.ID,
		"username": user.Username,
		"role":     string(user.Role),
		"exp":      time.Now().Add(AccessTokenDuration).Unix(),
		"iat":      time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

// GenerateRefreshToken creates a refresh token, stores its hash, and returns
// the plain token string.
func (s *AuthService) GenerateRefreshToken(user models.User) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", ErrTokenGeneration
	}
	plainToken := "rt_" + hex.EncodeToString(b)

	hash := sha256.Sum256([]byte(plainToken))
	tokenHash := hex.EncodeToString(hash[:])

	rt := models.RefreshToken{
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(RefreshTokenDuration),
	}
	if err := s.repo.CreateRefreshToken(&rt); err != nil {
		return "", err
	}

	return plainToken, nil
}

// RotateTokens validates the old refresh token, rotates it, and returns the
// user, new access token, and new refresh token.
func (s *AuthService) RotateTokens(oldPlainToken string) (*models.User, string, string, error) {
	hash := sha256.Sum256([]byte(oldPlainToken))
	tokenHash := hex.EncodeToString(hash[:])

	rt, err := s.repo.FindRefreshToken(tokenHash)
	if err != nil {
		return nil, "", "", ErrInvalidRefreshToken
	}

	if time.Now().After(rt.ExpiresAt) {
		return nil, "", "", ErrRefreshTokenExpired
	}

	user, err := s.repo.FindUserByID(rt.UserID)
	if err != nil {
		return nil, "", "", ErrInvalidCredentials
	}

	// Generate new refresh token
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, "", "", ErrTokenGeneration
	}
	plainToken := "rt_" + hex.EncodeToString(b)
	newHash := sha256.Sum256([]byte(plainToken))
	newRT := models.RefreshToken{
		UserID:    user.ID,
		TokenHash: hex.EncodeToString(newHash[:]),
		ExpiresAt: time.Now().Add(RefreshTokenDuration),
	}

	if err := s.repo.RotateRefreshToken(rt, &newRT); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", "", ErrInvalidRefreshToken
		}
		return nil, "", "", err
	}

	accessToken, err := s.GenerateAccessToken(*user)
	if err != nil {
		return nil, "", "", err
	}

	return user, accessToken, plainToken, nil
}
