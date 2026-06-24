package repository

import (
	"errors"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

var ErrNoUsableSignInMethod = errors.New("no usable sign-in method would remain")

type OIDCRepository struct {
	db *gorm.DB
}

func NewOIDCRepository(db *gorm.DB) *OIDCRepository {
	return &OIDCRepository{db: db}
}

func (r *OIDCRepository) CreateProvider(provider *models.OIDCProvider) error {
	return r.db.Create(provider).Error
}

func (r *OIDCRepository) GetProviderByID(id uint) (*models.OIDCProvider, error) {
	var provider models.OIDCProvider
	err := r.db.First(&provider, id).Error
	return &provider, err
}

func (r *OIDCRepository) GetProviderByName(name string) (*models.OIDCProvider, error) {
	var provider models.OIDCProvider
	err := r.db.Where("name = ?", name).First(&provider).Error
	return &provider, err
}

func (r *OIDCRepository) GetProviderByIssuerAndClientID(issuerURL, clientID string) (*models.OIDCProvider, error) {
	var provider models.OIDCProvider
	err := r.db.Where("issuer_url = ? AND client_id = ?", issuerURL, clientID).First(&provider).Error
	return &provider, err
}

func (r *OIDCRepository) ListProviders() ([]models.OIDCProvider, error) {
	var providers []models.OIDCProvider
	err := r.db.Order("display_name ASC, id ASC").Find(&providers).Error
	return providers, err
}

func (r *OIDCRepository) ListEnabledProviders() ([]models.OIDCProvider, error) {
	var providers []models.OIDCProvider
	err := r.db.Where("enabled = ?", true).Order("display_name ASC, id ASC").Find(&providers).Error
	return providers, err
}

func (r *OIDCRepository) SaveProvider(provider *models.OIDCProvider) error {
	return r.db.Save(provider).Error
}

func (r *OIDCRepository) DeleteProvider(id uint) (int64, error) {
	result := r.db.Delete(&models.OIDCProvider{}, id)
	return result.RowsAffected, result.Error
}

func (r *OIDCRepository) CountExternalIdentitiesForProvider(providerID uint) (int64, error) {
	var count int64
	err := r.db.Model(&models.ExternalIdentity{}).Where("provider_id = ?", providerID).Count(&count).Error
	return count, err
}

func (r *OIDCRepository) CreateExternalIdentity(identity *models.ExternalIdentity) error {
	return r.db.Create(identity).Error
}

func (r *OIDCRepository) FindExternalIdentity(providerID uint, issuer, subject string) (*models.ExternalIdentity, error) {
	var identity models.ExternalIdentity
	err := r.db.Where("provider_id = ? AND issuer = ? AND subject = ?", providerID, issuer, subject).First(&identity).Error
	return &identity, err
}

func (r *OIDCRepository) ListExternalIdentitiesForUser(userID uint) ([]models.ExternalIdentity, error) {
	var identities []models.ExternalIdentity
	err := r.db.Where("user_id = ?", userID).Preload("Provider").Order("created_at DESC, id DESC").Find(&identities).Error
	return identities, err
}

func (r *OIDCRepository) CountExternalIdentitiesForUser(userID uint) (int64, error) {
	var count int64
	err := r.db.Model(&models.ExternalIdentity{}).Where("user_id = ?", userID).Count(&count).Error
	return count, err
}

func (r *OIDCRepository) GetExternalIdentityForUser(identityID, userID uint) (*models.ExternalIdentity, error) {
	var identity models.ExternalIdentity
	err := r.db.Where("id = ? AND user_id = ?", identityID, userID).Preload("Provider").First(&identity).Error
	return &identity, err
}

func (r *OIDCRepository) FindExternalIdentitiesByEmail(email string) ([]models.ExternalIdentity, error) {
	var identities []models.ExternalIdentity
	err := r.db.Where("email = ?", email).Preload("Provider").Find(&identities).Error
	return identities, err
}

func (r *OIDCRepository) FindVerifiedExternalIdentitiesByEmail(email string) ([]models.ExternalIdentity, error) {
	var identities []models.ExternalIdentity
	err := r.db.Where("email = ? AND email_verified = ?", email, true).Preload("Provider").Find(&identities).Error
	return identities, err
}

func (r *OIDCRepository) UpdateExternalIdentityLastLogin(identityID uint, at time.Time) error {
	return r.db.Model(&models.ExternalIdentity{}).Where("id = ?", identityID).Update("last_login_at", at).Error
}

func (r *OIDCRepository) DeleteExternalIdentity(identityID, userID uint) (int64, error) {
	result := r.db.Where("id = ? AND user_id = ?", identityID, userID).Delete(&models.ExternalIdentity{})
	return result.RowsAffected, result.Error
}

func (r *OIDCRepository) DeleteExternalIdentityWithSignInGuard(identityID, userID uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var identity models.ExternalIdentity
		if err := tx.Where("id = ? AND user_id = ?", identityID, userID).First(&identity).Error; err != nil {
			return err
		}

		var user models.User
		if err := tx.First(&user, userID).Error; err != nil {
			return err
		}

		var otherOIDCCount int64
		if err := tx.Model(&models.ExternalIdentity{}).
			Where("user_id = ? AND id <> ?", userID, identityID).
			Count(&otherOIDCCount).Error; err != nil {
			return err
		}

		var webAuthnCount int64
		if err := tx.Model(&models.WebAuthnCredential{}).
			Where("user_id = ?", userID).
			Count(&webAuthnCount).Error; err != nil {
			return err
		}

		if user.PasswordHash == "" && webAuthnCount == 0 && otherOIDCCount == 0 {
			return ErrNoUsableSignInMethod
		}

		result := tx.Where("id = ? AND user_id = ?", identityID, userID).Delete(&models.ExternalIdentity{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

func (r *OIDCRepository) FindUserByEmail(email string) (*models.User, error) {
	var user models.User
	err := r.db.Where("email = ?", email).First(&user).Error
	return &user, err
}

func (r *OIDCRepository) FindUserByID(userID uint) (*models.User, error) {
	var user models.User
	err := r.db.First(&user, userID).Error
	return &user, err
}

func (r *OIDCRepository) CreateAuthState(state *models.OIDCAuthState) error {
	return r.db.Create(state).Error
}

func (r *OIDCRepository) GetAuthStateByHash(stateHash string) (*models.OIDCAuthState, error) {
	var state models.OIDCAuthState
	err := r.db.Where("state_hash = ?", stateHash).First(&state).Error
	return &state, err
}

func (r *OIDCRepository) ConsumeAuthState(stateHash string, providerID uint, now time.Time) (*models.OIDCAuthState, error) {
	var state models.OIDCAuthState
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("state_hash = ? AND provider_id = ? AND consumed_at IS NULL AND expires_at > ?", stateHash, providerID, now).First(&state).Error; err != nil {
			return err
		}
		update := tx.Model(&models.OIDCAuthState{}).
			Where("id = ? AND consumed_at IS NULL", state.ID).
			Update("consumed_at", now)
		if update.Error != nil {
			return update.Error
		}
		if update.RowsAffected != 1 {
			return gorm.ErrRecordNotFound
		}
		state.ConsumedAt = &now
		return nil
	})
	return &state, err
}

func (r *OIDCRepository) DeleteExpiredAuthStates(now time.Time) (int64, error) {
	result := r.db.Where("expires_at <= ?", now).Delete(&models.OIDCAuthState{})
	return result.RowsAffected, result.Error
}
