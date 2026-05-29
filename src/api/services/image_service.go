package services

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

var (
	ErrCoinNotFound      = errors.New("coin not found")
	ErrImageNotFound     = errors.New("image not found")
	ErrInvalidBase64     = errors.New("invalid base64 image data")
	ErrInvalidImageType  = errors.New("invalid image type")
	ErrInvalidFileExt    = errors.New("invalid file extension")
	ErrImageTooLarge     = errors.New("image exceeds 20MB limit")
	ErrDirectoryCreation = errors.New("failed to create upload directory")
	ErrFileSave          = errors.New("failed to save image")
	ErrImageRecord       = errors.New("failed to save image record")
)

// ImageService handles image upload orchestration and file management.
type ImageService struct {
	repo      *repository.ImageRepository
	uploadDir string
}

// NewImageService creates a new ImageService.
func NewImageService(repo *repository.ImageRepository, uploadDir string) *ImageService {
	return &ImageService{repo: repo, uploadDir: uploadDir}
}

// UploadImage saves image file data to disk and creates a DB record.
func (s *ImageService) UploadImage(coinID, userID uint, fileData []byte, ext string, imageType string, isPrimary bool) (*models.CoinImage, error) {
	if _, err := s.repo.FindCoinByOwner(coinID, userID); err != nil {
		return nil, ErrCoinNotFound
	}
	safeExt, err := normalizeImageExt(ext)
	if err != nil {
		return nil, err
	}
	safeImageType, err := normalizeImageType(imageType)
	if err != nil {
		return nil, err
	}

	coinDir := filepath.Join(s.uploadDir, fmt.Sprintf("coin-%d", coinID))
	if err := os.MkdirAll(coinDir, 0755); err != nil {
		return nil, ErrDirectoryCreation
	}

	filename := fmt.Sprintf("%d-%s%s", time.Now().UnixNano(), safeImageType, safeExt)
	filePath := filepath.Clean(filepath.Join(coinDir, filename))
	coinDirClean := filepath.Clean(coinDir)
	rel, err := filepath.Rel(coinDirClean, filePath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return nil, ErrFileSave
	}

	if err := os.WriteFile(filePath, fileData, 0644); err != nil {
		return nil, ErrFileSave
	}

	image := models.CoinImage{
		CoinID:    coinID,
		FilePath:  filepath.ToSlash(filepath.Join(fmt.Sprintf("coin-%d", coinID), filename)),
		ImageType: safeImageType,
		IsPrimary: isPrimary,
	}

	var dbErr error
	if isPrimary {
		dbErr = s.repo.SetPrimaryAndCreate(coinID, &image)
	} else {
		dbErr = s.repo.CreateImage(&image)
	}
	if dbErr != nil {
		// Clean up the file written to disk to prevent orphans
		os.Remove(filePath)
		return nil, ErrImageRecord
	}

	return &image, nil
}

// UploadBase64Image decodes a base64 string and saves it as an image.
func (s *ImageService) UploadBase64Image(coinID, userID uint, base64Data string, ext string, imageType string, isPrimary bool) (*models.CoinImage, error) {
	decoded, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		decoded, err = base64.RawStdEncoding.DecodeString(base64Data)
		if err != nil {
			return nil, ErrInvalidBase64
		}
	}

	const maxSize = 20 * 1024 * 1024
	if len(decoded) > maxSize {
		return nil, ErrImageTooLarge
	}

	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}

	return s.UploadImage(coinID, userID, decoded, ext, imageType, isPrimary)
}

func normalizeImageType(imageType string) (models.ImageType, error) {
	switch strings.ToLower(strings.TrimSpace(imageType)) {
	case string(models.ImageTypeObverse):
		return models.ImageTypeObverse, nil
	case string(models.ImageTypeReverse):
		return models.ImageTypeReverse, nil
	case string(models.ImageTypeDetail):
		return models.ImageTypeDetail, nil
	case "", string(models.ImageTypeOther):
		return models.ImageTypeOther, nil
	default:
		return "", ErrInvalidImageType
	}
}

func normalizeImageExt(ext string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(ext))
	if !strings.HasPrefix(normalized, ".") {
		normalized = "." + normalized
	}

	switch normalized {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		return normalized, nil
	default:
		return "", ErrInvalidFileExt
	}
}

// DeleteImage removes an image file from disk and its DB record.
// Returns the deleted file path.
func (s *ImageService) DeleteImage(coinID, imageID, userID uint) (string, error) {
	if _, err := s.repo.FindCoinByOwner(coinID, userID); err != nil {
		return "", ErrCoinNotFound
	}

	image, err := s.repo.FindImage(imageID, coinID)
	if err != nil {
		return "", ErrImageNotFound
	}

	fullPath := filepath.Join(s.uploadDir, image.FilePath)
	os.Remove(fullPath)

	s.repo.DeleteImage(image)
	return image.FilePath, nil
}
