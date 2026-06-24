package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

var (
	ErrOIDCProviderNotFound      = errors.New("oidc provider not found")
	ErrOIDCProviderInvalid       = errors.New("invalid oidc provider configuration")
	ErrOIDCProviderDuplicate     = errors.New("oidc provider already exists")
	ErrOIDCProviderInUse         = errors.New("oidc provider has linked identities")
	ErrOIDCProviderDiscovery     = errors.New("oidc provider discovery failed")
	ErrOIDCProviderConfiguration = errors.New("oidc provider runtime configuration failed")
	ErrOIDCProviderDenied        = errors.New("oidc provider denied request")
	ErrOIDCProviderSecretMissing = errors.New("client secret is required")
	ErrOIDCProviderDisabled      = errors.New("oidc provider is disabled")
	ErrOIDCInvalidRedirect       = errors.New("invalid oidc redirect path")
	ErrOIDCInvalidState          = errors.New("invalid oidc state")
	ErrOIDCValidationFailed      = errors.New("oidc validation failed")
	ErrOIDCCodeExchangeFailed    = errors.New("oidc code exchange failed")
	ErrOIDCIdentityNotLinked     = errors.New("oidc identity is not linked")
	ErrOIDCIdentityNotFound      = errors.New("oidc identity not found")
	ErrOIDCIdentityAlreadyLinked = errors.New("oidc identity is already linked")
	ErrOIDCAccountConflict       = errors.New("oidc account must be linked before login")
	ErrOIDCNoUsableSignInMethod  = errors.New("oidc unlink would remove last usable sign-in method")
	ErrOIDCTokenIssueFailed      = errors.New("failed to issue app tokens")
)

var oidcProviderNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{1,99}$`)

const oidcAuthStateTTL = 10 * time.Minute

type OIDCService struct {
	repo            *repository.OIDCRepository
	discover        OIDCDiscoveryFactory
	securityService *SecurityService
	authService     *AuthService
	now             func() time.Time
}

type OIDCRuntimeConfig struct {
	Provider     *oidc.Provider
	OAuth2Config oauth2.Config
}

type OIDCDiscoveryFactory func(ctx context.Context, issuerURL string) (*oidc.Provider, error)

func NewDefaultOIDCDiscoveryFactory() OIDCDiscoveryFactory {
	return oidc.NewProvider
}

func NewOIDCService(repo *repository.OIDCRepository, discover OIDCDiscoveryFactory) *OIDCService {
	if discover == nil {
		discover = NewDefaultOIDCDiscoveryFactory()
	}
	return &OIDCService{repo: repo, discover: discover, now: time.Now}
}

func (s *OIDCService) WithSecurity(securityService *SecurityService) *OIDCService {
	s.securityService = securityService
	return s
}

type OIDCAdminProviderInput struct {
	Name                 string                  `json:"name"`
	DisplayName          string                  `json:"displayName"`
	ProviderType         models.OIDCProviderType `json:"providerType"`
	Enabled              *bool                   `json:"enabled,omitempty"`
	IssuerURL            string                  `json:"issuerUrl"`
	ClientID             string                  `json:"clientId"`
	ClientSecret         string                  `json:"clientSecret,omitempty"`
	Scopes               []string                `json:"scopes"`
	CallbackPath         string                  `json:"callbackPath"`
	RequireVerifiedEmail *bool                   `json:"requireVerifiedEmail,omitempty"`
}

type OIDCAdminProviderDTO struct {
	ID                     uint                          `json:"id"`
	Name                   string                        `json:"name"`
	DisplayName            string                        `json:"displayName"`
	ProviderType           models.OIDCProviderType       `json:"providerType"`
	Enabled                bool                          `json:"enabled"`
	IssuerURL              string                        `json:"issuerUrl"`
	ClientID               string                        `json:"clientId"`
	ClientSecretConfigured bool                          `json:"clientSecretConfigured"`
	Scopes                 []string                      `json:"scopes"`
	CallbackPath           string                        `json:"callbackPath"`
	RequireVerifiedEmail   bool                          `json:"requireVerifiedEmail"`
	LastTestedAt           *time.Time                    `json:"lastTestedAt,omitempty"`
	LastTestStatus         models.OIDCProviderTestStatus `json:"lastTestStatus"`
	LastTestMessage        string                        `json:"lastTestMessage,omitempty"`
	CreatedAt              time.Time                     `json:"createdAt"`
	UpdatedAt              time.Time                     `json:"updatedAt"`
}

type OIDCProviderTestResult struct {
	Available             bool   `json:"available"`
	Message               string `json:"message"`
	Issuer                string `json:"issuer,omitempty"`
	AuthorizationEndpoint string `json:"authorizationEndpoint,omitempty"`
	TokenEndpoint         string `json:"tokenEndpoint,omitempty"`
}

type OIDCPublicProviderDTO struct {
	ID           uint                    `json:"id"`
	Name         string                  `json:"name"`
	DisplayName  string                  `json:"displayName"`
	ProviderType models.OIDCProviderType `json:"providerType"`
}

type OIDCStartLoginInput struct {
	RedirectPath string `json:"redirectPath"`
}

type OIDCStartLoginResult struct {
	AuthorizationURL string    `json:"authorizationUrl"`
	ExpiresAt        time.Time `json:"expiresAt"`
}

type OIDCLinkedIdentityDTO struct {
	ID                  uint       `json:"id"`
	ProviderID          uint       `json:"providerId"`
	ProviderDisplayName string     `json:"providerDisplayName"`
	Issuer              string     `json:"issuer"`
	SubjectPreview      string     `json:"subjectPreview"`
	Email               string     `json:"email,omitempty"`
	EmailVerified       bool       `json:"emailVerified"`
	CreatedAt           time.Time  `json:"createdAt"`
	LastLoginAt         *time.Time `json:"lastLoginAt,omitempty"`
}

type OIDCLinkCallbackResult struct {
	Message  string                `json:"message"`
	Identity OIDCLinkedIdentityDTO `json:"identity"`
}

type OIDCAuditContext struct {
	AdminID   uint
	ClientIP  string
	UserAgent string
}

func (s *OIDCService) WithAuth(authService *AuthService) *OIDCService {
	s.authService = authService
	return s
}

func (s *OIDCService) ListPublicProviders() ([]OIDCPublicProviderDTO, error) {
	providers, err := s.repo.ListEnabledProviders()
	if err != nil {
		return nil, err
	}
	result := make([]OIDCPublicProviderDTO, 0, len(providers))
	for _, provider := range providers {
		result = append(result, OIDCPublicProviderDTO{
			ID:           provider.ID,
			Name:         provider.Name,
			DisplayName:  provider.DisplayName,
			ProviderType: provider.ProviderType,
		})
	}
	return result, nil
}

func (s *OIDCService) StartLogin(ctx context.Context, providerID uint, redirectPath, requestOrigin string) (OIDCStartLoginResult, error) {
	return s.startFlow(ctx, providerID, nil, models.OIDCFlowTypeLogin, redirectPath, requestOrigin)
}

func (s *OIDCService) StartLink(ctx context.Context, providerID, userID uint, redirectPath, requestOrigin string) (OIDCStartLoginResult, error) {
	return s.startFlow(ctx, providerID, &userID, models.OIDCFlowTypeLink, redirectPath, requestOrigin)
}

func (s *OIDCService) startFlow(ctx context.Context, providerID uint, userID *uint, flowType models.OIDCFlowType, redirectPath, requestOrigin string) (OIDCStartLoginResult, error) {
	provider, err := s.enabledProvider(providerID)
	if err != nil {
		return OIDCStartLoginResult{}, err
	}
	redirectPath = normalizeOIDCRedirectPath(redirectPath)
	if !isSafeRelativeRedirectPath(redirectPath) {
		return OIDCStartLoginResult{}, ErrOIDCInvalidRedirect
	}
	runtime, err := s.BuildRuntimeConfig(ctx, *provider)
	if err != nil {
		return OIDCStartLoginResult{}, err
	}
	runtime.OAuth2Config.RedirectURL = absoluteOIDCURL(requestOrigin, oidcFlowCallbackPath(*provider, flowType))

	state, err := secureRandomURLToken(32)
	if err != nil {
		return OIDCStartLoginResult{}, err
	}
	nonce, err := secureRandomURLToken(32)
	if err != nil {
		return OIDCStartLoginResult{}, err
	}
	verifier, err := secureRandomURLToken(64)
	if err != nil {
		return OIDCStartLoginResult{}, err
	}
	expiresAt := s.now().Add(oidcAuthStateTTL)
	authState := models.OIDCAuthState{
		StateHash:        hashOIDCSecret(state),
		ProviderID:       provider.ID,
		FlowType:         flowType,
		UserID:           userID,
		PKCEVerifierHash: verifier,
		NonceHash:        hashOIDCSecret(nonce),
		RedirectPath:     redirectPath,
		ExpiresAt:        expiresAt,
	}
	if err := s.repo.CreateAuthState(&authState); err != nil {
		return OIDCStartLoginResult{}, err
	}
	authURL := runtime.OAuth2Config.AuthCodeURL(
		state,
		oauth2.SetAuthURLParam("nonce", nonce),
		oauth2.SetAuthURLParam("code_challenge", pkceChallenge(verifier)),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)
	return OIDCStartLoginResult{AuthorizationURL: authURL, ExpiresAt: expiresAt}, nil
}

func (s *OIDCService) CompleteLoginCallback(ctx context.Context, providerID uint, code, state, requestOrigin string, audit OIDCAuditContext) (AuthResult, error) {
	provider, providerErr := s.enabledProvider(providerID)
	if providerErr != nil {
		return AuthResult{}, providerErr
	}
	if strings.TrimSpace(code) == "" || strings.TrimSpace(state) == "" {
		s.recordLoginFailure(nil, "", *provider, audit, "missing callback parameters")
		return AuthResult{}, ErrOIDCInvalidState
	}
	consumed, err := s.repo.ConsumeAuthState(hashOIDCSecret(state), provider.ID, s.now())
	if err != nil {
		s.recordLoginFailure(nil, "", *provider, audit, "invalid or replayed state")
		return AuthResult{}, ErrOIDCInvalidState
	}
	if consumed.FlowType != models.OIDCFlowTypeLogin {
		s.recordLoginFailure(nil, "", *provider, audit, "state flow mismatch")
		return AuthResult{}, ErrOIDCInvalidState
	}
	verified, claims, err := s.exchangeAndValidateCallback(ctx, *provider, consumed, code, requestOrigin)
	if err != nil {
		s.recordLoginFailure(nil, "", *provider, audit, oidcFailureReason(err))
		return AuthResult{}, err
	}
	email := strings.TrimSpace(strings.ToLower(claims.Email))
	identity, err := s.repo.FindExternalIdentity(provider.ID, verified.Issuer, claims.Subject)
	if err != nil {
		if repository.IsRecordNotFound(err) {
			if email != "" && claims.EmailVerified {
				if user, userErr := s.repo.FindUserByEmail(email); userErr == nil && user != nil {
					userID := user.ID
					s.recordLoginFailure(&userID, user.Username, *provider, audit, "matching local email requires explicit link")
					return AuthResult{}, ErrOIDCAccountConflict
				}
			}
			s.recordLoginFailure(nil, "", *provider, audit, "external identity not linked")
			return AuthResult{}, ErrOIDCIdentityNotLinked
		}
		return AuthResult{}, err
	}
	user, err := s.repo.FindUserByID(identity.UserID)
	if err != nil {
		s.recordLoginFailure(&identity.UserID, "", *provider, audit, "linked user not found")
		return AuthResult{}, ErrOIDCIdentityNotLinked
	}
	if err := s.repo.UpdateExternalIdentityLastLogin(identity.ID, s.now()); err != nil {
		return AuthResult{}, err
	}
	if s.authService == nil {
		return AuthResult{}, ErrOIDCTokenIssueFailed
	}
	result, err := s.authService.IssueTokens(*user)
	if err != nil {
		return AuthResult{}, ErrOIDCTokenIssueFailed
	}
	if s.securityService != nil {
		s.securityService.RecordOIDCLoginSuccess(*user, provider.ID, provider.DisplayName, audit.ClientIP, audit.UserAgent)
	}
	return result, nil
}

func (s *OIDCService) CompleteLinkCallback(ctx context.Context, providerID uint, code, state, requestOrigin string, audit OIDCAuditContext) (OIDCLinkCallbackResult, error) {
	provider, providerErr := s.enabledProvider(providerID)
	if providerErr != nil {
		return OIDCLinkCallbackResult{}, providerErr
	}
	if strings.TrimSpace(code) == "" || strings.TrimSpace(state) == "" {
		s.recordLinkFailure(nil, "", *provider, audit, "missing callback parameters")
		return OIDCLinkCallbackResult{}, ErrOIDCInvalidState
	}
	consumed, err := s.repo.ConsumeAuthState(hashOIDCSecret(state), provider.ID, s.now())
	if err != nil {
		s.recordLinkFailure(nil, "", *provider, audit, "invalid or replayed state")
		return OIDCLinkCallbackResult{}, ErrOIDCInvalidState
	}
	if consumed.FlowType != models.OIDCFlowTypeLink || consumed.UserID == nil || *consumed.UserID == 0 {
		s.recordLinkFailure(nil, "", *provider, audit, "state flow mismatch")
		return OIDCLinkCallbackResult{}, ErrOIDCInvalidState
	}
	user, err := s.repo.FindUserByID(*consumed.UserID)
	if err != nil {
		s.recordLinkFailure(consumed.UserID, "", *provider, audit, "linking user not found")
		return OIDCLinkCallbackResult{}, ErrOIDCInvalidState
	}
	verified, claims, err := s.exchangeAndValidateCallback(ctx, *provider, consumed, code, requestOrigin)
	if err != nil {
		s.recordLinkFailure(&user.ID, user.Username, *provider, audit, oidcFailureReason(err))
		return OIDCLinkCallbackResult{}, err
	}
	email := strings.TrimSpace(strings.ToLower(claims.Email))
	if email != "" && claims.EmailVerified {
		if existingUser, userErr := s.repo.FindUserByEmail(email); userErr == nil && existingUser != nil && existingUser.ID != user.ID {
			s.recordLinkFailure(&user.ID, user.Username, *provider, audit, "verified email belongs to another local user")
			return OIDCLinkCallbackResult{}, ErrOIDCAccountConflict
		}
	}
	existing, err := s.repo.FindExternalIdentity(provider.ID, verified.Issuer, claims.Subject)
	if err == nil && existing != nil {
		if existing.UserID != user.ID {
			s.recordLinkFailure(&user.ID, user.Username, *provider, audit, "external identity already linked")
			return OIDCLinkCallbackResult{}, ErrOIDCIdentityAlreadyLinked
		}
		existing.Provider = *provider
		dto := linkedIdentityDTO(*existing)
		s.recordLinkSuccess(*user, *provider, audit)
		return OIDCLinkCallbackResult{Message: "OIDC identity linked", Identity: dto}, nil
	}
	if err != nil && !repository.IsRecordNotFound(err) {
		return OIDCLinkCallbackResult{}, err
	}
	identity := models.ExternalIdentity{
		UserID:        user.ID,
		ProviderID:    provider.ID,
		Provider:      *provider,
		Issuer:        verified.Issuer,
		Subject:       claims.Subject,
		Email:         email,
		EmailVerified: claims.EmailVerified,
		DisplayName:   strings.TrimSpace(claims.Name),
	}
	if err := s.repo.CreateExternalIdentity(&identity); err != nil {
		s.recordLinkFailure(&user.ID, user.Username, *provider, audit, "external identity already linked")
		return OIDCLinkCallbackResult{}, ErrOIDCIdentityAlreadyLinked
	}
	identity.Provider = *provider
	s.recordLinkSuccess(*user, *provider, audit)
	return OIDCLinkCallbackResult{Message: "OIDC identity linked", Identity: linkedIdentityDTO(identity)}, nil
}

func (s *OIDCService) ListLinkedIdentities(userID uint) ([]OIDCLinkedIdentityDTO, error) {
	identities, err := s.repo.ListExternalIdentitiesForUser(userID)
	if err != nil {
		return nil, err
	}
	result := make([]OIDCLinkedIdentityDTO, 0, len(identities))
	for _, identity := range identities {
		result = append(result, linkedIdentityDTO(identity))
	}
	return result, nil
}

func (s *OIDCService) UnlinkIdentity(identityID, userID uint, audit OIDCAuditContext) error {
	identity, err := s.repo.GetExternalIdentityForUser(identityID, userID)
	if err != nil {
		if repository.IsRecordNotFound(err) {
			return ErrOIDCIdentityNotFound
		}
		return err
	}
	user, err := s.repo.FindUserByID(userID)
	if err != nil {
		return err
	}
	if err := s.repo.DeleteExternalIdentityWithSignInGuard(identityID, userID); err != nil {
		if errors.Is(err, repository.ErrNoUsableSignInMethod) {
			s.recordUnlinkFailure(&user.ID, user.Username, identity.Provider, audit, "unlink would remove last usable sign-in method")
			return ErrOIDCNoUsableSignInMethod
		}
		if repository.IsRecordNotFound(err) {
			return ErrOIDCIdentityNotFound
		}
		return err
	}
	s.recordUnlinkSuccess(*user, identity.Provider, audit)
	return nil
}

func (s *OIDCService) RecordLoginFailure(providerID uint, audit OIDCAuditContext, reason string) {
	provider, err := s.repo.GetProviderByID(providerID)
	if err != nil {
		return
	}
	s.recordLoginFailure(nil, "", *provider, audit, reason)
}

func (s *OIDCService) exchangeAndValidateCallback(ctx context.Context, provider models.OIDCProvider, consumed *models.OIDCAuthState, code, requestOrigin string) (*oidc.IDToken, oidcLoginClaims, error) {
	runtime, err := s.BuildRuntimeConfig(ctx, provider)
	if err != nil {
		return nil, oidcLoginClaims{}, err
	}
	runtime.OAuth2Config.RedirectURL = absoluteOIDCURL(requestOrigin, oidcFlowCallbackPath(provider, consumed.FlowType))
	token, err := runtime.OAuth2Config.Exchange(ctx, code, oauth2.SetAuthURLParam("code_verifier", consumed.PKCEVerifierHash))
	if err != nil {
		return nil, oidcLoginClaims{}, ErrOIDCCodeExchangeFailed
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return nil, oidcLoginClaims{}, ErrOIDCValidationFailed
	}
	verified, err := runtime.Provider.Verifier(&oidc.Config{ClientID: provider.ClientID}).Verify(ctx, rawIDToken)
	if err != nil {
		return nil, oidcLoginClaims{}, ErrOIDCValidationFailed
	}
	claims := oidcLoginClaims{}
	if err := verified.Claims(&claims); err != nil {
		return nil, oidcLoginClaims{}, ErrOIDCValidationFailed
	}
	if claims.Subject == "" {
		return nil, oidcLoginClaims{}, ErrOIDCValidationFailed
	}
	if hashOIDCSecret(claims.Nonce) != consumed.NonceHash {
		return nil, oidcLoginClaims{}, ErrOIDCValidationFailed
	}
	email := strings.TrimSpace(strings.ToLower(claims.Email))
	if provider.RequireVerifiedEmail && (email == "" || !claims.EmailVerified) {
		return nil, oidcLoginClaims{}, ErrOIDCValidationFailed
	}
	claims.Email = email
	return verified, claims, nil
}

type oidcDiscoveryMetadata struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	JWKSURI               string `json:"jwks_uri"`
}

func (s *OIDCService) ListAdminProviders() ([]OIDCAdminProviderDTO, error) {
	providers, err := s.repo.ListProviders()
	if err != nil {
		return nil, err
	}
	result := make([]OIDCAdminProviderDTO, 0, len(providers))
	for _, provider := range providers {
		result = append(result, oidcProviderDTO(provider))
	}
	return result, nil
}

func (s *OIDCService) CreateAdminProvider(ctx context.Context, input OIDCAdminProviderInput, audit OIDCAuditContext) (OIDCAdminProviderDTO, error) {
	provider, err := s.providerFromInput(input, nil)
	if err != nil {
		return OIDCAdminProviderDTO{}, err
	}
	if strings.TrimSpace(input.ClientSecret) == "" {
		return OIDCAdminProviderDTO{}, ErrOIDCProviderSecretMissing
	}
	if err := s.ensureProviderNameAvailable(provider.Name, 0); err != nil {
		return OIDCAdminProviderDTO{}, err
	}
	if err := s.ensureProviderIssuerClientAvailable(provider.IssuerURL, provider.ClientID, 0); err != nil {
		return OIDCAdminProviderDTO{}, err
	}
	if provider.CallbackPath == "" {
		provider.CallbackPath = "/api/auth/oidc/0/callback"
	}
	if err := s.repo.CreateProvider(&provider); err != nil {
		return OIDCAdminProviderDTO{}, err
	}
	if input.CallbackPath == "" {
		provider.CallbackPath = defaultOIDCCallbackPath(provider.ID)
		if err := s.repo.SaveProvider(&provider); err != nil {
			return OIDCAdminProviderDTO{}, err
		}
	}
	s.recordProviderConfigChanged(audit, provider, "created")
	return oidcProviderDTO(provider), nil
}

func (s *OIDCService) UpdateAdminProvider(ctx context.Context, id uint, input OIDCAdminProviderInput, audit OIDCAuditContext) (OIDCAdminProviderDTO, error) {
	existing, err := s.repo.GetProviderByID(id)
	if err != nil {
		if repository.IsRecordNotFound(err) {
			return OIDCAdminProviderDTO{}, ErrOIDCProviderNotFound
		}
		return OIDCAdminProviderDTO{}, err
	}
	updated, err := s.providerFromInput(input, existing)
	if err != nil {
		return OIDCAdminProviderDTO{}, err
	}
	if err := s.ensureProviderNameAvailable(updated.Name, id); err != nil {
		return OIDCAdminProviderDTO{}, err
	}
	if err := s.ensureProviderIssuerClientAvailable(updated.IssuerURL, updated.ClientID, id); err != nil {
		return OIDCAdminProviderDTO{}, err
	}
	updated.ID = existing.ID
	updated.CreatedAt = existing.CreatedAt
	updated.LastTestedAt = existing.LastTestedAt
	updated.LastTestStatus = existing.LastTestStatus
	updated.LastTestMessage = existing.LastTestMessage
	if isOIDCRedactedSecretValue(input.ClientSecret) {
		updated.ClientSecret = existing.ClientSecret
	}
	if updated.CallbackPath == "" {
		updated.CallbackPath = defaultOIDCCallbackPath(id)
	}
	if err := s.repo.SaveProvider(&updated); err != nil {
		return OIDCAdminProviderDTO{}, err
	}
	s.recordProviderConfigChanged(audit, updated, "updated")
	return oidcProviderDTO(updated), nil
}

func (s *OIDCService) DeleteAdminProvider(id uint, audit OIDCAuditContext) error {
	provider, err := s.repo.GetProviderByID(id)
	if err != nil {
		if repository.IsRecordNotFound(err) {
			return ErrOIDCProviderNotFound
		}
		return err
	}
	count, err := s.repo.CountExternalIdentitiesForProvider(id)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrOIDCProviderInUse
	}
	rows, err := s.repo.DeleteProvider(id)
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrOIDCProviderNotFound
	}
	s.recordProviderConfigChanged(audit, *provider, "deleted")
	return nil
}

func (s *OIDCService) TestAdminProvider(ctx context.Context, id uint, audit OIDCAuditContext) (OIDCProviderTestResult, error) {
	provider, err := s.repo.GetProviderByID(id)
	if err != nil {
		if repository.IsRecordNotFound(err) {
			return OIDCProviderTestResult{}, ErrOIDCProviderNotFound
		}
		return OIDCProviderTestResult{}, err
	}
	result := s.testProviderDiscovery(ctx, *provider)
	now := s.now()
	provider.LastTestedAt = &now
	if result.Available {
		provider.LastTestStatus = models.OIDCProviderTestStatusOK
	} else {
		provider.LastTestStatus = models.OIDCProviderTestStatusFailed
	}
	provider.LastTestMessage = result.Message
	if err := s.repo.SaveProvider(provider); err != nil {
		return OIDCProviderTestResult{}, err
	}
	if !result.Available && s.securityService != nil {
		s.securityService.RecordOIDCProviderTestFailure(audit.AdminID, provider.ID, provider.DisplayName, audit.ClientIP, audit.UserAgent, result.Message)
	}
	if !result.Available {
		return result, ErrOIDCProviderDiscovery
	}
	return result, nil
}

func (s *OIDCService) BuildRuntimeConfig(ctx context.Context, provider models.OIDCProvider) (OIDCRuntimeConfig, error) {
	if err := validateProviderForRuntime(provider); err != nil {
		return OIDCRuntimeConfig{}, ErrOIDCProviderConfiguration
	}
	discoveryCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	discovered, err := s.discover(discoveryCtx, provider.IssuerURL)
	if err != nil {
		return OIDCRuntimeConfig{}, ErrOIDCProviderConfiguration
	}
	endpoint, err := oidcOAuth2Endpoint(discovered)
	if err != nil {
		return OIDCRuntimeConfig{}, ErrOIDCProviderConfiguration
	}
	return OIDCRuntimeConfig{
		Provider: discovered,
		OAuth2Config: oauth2.Config{
			ClientID:     provider.ClientID,
			ClientSecret: provider.ClientSecret,
			Endpoint:     endpoint,
			Scopes:       []string(provider.Scopes),
			RedirectURL:  provider.CallbackPath,
		},
	}, nil
}

func (s *OIDCService) enabledProvider(providerID uint) (*models.OIDCProvider, error) {
	provider, err := s.repo.GetProviderByID(providerID)
	if err != nil {
		if repository.IsRecordNotFound(err) {
			return nil, ErrOIDCProviderNotFound
		}
		return nil, err
	}
	if !provider.Enabled {
		return nil, ErrOIDCProviderDisabled
	}
	return provider, nil
}

func (s *OIDCService) providerFromInput(input OIDCAdminProviderInput, existing *models.OIDCProvider) (models.OIDCProvider, error) {
	provider := models.OIDCProvider{}
	if existing != nil {
		provider = *existing
	}
	provider.Name = strings.ToLower(strings.TrimSpace(input.Name))
	provider.DisplayName = strings.TrimSpace(input.DisplayName)
	provider.ProviderType = models.OIDCProviderType(strings.TrimSpace(string(input.ProviderType)))
	if input.Enabled != nil {
		provider.Enabled = *input.Enabled
	} else if existing == nil {
		provider.Enabled = false
	}
	provider.IssuerURL = strings.TrimRight(strings.TrimSpace(input.IssuerURL), "/")
	provider.ClientID = strings.TrimSpace(input.ClientID)
	if strings.TrimSpace(input.ClientSecret) != "" {
		provider.ClientSecret = strings.TrimSpace(input.ClientSecret)
	}
	provider.Scopes = models.StringList(normalizeOIDCScopes(input.Scopes))
	provider.CallbackPath = strings.TrimSpace(input.CallbackPath)
	if input.RequireVerifiedEmail != nil {
		provider.RequireVerifiedEmail = *input.RequireVerifiedEmail
	} else if existing == nil {
		provider.RequireVerifiedEmail = true
	}
	if provider.LastTestStatus == "" {
		provider.LastTestStatus = models.OIDCProviderTestStatusUnknown
	}
	if err := validateOIDCProvider(provider); err != nil {
		return provider, err
	}
	return provider, nil
}

func (s *OIDCService) ensureProviderNameAvailable(name string, currentID uint) error {
	existing, err := s.repo.GetProviderByName(name)
	if err != nil {
		if repository.IsRecordNotFound(err) {
			return nil
		}
		return err
	}
	if existing != nil && existing.ID != currentID {
		return ErrOIDCProviderDuplicate
	}
	return nil
}

func (s *OIDCService) ensureProviderIssuerClientAvailable(issuerURL, clientID string, currentID uint) error {
	existing, err := s.repo.GetProviderByIssuerAndClientID(issuerURL, clientID)
	if err != nil {
		if repository.IsRecordNotFound(err) {
			return nil
		}
		return err
	}
	if existing != nil && existing.ID != currentID {
		return ErrOIDCProviderDuplicate
	}
	return nil
}

func (s *OIDCService) testProviderDiscovery(ctx context.Context, provider models.OIDCProvider) OIDCProviderTestResult {
	if err := validateProviderForRuntime(provider); err != nil {
		return OIDCProviderTestResult{Available: false, Message: "Provider configuration is invalid"}
	}
	discoveryCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	discovered, err := s.discover(discoveryCtx, provider.IssuerURL)
	if err != nil {
		return OIDCProviderTestResult{Available: false, Message: "Discovery failed"}
	}
	metadata := oidcDiscoveryMetadata{}
	if err := discovered.Claims(&metadata); err != nil {
		return OIDCProviderTestResult{Available: false, Message: "Discovery metadata is invalid"}
	}
	endpoint, err := oidcOAuth2Endpoint(discovered)
	if err != nil {
		return OIDCProviderTestResult{Available: false, Message: "Discovery metadata is incomplete"}
	}
	if metadata.Issuer == "" || endpoint.AuthURL == "" || endpoint.TokenURL == "" {
		return OIDCProviderTestResult{Available: false, Message: "Discovery metadata is incomplete"}
	}
	return OIDCProviderTestResult{
		Available:             true,
		Message:               "Discovery succeeded",
		Issuer:                metadata.Issuer,
		AuthorizationEndpoint: endpoint.AuthURL,
		TokenEndpoint:         endpoint.TokenURL,
	}
}

func oidcOAuth2Endpoint(discovered *oidc.Provider) (oauth2.Endpoint, error) {
	metadata := oidcDiscoveryMetadata{}
	if err := discovered.Claims(&metadata); err != nil {
		return oauth2.Endpoint{}, err
	}
	authURL := strings.TrimSpace(metadata.AuthorizationEndpoint)
	tokenURL := strings.TrimSpace(metadata.TokenEndpoint)
	if authURL == "" || tokenURL == "" || authURL == tokenURL {
		return oauth2.Endpoint{}, ErrOIDCProviderDiscovery
	}
	endpoint := discovered.Endpoint()
	endpoint.AuthURL = authURL
	endpoint.TokenURL = tokenURL
	return endpoint, nil
}

func validateOIDCProvider(provider models.OIDCProvider) error {
	if !oidcProviderNamePattern.MatchString(provider.Name) {
		return ErrOIDCProviderInvalid
	}
	if provider.DisplayName == "" || len(provider.DisplayName) > 150 {
		return ErrOIDCProviderInvalid
	}
	if !validOIDCProviderType(provider.ProviderType) {
		return ErrOIDCProviderInvalid
	}
	if err := validateOIDCIssuerURL(provider.IssuerURL); err != nil {
		return err
	}
	if provider.ClientID == "" {
		return ErrOIDCProviderInvalid
	}
	if len(provider.Scopes) == 0 || !scopeContainsOpenID(provider.Scopes) {
		return ErrOIDCProviderInvalid
	}
	if provider.CallbackPath != "" && !isSafeRelativeCallbackPath(provider.CallbackPath) {
		return ErrOIDCProviderInvalid
	}
	return nil
}

func validateProviderForRuntime(provider models.OIDCProvider) error {
	if err := validateOIDCProvider(provider); err != nil {
		return err
	}
	if provider.ClientSecret == "" {
		return ErrOIDCProviderSecretMissing
	}
	if provider.CallbackPath == "" {
		return ErrOIDCProviderInvalid
	}
	return nil
}

func validOIDCProviderType(providerType models.OIDCProviderType) bool {
	switch providerType {
	case models.OIDCProviderTypeEntra, models.OIDCProviderTypePocketID, models.OIDCProviderTypeGeneric:
		return true
	default:
		return false
	}
}

func validateOIDCIssuerURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return ErrOIDCProviderInvalid
	}
	if parsed.Scheme != "https" && !(parsed.Scheme == "http" && isLocalhost(parsed.Hostname())) {
		return ErrOIDCProviderInvalid
	}
	return nil
}

func isLocalhost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func isSafeRelativeCallbackPath(path string) bool {
	if !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") || strings.Contains(path, "\\") {
		return false
	}
	parsed, err := url.Parse(path)
	return err == nil && !parsed.IsAbs() && parsed.Host == "" && parsed.RawQuery == "" && parsed.Fragment == ""
}

func normalizeOIDCRedirectPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "/"
	}
	return path
}

func isSafeRelativeRedirectPath(path string) bool {
	if !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") || strings.Contains(path, "\\") {
		return false
	}
	parsed, err := url.Parse(path)
	return err == nil && !parsed.IsAbs() && parsed.Host == ""
}

func absoluteOIDCURL(origin, path string) string {
	return strings.TrimRight(origin, "/") + path
}

func secureRandomURLToken(byteCount int) (string, error) {
	buf := make([]byte, byteCount)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func hashOIDCSecret(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func pkceChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func normalizeOIDCScopes(scopes []string) []string {
	if len(scopes) == 0 {
		return []string{"openid", "profile", "email"}
	}
	seen := map[string]bool{}
	normalized := make([]string, 0, len(scopes)+1)
	for _, scope := range scopes {
		value := strings.TrimSpace(scope)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		normalized = append(normalized, value)
	}
	if !seen["openid"] {
		normalized = append([]string{"openid"}, normalized...)
	}
	return normalized
}

func scopeContainsOpenID(scopes []string) bool {
	for _, scope := range scopes {
		if scope == "openid" {
			return true
		}
	}
	return false
}

func isOIDCRedactedSecretValue(secret string) bool {
	normalized := strings.ToLower(strings.TrimSpace(secret))
	return normalized == "" ||
		normalized == "configured" ||
		normalized == "redacted" ||
		normalized == "<redacted>" ||
		normalized == "********" ||
		normalized == "••••••••"
}

func defaultOIDCCallbackPath(providerID uint) string {
	return fmt.Sprintf("/api/auth/oidc/%d/callback", providerID)
}

func defaultOIDCLinkCallbackPath(providerID uint) string {
	return fmt.Sprintf("/api/auth/oidc/%d/link/callback", providerID)
}

func oidcFlowCallbackPath(provider models.OIDCProvider, flowType models.OIDCFlowType) string {
	if flowType == models.OIDCFlowTypeLink {
		return defaultOIDCLinkCallbackPath(provider.ID)
	}
	return provider.CallbackPath
}

func oidcProviderDTO(provider models.OIDCProvider) OIDCAdminProviderDTO {
	return OIDCAdminProviderDTO{
		ID:                     provider.ID,
		Name:                   provider.Name,
		DisplayName:            provider.DisplayName,
		ProviderType:           provider.ProviderType,
		Enabled:                provider.Enabled,
		IssuerURL:              provider.IssuerURL,
		ClientID:               provider.ClientID,
		ClientSecretConfigured: provider.ClientSecretConfigured(),
		Scopes:                 []string(provider.Scopes),
		CallbackPath:           provider.CallbackPath,
		RequireVerifiedEmail:   provider.RequireVerifiedEmail,
		LastTestedAt:           provider.LastTestedAt,
		LastTestStatus:         provider.LastTestStatus,
		LastTestMessage:        provider.LastTestMessage,
		CreatedAt:              provider.CreatedAt,
		UpdatedAt:              provider.UpdatedAt,
	}
}

type oidcLoginClaims struct {
	Subject       string `json:"sub"`
	Nonce         string `json:"nonce"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
}

func linkedIdentityDTO(identity models.ExternalIdentity) OIDCLinkedIdentityDTO {
	return OIDCLinkedIdentityDTO{
		ID:                  identity.ID,
		ProviderID:          identity.ProviderID,
		ProviderDisplayName: identity.Provider.DisplayName,
		Issuer:              identity.Issuer,
		SubjectPreview:      subjectPreview(identity.Subject),
		Email:               identity.Email,
		EmailVerified:       identity.EmailVerified,
		CreatedAt:           identity.CreatedAt,
		LastLoginAt:         identity.LastLoginAt,
	}
}

func subjectPreview(subject string) string {
	subject = strings.TrimSpace(subject)
	if len(subject) <= 8 {
		return subject
	}
	return subject[:8] + "..."
}

func oidcFailureReason(err error) string {
	switch {
	case errors.Is(err, ErrOIDCProviderConfiguration):
		return "provider configuration failed"
	case errors.Is(err, ErrOIDCCodeExchangeFailed):
		return "code exchange failed"
	case errors.Is(err, ErrOIDCValidationFailed):
		return "id token validation failed"
	default:
		return "oidc request failed"
	}
}

func (s *OIDCService) recordProviderConfigChanged(audit OIDCAuditContext, provider models.OIDCProvider, action string) {
	if s.securityService == nil || audit.AdminID == 0 {
		return
	}
	s.securityService.RecordOIDCProviderConfigChanged(audit.AdminID, provider.ID, provider.DisplayName, audit.ClientIP, audit.UserAgent, action)
}

func (s *OIDCService) recordLoginFailure(userID *uint, username string, provider models.OIDCProvider, audit OIDCAuditContext, reason string) {
	if s.securityService == nil {
		return
	}
	s.securityService.RecordOIDCLoginFailure(userID, username, provider.ID, provider.DisplayName, audit.ClientIP, audit.UserAgent, reason)
}

func (s *OIDCService) recordLinkSuccess(user models.User, provider models.OIDCProvider, audit OIDCAuditContext) {
	if s.securityService == nil {
		return
	}
	s.securityService.RecordOIDCLinkSuccess(user, provider.ID, provider.DisplayName, audit.ClientIP, audit.UserAgent)
}

func (s *OIDCService) recordLinkFailure(userID *uint, username string, provider models.OIDCProvider, audit OIDCAuditContext, reason string) {
	if s.securityService == nil {
		return
	}
	s.securityService.RecordOIDCLinkFailure(userID, username, provider.ID, provider.DisplayName, audit.ClientIP, audit.UserAgent, reason)
}

func (s *OIDCService) recordUnlinkSuccess(user models.User, provider models.OIDCProvider, audit OIDCAuditContext) {
	if s.securityService == nil {
		return
	}
	s.securityService.RecordOIDCUnlinkSuccess(user, provider.ID, provider.DisplayName, audit.ClientIP, audit.UserAgent)
}

func (s *OIDCService) recordUnlinkFailure(userID *uint, username string, provider models.OIDCProvider, audit OIDCAuditContext, reason string) {
	if s.securityService == nil {
		return
	}
	s.securityService.RecordOIDCUnlinkFailure(userID, username, provider.ID, provider.DisplayName, audit.ClientIP, audit.UserAgent, reason)
}
