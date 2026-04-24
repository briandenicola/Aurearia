package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type UserHandler struct {
	UploadDir string
	repo      *repository.UserRepository
	logger    *services.Logger
}

func NewUserHandler(uploadDir string, repo *repository.UserRepository, logger *services.Logger) *UserHandler {
	return &UserHandler{UploadDir: uploadDir, repo: repo, logger: logger}
}

// ChangePassword allows a user to change their own password.
//
//	@Summary		Change password
//	@Description	Change the authenticated user's password. Requires the current password for verification.
//	@Tags			User
//	@Accept			json
//	@Produce		json
//	@Param			body	body		ChangePasswordRequest	true	"Password change request"
//	@Success		200		{object}	PasswordChangedResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/auth/change-password [post]
func (h *UserHandler) ChangePassword(c *gin.Context) {
	userID := c.GetUint("userId")

	var req struct {
		CurrentPassword string `json:"currentPassword" binding:"required"`
		NewPassword     string `json:"newPassword" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	var user models.User
	if found, err := h.repo.FindByID(userID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	} else {
		user = *found
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Current password is incorrect"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	if err := h.repo.UpdateField(&user, "password_hash", string(hash)); err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to update password", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Password changed"})
}

// GetMe returns the current authenticated user's info.
//
//	@Summary		Get current user
//	@Description	Returns profile information for the authenticated user.
//	@Tags			User
//	@Produce		json
//	@Success		200	{object}	UserInfoResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/auth/me [get]
func (h *UserHandler) GetMe(c *gin.Context) {
	userID := c.GetUint("userId")

	var user models.User
	if found, err := h.repo.FindByID(userID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	} else {
		user = *found
	}

	c.JSON(http.StatusOK, gin.H{
		"id":                  user.ID,
		"username":            user.Username,
		"role":                user.Role,
		"email":               user.Email,
		"avatarPath":          user.AvatarPath,
		"isPublic":            user.IsPublic,
		"bio":                 user.Bio,
		"zipCode":             user.ZipCode,
		"emailMissing":        user.Email == "",
		"createdAt":           user.CreatedAt,
		"numisBidsUsername":   user.NumisBidsUsername,
		"numisBidsConfigured": user.NumisBidsUsername != "" && user.NumisBidsPassword != "",
	})
}

// ExportCollection exports the current user's coins and images as a zip archive.
//
//	@Summary		Export collection
//	@Description	Downloads all coins and images as a ZIP archive containing coins.json and image files.
//	@Tags			User
//	@Produce		application/zip
//	@Success		200	"ZIP archive"
//	@Failure		401	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/user/export [get]
func (h *UserHandler) ExportCollection(c *gin.Context) {
	userID := c.GetUint("userId")

	var coins []models.Coin
	coins, _ = h.repo.GetCoinsWithImages(userID)

	filename := fmt.Sprintf("my-coins-export-%s.zip", time.Now().Format("2006-01-02"))
	writeCollectionZip(c, coins, h.UploadDir, filename, h.logger)
}

// ExportCatalogPDF generates a styled PDF catalog of the user's collection.
//
//	@Summary		Export PDF catalog
//	@Description	Generates a PDF catalog with photos, grades, provenance, and valuations.
//	@Tags			User
//	@Produce		application/pdf
//	@Success		200	"PDF document"
//	@Failure		401	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/user/export/catalog [get]
func (h *UserHandler) ExportCatalogPDF(c *gin.Context) {
	userID := c.GetUint("userId")

	var coins []models.Coin
	coins, _ = h.repo.GetCoinsWithImages(userID)

	// Get username for cover page
	user, _ := h.repo.FindByID(userID)
	username := "Collector"
	if user != nil {
		username = user.Username
	}

	pdf, err := writeCatalogPDF(coins, h.UploadDir, username, h.logger)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate catalog"})
		return
	}

	filename := fmt.Sprintf("coin-catalog-%s.pdf", time.Now().Format("2006-01-02"))
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	if err := pdf.Output(c.Writer); err != nil {
		h.logger.Error("pdf", "Failed to write PDF: %v", err)
	}
}

// ImportCollection imports coins from JSON for the current user.
//
//	@Summary		Import collection
//	@Description	Imports coins from a JSON array. Each coin is created fresh with a new ID for the authenticated user.
//	@Tags			User
//	@Accept			json
//	@Produce		json
//	@Param			body	body		[]models.Coin	true	"Array of coins to import"
//	@Success		200		{object}	ImportResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/user/import [post]
func (h *UserHandler) ImportCollection(c *gin.Context) {
	userID := c.GetUint("userId")

	var coins []models.Coin
	if err := json.NewDecoder(c.Request.Body).Decode(&coins); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format"})
		return
	}

	imported := 0
	for _, coin := range coins {
		coin.ID = 0
		coin.UserID = userID
		// Reset image associations for clean import
		coin.Images = nil
		if err := h.repo.CreateCoin(&coin); err == nil {
			imported++
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Import complete", "imported": imported})
}

// UpdateProfile updates the authenticated user's profile info.
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	logger := h.logger
	userID := c.GetUint("userId")

	var req struct {
		Email             *string `json:"email"`
		Bio               *string `json:"bio"`
		IsPublic          *bool   `json:"isPublic"`
		ZipCode           *string `json:"zipCode"`
		NumisBidsUsername *string `json:"numisBidsUsername"`
		NumisBidsPassword *string `json:"numisBidsPassword"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	var user models.User
	if found, err := h.repo.FindByID(userID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	} else {
		user = *found
	}

	updates := map[string]interface{}{}
	if req.Email != nil {
		email := strings.TrimSpace(*req.Email)
		if email != "" && !strings.Contains(email, "@") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid email format"})
			return
		}
		// Check uniqueness
		if email != "" {
			if _, err := h.repo.FindByEmail(email, userID); err == nil {
				c.JSON(http.StatusConflict, gin.H{"error": "Email already in use"})
				return
			}
		}
		updates["email"] = email
	}
	if req.Bio != nil {
		updates["bio"] = *req.Bio
	}
	if req.ZipCode != nil {
		updates["zip_code"] = strings.TrimSpace(*req.ZipCode)
	}
	if req.NumisBidsUsername != nil {
		updates["numis_bids_username"] = strings.TrimSpace(*req.NumisBidsUsername)
	}
	if req.NumisBidsPassword != nil {
		updates["numis_bids_password"] = *req.NumisBidsPassword
	}
	if req.IsPublic != nil {
		updates["is_public"] = *req.IsPublic
		goingPrivate := !*req.IsPublic && user.IsPublic
		if goingPrivate {
			logger.Info("user", "User %d going private — followers will be removed", userID)
		}
		if len(updates) > 0 {
			if err := h.repo.UpdateProfileWithPrivacy(&user, updates, goingPrivate); err != nil {
				respondError(c, http.StatusInternalServerError, "Failed to update profile", err)
				return
			}
			logger.Info("user", "Profile updated for user %d", userID)
		}
	} else if len(updates) > 0 {
		if err := h.repo.UpdateFields(&user, updates); err != nil {
			respondError(c, http.StatusInternalServerError, "Failed to update profile", err)
			return
		}
		logger.Info("user", "Profile updated for user %d", userID)
	}

	// Reload and return
	if err := h.repo.Reload(&user); err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to reload profile", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":                  user.ID,
		"username":            user.Username,
		"role":                user.Role,
		"email":               user.Email,
		"avatarPath":          user.AvatarPath,
		"isPublic":            user.IsPublic,
		"bio":                 user.Bio,
		"zipCode":             user.ZipCode,
		"numisBidsUsername":   user.NumisBidsUsername,
		"numisBidsConfigured": user.NumisBidsUsername != "" && user.NumisBidsPassword != "",
	})
}

// UploadAvatar uploads a profile avatar image for the authenticated user.
func (h *UserHandler) UploadAvatar(c *gin.Context) {
	logger := h.logger
	userID := c.GetUint("userId")

	file, err := c.FormFile("avatar")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No avatar file provided"})
		return
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowed := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true}
	if !allowed[ext] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File type not allowed. Accepted: .jpg, .jpeg, .png, .gif, .webp"})
		return
	}

	avatarDir := filepath.Join(h.UploadDir, "avatars")
	if err := os.MkdirAll(avatarDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create avatar directory"})
		return
	}

	// Delete old avatar if exists
	var user models.User
	if found, _ := h.repo.FindByID(userID); found != nil {
		user = *found
	}
	if user.AvatarPath != "" {
		os.Remove(filepath.Join(h.UploadDir, user.AvatarPath))
	}

	filename := fmt.Sprintf("user-%d%s", userID, ext)
	filePath := filepath.Join(avatarDir, filename)

	if err := c.SaveUploadedFile(file, filePath); err != nil {
		logger.Error("user", "Failed to save avatar: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save avatar"})
		return
	}

	avatarRelPath := filepath.ToSlash(filepath.Join("avatars", filename))
	if err := h.repo.UpdateField(&user, "avatar_path", avatarRelPath); err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to update avatar", err)
		return
	}

	logger.Info("user", "Avatar uploaded for user %d: %s", userID, avatarRelPath)
	c.JSON(http.StatusOK, gin.H{"avatarPath": avatarRelPath})
}

// DeleteAvatar removes the authenticated user's avatar.
func (h *UserHandler) DeleteAvatar(c *gin.Context) {
	userID := c.GetUint("userId")

	var user models.User
	if found, err := h.repo.FindByID(userID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	} else {
		user = *found
	}

	if user.AvatarPath != "" {
		if err := os.Remove(filepath.Join(h.UploadDir, user.AvatarPath)); err != nil {
			log.Printf("[handler] DeleteAvatar: failed to remove file: %v", err)
		}
		if err := h.repo.UpdateField(&user, "avatar_path", ""); err != nil {
			respondError(c, http.StatusInternalServerError, "Failed to remove avatar", err)
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Avatar removed"})
}
