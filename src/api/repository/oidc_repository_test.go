package repository

import (
	"errors"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupOIDCRepositoryTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.OIDCProvider{}, &models.ExternalIdentity{}, &models.OIDCAuthState{}); err != nil {
		t.Fatalf("failed to migrate test db: %v", err)
	}
	return db
}

func createOIDCRepositoryTestUser(t *testing.T, db *gorm.DB, username string) models.User {
	t.Helper()

	user := models.User{
		Username:     username,
		Email:        username + "@example.com",
		PasswordHash: "local-password-hash",
		Role:         models.RoleUser,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	return user
}

func createOIDCRepositoryTestProvider(t *testing.T, repo *OIDCRepository, name string) models.OIDCProvider {
	t.Helper()

	provider := models.OIDCProvider{
		Name:                 name,
		DisplayName:          name,
		ProviderType:         models.OIDCProviderTypeGeneric,
		Enabled:              true,
		IssuerURL:            "https://" + name + ".example.com",
		ClientID:             "client-" + name,
		ClientSecret:         "secret-" + name,
		Scopes:               models.StringList{"openid", "profile", "email"},
		CallbackPath:         "/api/auth/oidc/callback/" + name,
		RequireVerifiedEmail: true,
	}
	if err := repo.CreateProvider(&provider); err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	return provider
}

func TestOIDCRepositoryExternalIdentityUniqueness(t *testing.T) {
	db := setupOIDCRepositoryTestDB(t)
	repo := NewOIDCRepository(db)
	user := createOIDCRepositoryTestUser(t, db, "collector")
	otherUser := createOIDCRepositoryTestUser(t, db, "other")
	provider := createOIDCRepositoryTestProvider(t, repo, "entra")
	otherProvider := createOIDCRepositoryTestProvider(t, repo, "pocket")

	first := models.ExternalIdentity{
		UserID:        user.ID,
		ProviderID:    provider.ID,
		Issuer:        "https://issuer.example.com",
		Subject:       "subject-123",
		Email:         "collector@example.com",
		EmailVerified: true,
	}
	if err := repo.CreateExternalIdentity(&first); err != nil {
		t.Fatalf("failed to create first external identity: %v", err)
	}

	duplicate := models.ExternalIdentity{
		UserID:        otherUser.ID,
		ProviderID:    provider.ID,
		Issuer:        first.Issuer,
		Subject:       first.Subject,
		Email:         "other@example.com",
		EmailVerified: true,
	}
	if err := repo.CreateExternalIdentity(&duplicate); err == nil {
		t.Fatal("expected duplicate provider/issuer/subject identity to fail")
	}

	sameSubjectDifferentProvider := models.ExternalIdentity{
		UserID:        otherUser.ID,
		ProviderID:    otherProvider.ID,
		Issuer:        first.Issuer,
		Subject:       first.Subject,
		Email:         "other@example.com",
		EmailVerified: true,
	}
	if err := repo.CreateExternalIdentity(&sameSubjectDifferentProvider); err != nil {
		t.Fatalf("expected same subject from different provider to be allowed, got %v", err)
	}

	sameSubjectDifferentIssuer := models.ExternalIdentity{
		UserID:        otherUser.ID,
		ProviderID:    provider.ID,
		Issuer:        "https://issuer-two.example.com",
		Subject:       first.Subject,
		Email:         "other@example.com",
		EmailVerified: true,
	}
	if err := repo.CreateExternalIdentity(&sameSubjectDifferentIssuer); err != nil {
		t.Fatalf("expected same subject from different issuer to be allowed, got %v", err)
	}
}

func TestOIDCRepositoryConsumeAuthStatePreventsReplay(t *testing.T) {
	db := setupOIDCRepositoryTestDB(t)
	repo := NewOIDCRepository(db)
	provider := createOIDCRepositoryTestProvider(t, repo, "entra")
	now := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)

	state := models.OIDCAuthState{
		StateHash:        "state-hash",
		ProviderID:       provider.ID,
		FlowType:         models.OIDCFlowTypeLogin,
		PKCEVerifierHash: "pkce-hash",
		NonceHash:        "nonce-hash",
		RedirectPath:     "/",
		ExpiresAt:        now.Add(10 * time.Minute),
	}
	if err := repo.CreateAuthState(&state); err != nil {
		t.Fatalf("failed to create auth state: %v", err)
	}

	consumed, err := repo.ConsumeAuthState(state.StateHash, provider.ID, now)
	if err != nil {
		t.Fatalf("expected first consume to succeed, got %v", err)
	}
	if consumed.ConsumedAt == nil || !consumed.ConsumedAt.Equal(now) {
		t.Fatalf("expected consumed_at to be set to %v, got %v", now, consumed.ConsumedAt)
	}

	replayed, err := repo.ConsumeAuthState(state.StateHash, provider.ID, now.Add(time.Second))
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected replay to fail with record not found, got state=%+v err=%v", replayed, err)
	}

	var stored models.OIDCAuthState
	if err := db.First(&stored, state.ID).Error; err != nil {
		t.Fatalf("failed to reload auth state: %v", err)
	}
	if stored.ConsumedAt == nil || !stored.ConsumedAt.Equal(now) {
		t.Fatalf("expected stored consumed_at to remain first consume time %v, got %v", now, stored.ConsumedAt)
	}
}

func TestOIDCRepositoryConsumeAuthStateRejectsExpiredOrWrongProvider(t *testing.T) {
	db := setupOIDCRepositoryTestDB(t)
	repo := NewOIDCRepository(db)
	provider := createOIDCRepositoryTestProvider(t, repo, "entra")
	otherProvider := createOIDCRepositoryTestProvider(t, repo, "pocket")
	now := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)

	expired := models.OIDCAuthState{
		StateHash:        "expired-state",
		ProviderID:       provider.ID,
		FlowType:         models.OIDCFlowTypeLogin,
		PKCEVerifierHash: "pkce-hash",
		NonceHash:        "nonce-hash",
		RedirectPath:     "/",
		ExpiresAt:        now.Add(-time.Minute),
	}
	if err := repo.CreateAuthState(&expired); err != nil {
		t.Fatalf("failed to create expired auth state: %v", err)
	}
	if _, err := repo.ConsumeAuthState(expired.StateHash, provider.ID, now); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected expired state consume to fail with record not found, got %v", err)
	}

	wrongProvider := models.OIDCAuthState{
		StateHash:        "wrong-provider-state",
		ProviderID:       provider.ID,
		FlowType:         models.OIDCFlowTypeLogin,
		PKCEVerifierHash: "pkce-hash",
		NonceHash:        "nonce-hash",
		RedirectPath:     "/",
		ExpiresAt:        now.Add(10 * time.Minute),
	}
	if err := repo.CreateAuthState(&wrongProvider); err != nil {
		t.Fatalf("failed to create wrong-provider auth state: %v", err)
	}
	if _, err := repo.ConsumeAuthState(wrongProvider.StateHash, otherProvider.ID, now); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected wrong-provider state consume to fail with record not found, got %v", err)
	}

	var unconsumedCount int64
	if err := db.Model(&models.OIDCAuthState{}).Where("consumed_at IS NULL").Count(&unconsumedCount).Error; err != nil {
		t.Fatalf("failed to count unconsumed states: %v", err)
	}
	if unconsumedCount != 2 {
		t.Fatalf("expected rejected states to remain unconsumed, got %d", unconsumedCount)
	}
}
