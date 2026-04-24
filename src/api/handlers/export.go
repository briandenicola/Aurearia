package handlers

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

// writeCollectionZip streams a zip archive containing coins.json and all associated images.
func writeCollectionZip(c *gin.Context, coins []models.Coin, uploadDir string, filename string, logger *services.Logger) {

	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	zw := zip.NewWriter(c.Writer)
	defer zw.Close()

	// Write coins.json
	jsonWriter, err := zw.Create("coins.json")
	if err != nil {
		logger.Error("export", "Failed to create coins.json in zip: %v", err)
		return
	}
	encoder := json.NewEncoder(jsonWriter)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(coins); err != nil {
		logger.Error("export", "Failed to write coins.json: %v", err)
		return
	}

	// Add all image files
	for _, coin := range coins {
		for _, img := range coin.Images {
			diskPath := filepath.Join(uploadDir, filepath.FromSlash(img.FilePath))

			file, err := os.Open(diskPath)
			if err != nil {
				logger.Warn("export", "Skipping missing image %s: %v", img.FilePath, err)
				continue
			}

			zipPath := fmt.Sprintf("images/%s", img.FilePath)
			w, err := zw.Create(zipPath)
			if err != nil {
				file.Close()
				logger.Error("export", "Failed to create zip entry %s: %v", zipPath, err)
				continue
			}

			if _, err := io.Copy(w, file); err != nil {
				logger.Error("export", "Failed to write image %s to zip: %v", zipPath, err)
			}
			file.Close()
		}
	}

	logger.Info("export", "Export complete: %d coins, zip=%s", len(coins), filename)
}
