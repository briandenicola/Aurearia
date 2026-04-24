package handlers

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
	"golang.org/x/net/html"
)

type ImageHandler struct {
	UploadDir string
	repo      *repository.ImageRepository
	svc       *services.ImageService
	logger    *services.Logger
}

func NewImageHandler(uploadDir string, repo *repository.ImageRepository, svc *services.ImageService, logger *services.Logger) *ImageHandler {
	return &ImageHandler{UploadDir: uploadDir, repo: repo, svc: svc, logger: logger}
}

// Upload adds an image to a coin.
//
//	@Summary		Upload a coin image
//	@Description	Upload an image file for a specific coin. Supports setting image type and primary flag.
//	@Tags			Images
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			id			path		int		true	"Coin ID"
//	@Param			image		formData	file	true	"Image file"
//	@Param			imageType	formData	string	false	"Image type"	Enums(obverse, reverse, detail, other)	default(other)
//	@Param			isPrimary	formData	string	false	"Set as primary image"	Enums(true, false)	default(false)
//	@Success		201			{object}	models.CoinImage
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/coins/{id}/images [post]
func (h *ImageHandler) Upload(c *gin.Context) {
	logger := h.logger
	userID := c.GetUint("userId")
	coinID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid coin ID"})
		return
	}

	logger.Debug("images", "Upload request for coin %d (user %d)", coinID, userID)

	file, err := c.FormFile("image")
	if err != nil {
		logger.Warn("images", "No image file in upload request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "No image file provided"})
		return
	}

	logger.Debug("images", "Received file: %s (%d bytes)", file.Filename, file.Size)

	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowedExts := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true}
	if !allowedExts[ext] {
		logger.Warn("images", "Rejected upload with disallowed extension: %s", ext)
		c.JSON(http.StatusBadRequest, gin.H{"error": "File type not allowed. Accepted: .jpg, .jpeg, .png, .gif, .webp"})
		return
	}

	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read uploaded file"})
		return
	}
	defer f.Close()

	fileData, err := io.ReadAll(f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read uploaded file"})
		return
	}

	imageType := c.DefaultPostForm("imageType", "other")
	isPrimary := c.DefaultPostForm("isPrimary", "false") == "true"

	image, err := h.svc.UploadImage(uint(coinID), userID, fileData, ext, imageType, isPrimary)
	if err != nil {
		logger.Error("images", "Upload failed for coin %d: %v", coinID, err)
		switch err {
		case services.ErrCoinNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "Coin not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload image"})
		}
		return
	}

	logger.Info("images", "Uploaded %s image for coin %d: %s", imageType, coinID, image.FilePath)
	c.JSON(http.StatusCreated, image)
}

type base64ImageRequest struct {
	Image         string `json:"image" binding:"required" example:"/9j/4AAQSkZJRgABAQ..."`
	FileExtension string `json:"fileExtension" binding:"required" example:".jpg"`
	ImageType     string `json:"imageType" example:"obverse"`
	IsPrimary     bool   `json:"isPrimary" example:"false"`
}

// UploadBase64 adds an image to a coin from a base64-encoded string.
//
//	@Summary		Upload a coin image (base64)
//	@Description	Upload a raw base64-encoded image for a specific coin. The image field must contain only raw base64 data (no data URI prefix). The fileExtension field (e.g. ".jpg", ".png") tells the API how to save the file. Max 20MB decoded.
//	@Tags			Images
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int					true	"Coin ID"
//	@Param			body	body		base64ImageRequest	true	"Base64 image data"
//	@Success		201		{object}	models.CoinImage
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Security		ApiKeyAuth
//	@Router			/coins/{id}/images/base64 [post]
func (h *ImageHandler) UploadBase64(c *gin.Context) {
	logger := h.logger
	userID := c.GetUint("userId")
	coinID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid coin ID"})
		return
	}

	var req base64ImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	// Validate file extension
	ext := req.FileExtension
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	allowed := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true}
	if !allowed[strings.ToLower(ext)] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "fileExtension must be one of: .jpg, .jpeg, .png, .gif, .webp"})
		return
	}

	imageType := "other"
	if req.ImageType != "" {
		imageType = req.ImageType
	}

	image, err := h.svc.UploadBase64Image(uint(coinID), userID, req.Image, ext, imageType, req.IsPrimary)
	if err != nil {
		logger.Error("images", "Base64 upload failed for coin %d: %v", coinID, err)
		switch err {
		case services.ErrCoinNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "Coin not found"})
		case services.ErrInvalidBase64:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid base64 image data"})
		case services.ErrImageTooLarge:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Image exceeds 20MB limit"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload image"})
		}
		return
	}

	logger.Info("images", "Uploaded base64 %s image for coin %d: %s", imageType, coinID, image.FilePath)
	c.JSON(http.StatusCreated, image)
}

// Delete removes an image from a coin.
//
//	@Summary		Delete a coin image
//	@Description	Deletes an image from a coin. Removes the file from disk and the database record.
//	@Tags			Images
//	@Produce		json
//	@Param			id		path		int	true	"Coin ID"
//	@Param			imageId	path		int	true	"Image ID"
//	@Success		200		{object}	ImageDeletedResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/coins/{id}/images/{imageId} [delete]
func (h *ImageHandler) Delete(c *gin.Context) {
	userID := c.GetUint("userId")
	coinID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid coin ID"})
		return
	}
	imageID, err := strconv.ParseUint(c.Param("imageId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid image ID"})
		return
	}

	_, err = h.svc.DeleteImage(uint(coinID), uint(imageID), userID)
	if err != nil {
		switch err {
		case services.ErrCoinNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "Coin not found"})
		case services.ErrImageNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "Image not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete image"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Image deleted"})
}

// ProxyImage fetches an external image URL and streams it back to the client.
//
//	@Summary		Proxy an external image
//	@Description	Fetches an image from an external URL and streams it to the client. Limited to 20MB. Only http/https URLs with image content types are allowed.
//	@Tags			Images
//	@Produce		image/*
//	@Param			url	query	string	true	"External image URL"
//	@Success		200	"Image binary data"
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		502	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/proxy-image [get]
func (h *ImageHandler) ProxyImage(c *gin.Context) {
	logger := h.logger

	imageURL := c.Query("url")
	if imageURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "url parameter is required"})
		return
	}

	if !strings.HasPrefix(imageURL, "http://") && !strings.HasPrefix(imageURL, "https://") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "URL must start with http:// or https://"})
		return
	}

	logger.Debug("images", "Proxying image from %s", imageURL)

	req, err := http.NewRequest("GET", imageURL, nil)
	if err != nil {
		logger.Warn("images", "Failed to build proxy request for %s: %v", imageURL, err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to build request"})
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "image/*, */*")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Warn("images", "Failed to fetch image from %s: %v", imageURL, err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to fetch image"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Warn("images", "Proxy image %s returned HTTP %d", imageURL, resp.StatusCode)
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("Remote server returned %d", resp.StatusCode)})
		return
	}

	contentType := resp.Header.Get("Content-Type")
	// Accept image/* and common binary types that may contain images
	isImage := strings.HasPrefix(contentType, "image/") ||
		contentType == "application/octet-stream" ||
		contentType == "binary/octet-stream" ||
		contentType == ""
	if !isImage {
		logger.Warn("images", "Proxy image %s has non-image content-type: %s", imageURL, contentType)
		c.JSON(http.StatusBadRequest, gin.H{"error": "URL does not point to an image"})
		return
	}

	// If content type is ambiguous, try to detect from URL extension
	if !strings.HasPrefix(contentType, "image/") {
		lower := strings.ToLower(imageURL)
		switch {
		case strings.Contains(lower, ".jpg"), strings.Contains(lower, ".jpeg"):
			contentType = "image/jpeg"
		case strings.Contains(lower, ".png"):
			contentType = "image/png"
		case strings.Contains(lower, ".webp"):
			contentType = "image/webp"
		case strings.Contains(lower, ".gif"):
			contentType = "image/gif"
		default:
			contentType = "image/jpeg"
		}
	}

	// Limit to 20MB
	const maxSize = 20 * 1024 * 1024
	c.Header("Content-Type", contentType)
	c.Status(http.StatusOK)
	io.Copy(c.Writer, io.LimitReader(resp.Body, maxSize))
}

// ScrapeImage fetches a web page and extracts the primary image URL from meta tags.
//
//	@Summary		Scrape image URL from a web page
//	@Description	Fetches a web page and extracts image URL from og:image, twitter:image, or other meta tags. Useful as a fallback when direct image URLs are unavailable.
//	@Tags			Images
//	@Produce		json
//	@Param			url	query		string	true	"Web page URL to scrape"
//	@Success		200	{object}	map[string]string
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		502	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/scrape-image [get]
func (h *ImageHandler) ScrapeImage(c *gin.Context) {
	logger := h.logger

	pageURL := c.Query("url")
	if pageURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "url parameter is required"})
		return
	}

	if !strings.HasPrefix(pageURL, "http://") && !strings.HasPrefix(pageURL, "https://") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "URL must start with http:// or https://"})
		return
	}

	logger.Debug("images", "Scraping image from page %s", pageURL)

	req, err := http.NewRequest("GET", pageURL, nil)
	if err != nil {
		logger.Warn("images", "Failed to build scrape request for %s: %v", pageURL, err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to build request"})
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Warn("images", "Failed to fetch page %s: %v", pageURL, err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to fetch page"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Warn("images", "Scrape page %s returned HTTP %d", pageURL, resp.StatusCode)
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("Remote server returned %d", resp.StatusCode)})
		return
	}

	// Limit HTML reading to 2MB
	limitedBody := io.LimitReader(resp.Body, 2*1024*1024)
	doc, err := html.Parse(limitedBody)
	if err != nil {
		logger.Warn("images", "Failed to parse HTML from %s: %v", pageURL, err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to parse page HTML"})
		return
	}

	imageURL := extractImageFromHTML(doc)
	if imageURL == "" {
		logger.Info("images", "No og:image or meta image found on page %s", pageURL)
		c.JSON(http.StatusOK, gin.H{"imageUrl": ""})
		return
	}

	// Resolve relative URLs
	if strings.HasPrefix(imageURL, "//") {
		imageURL = "https:" + imageURL
	} else if strings.HasPrefix(imageURL, "/") {
		// Extract base URL from page URL
		parts := strings.SplitN(pageURL, "://", 2)
		if len(parts) == 2 {
			slashIdx := strings.Index(parts[1], "/")
			if slashIdx > 0 {
				imageURL = parts[0] + "://" + parts[1][:slashIdx] + imageURL
			} else {
				imageURL = parts[0] + "://" + parts[1] + imageURL
			}
		}
	}

	logger.Info("images", "Scraped image URL from %s: %s", pageURL, imageURL)
	c.JSON(http.StatusOK, gin.H{"imageUrl": imageURL})
}

// extractImageFromHTML walks the HTML tree looking for image URLs in meta tags.
// Priority: og:image > twitter:image > link[rel=image_src] > first large <img>.
func extractImageFromHTML(doc *html.Node) string {
	var ogImage, twitterImage, linkImage, firstImg string

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "meta":
				var property, name, content string
				for _, a := range n.Attr {
					switch a.Key {
					case "property":
						property = strings.ToLower(a.Val)
					case "name":
						name = strings.ToLower(a.Val)
					case "content":
						content = a.Val
					}
				}
				if content != "" {
					if property == "og:image" || property == "og:image:url" {
						ogImage = content
					} else if name == "twitter:image" || property == "twitter:image" {
						twitterImage = content
					}
				}
			case "link":
				var rel, href string
				for _, a := range n.Attr {
					switch a.Key {
					case "rel":
						rel = strings.ToLower(a.Val)
					case "href":
						href = a.Val
					}
				}
				if rel == "image_src" && href != "" {
					linkImage = href
				}
			case "img":
				if firstImg == "" {
					var src string
					var width, height int
					for _, a := range n.Attr {
						switch a.Key {
						case "src":
							src = a.Val
						case "data-src":
							if src == "" {
								src = a.Val
							}
						case "width":
							w, _ := strconv.Atoi(a.Val)
							width = w
						case "height":
							h, _ := strconv.Atoi(a.Val)
							height = h
						}
					}
					// Only use img tags that appear to be content images (not tiny icons)
					if src != "" && (width >= 100 || height >= 100 || (width == 0 && height == 0)) {
						lower := strings.ToLower(src)
						isIcon := strings.Contains(lower, "icon") ||
							strings.Contains(lower, "logo") ||
							strings.Contains(lower, "favicon") ||
							strings.Contains(lower, "sprite") ||
							strings.Contains(lower, "pixel") ||
							strings.Contains(lower, "spacer")
						if !isIcon {
							firstImg = src
						}
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	if ogImage != "" {
		return ogImage
	}
	if twitterImage != "" {
		return twitterImage
	}
	if linkImage != "" {
		return linkImage
	}
	return firstImg
}
