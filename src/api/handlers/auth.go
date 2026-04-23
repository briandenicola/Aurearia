package handlers

import (
	"log"
	"net/http"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	JWTSecret string
	repo      *repository.AuthRepository
	svc       *services.AuthService
}

type loginRequest struct {
	Username string `json:"username" binding:"required" example:"admin"`
	Password string `json:"password" binding:"required" example:"password123"`
}

type registerRequest struct {
	Username string `json:"username" binding:"required,min=3" example:"admin"`
	Password string `json:"password" binding:"required,min=6" example:"password123"`
	Email    string `json:"email" binding:"required,email" example:"user@example.com"`
}

type refreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

func NewAuthHandler(jwtSecret string, repo *repository.AuthRepository, svc *services.AuthService) *AuthHandler {
	return &AuthHandler{JWTSecret: jwtSecret, repo: repo, svc: svc}
}

// Register creates a new user account. The first user registered becomes an admin.
//
//	@Summary		Register a new user
//	@Description	Create a new user account. The first registered user is assigned the admin role. Returns access token (15min) and refresh token (30 days).
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		registerRequest	true	"Registration credentials"
//	@Success		201		{object}	AuthResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		409		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Router			/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[handler] Register: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	user, err := h.svc.RegisterUser(req.Username, req.Email, req.Password)
	if err != nil {
		switch err {
		case services.ErrHashingFailed:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		case services.ErrUsernameExists:
			c.JSON(http.StatusConflict, gin.H{"error": "Username already exists"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Registration failed"})
		}
		return
	}

	h.issueTokens(c, *user, http.StatusCreated)
}

// Login authenticates a user and returns JWT access and refresh tokens.
//
//	@Summary		Login
//	@Description	Authenticate with username and password. Returns access token (15min) and refresh token (30 days).
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		loginRequest	true	"Login credentials"
//	@Success		200		{object}	AuthResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Router			/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[handler] Login: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	user, err := h.svc.AuthenticateUser(req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	h.issueTokens(c, *user, http.StatusOK)
}

// Refresh exchanges a valid refresh token for a new access token and refresh token (rolling).
//
//	@Summary		Refresh tokens
//	@Description	Exchange a valid refresh token for new access and refresh tokens. The old refresh token is revoked (rolling refresh).
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		refreshRequest	true	"Refresh token"
//	@Success		200		{object}	AuthResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Router			/auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[handler] Refresh: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	user, accessToken, refreshToken, err := h.svc.RotateTokens(req.RefreshToken)
	if err != nil {
		switch err {
		case services.ErrInvalidRefreshToken:
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		case services.ErrRefreshTokenExpired:
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Refresh token expired"})
		case services.ErrInvalidCredentials:
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to rotate refresh token"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":        accessToken,
		"refreshToken": refreshToken,
		"user": gin.H{
			"id":         user.ID,
			"username":   user.Username,
			"role":       user.Role,
			"email":      user.Email,
			"avatarPath": user.AvatarPath,
			"isPublic":   user.IsPublic,
			"bio":        user.Bio,
			"zipCode":    user.ZipCode,
		},
	})
}

// issueTokens generates and returns both access and refresh tokens.
func (h *AuthHandler) issueTokens(c *gin.Context, user models.User, statusCode int) {
	accessToken, err := h.svc.GenerateAccessToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate access token"})
		return
	}

	refreshToken, err := h.svc.GenerateRefreshToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate refresh token"})
		return
	}

	c.JSON(statusCode, gin.H{
		"token":        accessToken,
		"refreshToken": refreshToken,
		"user": gin.H{
			"id":         user.ID,
			"username":   user.Username,
			"role":       user.Role,
			"email":      user.Email,
			"avatarPath": user.AvatarPath,
			"isPublic":   user.IsPublic,
			"bio":        user.Bio,
			"zipCode":    user.ZipCode,
		},
	})
}

// NeedsSetup returns whether the first user has been created yet.
//
//	@Summary		Check setup status
//	@Description	Returns whether the application needs initial setup (no users exist yet).
//	@Tags			Auth
//	@Produce		json
//	@Success		200	{object}	SetupResponse
//	@Router			/auth/setup [get]
func (h *AuthHandler) NeedsSetup(c *gin.Context) {
	count := h.repo.CountUsers()
	c.JSON(http.StatusOK, gin.H{"needsSetup": count == 0})
}
