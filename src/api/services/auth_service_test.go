package services

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const testJWTSecret = "test-secret-key-for-unit-tests"

func setupAuthTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:auth_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	err = db.AutoMigrate(&models.User{}, &models.RefreshToken{})
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func newTestAuthService(db *gorm.DB) *AuthService {
	repo := repository.NewAuthRepository(db)
	return NewAuthService(repo, testJWTSecret)
}

// --- Registration ---

func TestRegisterUser_Success(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newTestAuthService(db)

	user, err := svc.RegisterUser("testuser", "test@example.com", "password123")
	if err != nil {
		t.Fatalf("RegisterUser failed: %v", err)
	}
	if user.ID == 0 {
		t.Fatal("expected user ID to be set")
	}
	if user.Username != "testuser" {
		t.Errorf("expected username 'testuser', got %q", user.Username)
	}
	// First user should be admin
	if user.Role != models.RoleAdmin {
		t.Errorf("expected first user to be admin, got %q", user.Role)
	}
	// Password hash should be valid bcrypt
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("password123")); err != nil {
		t.Error("password hash does not match original password")
	}
}

func TestRegisterUser_SecondUserIsRegular(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newTestAuthService(db)

	// First user = admin
	_, err := svc.RegisterUser("admin", "admin@example.com", "password123")
	if err != nil {
		t.Fatalf("first RegisterUser failed: %v", err)
	}

	// Second user = regular user
	user2, err := svc.RegisterUser("regular", "regular@example.com", "password456")
	if err != nil {
		t.Fatalf("second RegisterUser failed: %v", err)
	}
	if user2.Role != models.RoleUser {
		t.Errorf("expected second user to be 'user', got %q", user2.Role)
	}
}

func TestRegisterUser_DuplicateUsername(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newTestAuthService(db)

	_, err := svc.RegisterUser("duplicate", "first@example.com", "password123")
	if err != nil {
		t.Fatalf("first registration failed: %v", err)
	}

	_, err = svc.RegisterUser("duplicate", "second@example.com", "password456")
	if err != ErrUsernameExists {
		t.Errorf("expected ErrUsernameExists, got %v", err)
	}
}

// --- Authentication ---

func TestAuthenticateUser_ValidCredentials(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newTestAuthService(db)

	_, err := svc.RegisterUser("authuser", "auth@example.com", "correctpassword")
	if err != nil {
		t.Fatalf("setup: RegisterUser failed: %v", err)
	}

	user, err := svc.AuthenticateUser("authuser", "correctpassword")
	if err != nil {
		t.Fatalf("AuthenticateUser failed: %v", err)
	}
	if user.Username != "authuser" {
		t.Errorf("expected username 'authuser', got %q", user.Username)
	}
}

func TestAuthenticateUser_InvalidPassword(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newTestAuthService(db)

	_, err := svc.RegisterUser("authuser", "auth@example.com", "correctpassword")
	if err != nil {
		t.Fatalf("setup: RegisterUser failed: %v", err)
	}

	_, err = svc.AuthenticateUser("authuser", "wrongpassword")
	if err != ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthenticateUser_NonExistentUser(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newTestAuthService(db)

	_, err := svc.AuthenticateUser("nobody", "password")
	if err != ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

// --- Token Generation ---

func TestGenerateAccessToken_Valid(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newTestAuthService(db)

	user := models.User{ID: 42, Username: "tokenuser", Role: models.RoleAdmin}

	tokenString, err := svc.GenerateAccessToken(user)
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}
	if tokenString == "" {
		t.Fatal("expected non-empty token")
	}

	// Parse and verify claims
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(testJWTSecret), nil
	})
	if err != nil {
		t.Fatalf("failed to parse token: %v", err)
	}
	if !token.Valid {
		t.Fatal("token is not valid")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatal("failed to cast claims")
	}
	if uint(claims["userId"].(float64)) != 42 {
		t.Errorf("expected userId 42, got %v", claims["userId"])
	}
	if claims["username"] != "tokenuser" {
		t.Errorf("expected username 'tokenuser', got %v", claims["username"])
	}
	if claims["role"] != "admin" {
		t.Errorf("expected role 'admin', got %v", claims["role"])
	}
}

func TestGenerateAccessToken_ExpiresInFuture(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newTestAuthService(db)

	user := models.User{ID: 1, Username: "expuser", Role: models.RoleUser}
	tokenString, err := svc.GenerateAccessToken(user)
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}

	token, _ := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(testJWTSecret), nil
	})
	claims := token.Claims.(jwt.MapClaims)
	exp := time.Unix(int64(claims["exp"].(float64)), 0)
	if !exp.After(time.Now()) {
		t.Error("expected token expiry to be in the future")
	}
}

func TestGenerateRefreshToken_Success(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newTestAuthService(db)

	user := models.User{ID: 1, Username: "rtuser", Role: models.RoleUser}
	db.Create(&user)

	plainToken, err := svc.GenerateRefreshToken(user)
	if err != nil {
		t.Fatalf("GenerateRefreshToken failed: %v", err)
	}
	if plainToken == "" {
		t.Fatal("expected non-empty refresh token")
	}
	if len(plainToken) < 10 {
		t.Error("refresh token seems too short")
	}

	// Verify a refresh token record was stored
	var count int64
	db.Model(&models.RefreshToken{}).Where("user_id = ?", user.ID).Count(&count)
	if count != 1 {
		t.Errorf("expected 1 refresh token in DB, got %d", count)
	}
}

// --- Token Validation (parse with wrong/malformed secrets) ---

func TestTokenValidation_WrongSecret(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newTestAuthService(db)

	user := models.User{ID: 1, Username: "testuser", Role: models.RoleUser}
	tokenString, err := svc.GenerateAccessToken(user)
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}

	// Parse with wrong secret
	_, err = jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte("wrong-secret"), nil
	})
	if err == nil {
		t.Error("expected error when parsing with wrong secret")
	}
}

func TestTokenValidation_MalformedToken(t *testing.T) {
	_, err := jwt.Parse("not.a.valid.jwt", func(token *jwt.Token) (interface{}, error) {
		return []byte(testJWTSecret), nil
	})
	if err == nil {
		t.Error("expected error for malformed token")
	}
}

// --- Token Rotation ---

func TestRotateTokens_Success(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newTestAuthService(db)

	user := models.User{ID: 1, Username: "rotuser", Role: models.RoleUser}
	db.Create(&user)

	plainToken, err := svc.GenerateRefreshToken(user)
	if err != nil {
		t.Fatalf("setup: GenerateRefreshToken failed: %v", err)
	}

	retUser, accessToken, newRefreshToken, err := svc.RotateTokens(plainToken)
	if err != nil {
		t.Fatalf("RotateTokens failed: %v", err)
	}
	if retUser.ID != user.ID {
		t.Errorf("expected user ID %d, got %d", user.ID, retUser.ID)
	}
	if accessToken == "" {
		t.Error("expected non-empty access token")
	}
	if newRefreshToken == "" {
		t.Error("expected non-empty new refresh token")
	}
	if newRefreshToken == plainToken {
		t.Error("expected new refresh token to differ from old one")
	}
}

func TestRotateTokens_InvalidToken(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newTestAuthService(db)

	_, _, _, err := svc.RotateTokens("rt_bogus_token_that_does_not_exist")
	if err != ErrInvalidRefreshToken {
		t.Errorf("expected ErrInvalidRefreshToken, got %v", err)
	}
}

func TestRotateTokens_OldTokenRevoked(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newTestAuthService(db)

	user := models.User{ID: 1, Username: "revokeuser", Role: models.RoleUser}
	db.Create(&user)

	plainToken, err := svc.GenerateRefreshToken(user)
	if err != nil {
		t.Fatalf("setup: GenerateRefreshToken failed: %v", err)
	}

	// First rotation succeeds
	_, _, _, err = svc.RotateTokens(plainToken)
	if err != nil {
		t.Fatalf("first RotateTokens failed: %v", err)
	}

	// Second rotation with same token should fail (token is revoked)
	_, _, _, err = svc.RotateTokens(plainToken)
	if err != ErrInvalidRefreshToken {
		t.Errorf("expected ErrInvalidRefreshToken for revoked token, got %v", err)
	}
}

func TestRotateTokens_ConcurrentSingleUse(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := newTestAuthService(db)

	user := models.User{ID: 1, Username: "raceuser", Role: models.RoleUser}
	db.Create(&user)

	plainToken, err := svc.GenerateRefreshToken(user)
	if err != nil {
		t.Fatalf("setup: GenerateRefreshToken failed: %v", err)
	}

	start := make(chan struct{})
	results := make(chan error, 2)
	var wg sync.WaitGroup

	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, _, _, rotateErr := svc.RotateTokens(plainToken)
			results <- rotateErr
		}()
	}

	close(start)
	wg.Wait()
	close(results)

	var successCount, invalidCount int
	for rotateErr := range results {
		switch rotateErr {
		case nil:
			successCount++
		case ErrInvalidRefreshToken:
			invalidCount++
		default:
			t.Fatalf("expected nil or ErrInvalidRefreshToken, got %v", rotateErr)
		}
	}

	if successCount != 1 || invalidCount != 1 {
		t.Fatalf("expected exactly one success and one invalid token error, got success=%d invalid=%d", successCount, invalidCount)
	}

	var activeCount int64
	db.Model(&models.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", user.ID).
		Count(&activeCount)
	if activeCount != 1 {
		t.Fatalf("expected exactly one active refresh token after concurrent rotate, got %d", activeCount)
	}
}

// --- Password Hashing ---

func TestPasswordHashing_BcryptRoundTrip(t *testing.T) {
	password := "my-secret-password-123!"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("bcrypt hash failed: %v", err)
	}

	if err := bcrypt.CompareHashAndPassword(hash, []byte(password)); err != nil {
		t.Error("bcrypt compare failed for correct password")
	}
	if err := bcrypt.CompareHashAndPassword(hash, []byte("wrong")); err == nil {
		t.Error("bcrypt compare should fail for wrong password")
	}
}
