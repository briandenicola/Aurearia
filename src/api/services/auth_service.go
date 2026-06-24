package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
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
	ErrRegistrationClosed  = errors.New("registration closed")
)

// AuthService handles authentication business logic.
type AuthService struct {
	repo      *repository.AuthRepository
	jwtSecret string
	settings  *SettingsService
	security  *SecurityService
	oidc      *repository.OIDCRepository
}

type AuthResult struct {
	Token        string
	RefreshToken string
	User         models.User
}

// NewAuthService creates a new AuthService.
func NewAuthService(repo *repository.AuthRepository, jwtSecret string) *AuthService {
	return &AuthService{repo: repo, jwtSecret: jwtSecret}
}

func (s *AuthService) WithSettings(settings *SettingsService) *AuthService {
	s.settings = settings
	return s
}

func (s *AuthService) WithSecurity(security *SecurityService) *AuthService {
	s.security = security
	return s
}

func (s *AuthService) WithOIDC(oidc *repository.OIDCRepository) *AuthService {
	s.oidc = oidc
	return s
}

func (s *AuthService) CheckAccountAllowed(username string) error {
	if s.security == nil {
		return nil
	}
	return s.security.CheckAccountAllowed(username)
}

func (s *AuthService) RecordWebAuthnSuccess(user models.User, clientIP, userAgent string) {
	if s.security != nil {
		s.security.RecordWebAuthnSuccess(user, clientIP, userAgent)
	}
}

func (s *AuthService) RecordWebAuthnFailure(username, clientIP, userAgent, message string) {
	if s.security != nil {
		var userID *uint
		if user, err := s.repo.FindUserByUsername(username); err == nil {
			id := user.ID
			userID = &id
		}
		s.security.RecordEvent(models.SecurityEventWebAuthnLoginFailure, userID, username, clientIP, userAgent, message)
	}
}

// RegisterUser creates a new user. The first user becomes admin.
func (s *AuthService) RegisterUser(username, email, password string) (*models.User, error) {
	count := s.repo.CountUsers()
	if count > 0 && s.settings != nil {
		allowed, err := s.additionalLocalRegistrationAllowed()
		if err != nil {
			return nil, err
		}
		if !allowed {
			return nil, ErrRegistrationClosed
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, ErrHashingFailed
	}

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

func (s *AuthService) additionalLocalRegistrationAllowed() (bool, error) {
	mode := strings.ToLower(strings.TrimSpace(s.settings.GetSetting(SettingRegistrationMode)))
	if mode == "open" {
		return true, nil
	}
	if s.oidc == nil {
		return false, nil
	}
	providers, err := s.oidc.ListEnabledProviders()
	if err != nil {
		return false, err
	}
	return len(providers) == 0, nil
}

// AuthenticateUser verifies credentials and returns the user on success.
func (s *AuthService) AuthenticateUser(username, password string) (*models.User, error) {
	return s.AuthenticateUserWithRequest(username, password, "", "")
}

func (s *AuthService) AuthenticateUserWithRequest(username, password, clientIP, userAgent string) (*models.User, error) {
	if s.security != nil {
		if err := s.security.CheckAccountAllowed(username); err != nil {
			s.security.RecordPasswordFailure(username, clientIP, userAgent)
			return nil, ErrInvalidCredentials
		}
	}
	user, err := s.repo.FindUserByUsername(username)
	if err != nil {
		if s.security != nil {
			s.security.RecordPasswordFailure(username, clientIP, userAgent)
		}
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		if s.security != nil {
			s.security.RecordPasswordFailure(username, clientIP, userAgent)
		}
		return nil, ErrInvalidCredentials
	}

	if s.security != nil {
		s.security.RecordPasswordSuccess(*user, clientIP, userAgent)
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

func (s *AuthService) IssueTokens(user models.User) (AuthResult, error) {
	accessToken, err := s.GenerateAccessToken(user)
	if err != nil {
		return AuthResult{}, err
	}
	refreshToken, err := s.GenerateRefreshToken(user)
	if err != nil {
		return AuthResult{}, err
	}
	return AuthResult{Token: accessToken, RefreshToken: refreshToken, User: user}, nil
}

// RotateTokens validates the old refresh token, rotates it, and returns the
// user, new access token, and new refresh token.
func (s *AuthService) RotateTokens(oldPlainToken string) (*models.User, string, string, error) {
	hash := sha256.Sum256([]byte(oldPlainToken))
	tokenHash := hex.EncodeToString(hash[:])

	rt, err := s.repo.FindRefreshToken(tokenHash)
	if err != nil {
		if s.security != nil {
			s.security.RecordEvent(models.SecurityEventRefreshFailure, nil, "", "", "", "")
		}
		return nil, "", "", ErrInvalidRefreshToken
	}

	if time.Now().After(rt.ExpiresAt) {
		if s.security != nil {
			s.security.RecordEvent(models.SecurityEventRefreshFailure, &rt.UserID, "", "", "", "expired refresh token")
		}
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
		if repository.IsRecordNotFound(err) {
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
