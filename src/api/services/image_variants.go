package services

import (
	"bytes"
	"errors"
	"image"
	_ "image/gif" // GIF decoder
	"image/jpeg"
	_ "image/jpeg" // JPEG decoder
	_ "image/png"  // PNG decoder
	"math"
	"os"
	"path/filepath"
	"strings"
)

// variantThumbMaxDim is the maximum pixel dimension for thumbnail variants.
const variantThumbMaxDim = 300

// variantThumbQuality is the JPEG quality for thumbnail variants.
const variantThumbQuality = 70

// variantMediumMaxDim is the maximum pixel dimension for medium variants.
const variantMediumMaxDim = 800

// variantMediumQuality is the JPEG quality for medium variants.
const variantMediumQuality = 82

// VariantSize identifies an image size variant.
type VariantSize string

const (
	VariantSizeThumb  VariantSize = "thumb"
	VariantSizeMedium VariantSize = "medium"
	VariantSizeFull   VariantSize = "full"
)

// generateImageVariants returns thumbnail and medium JPEG variants of data.
// Both are generated independently; an error from one does not prevent the other.
// Returns an error only if image decoding fails (i.e. data is not a supported image format).
func generateImageVariants(data []byte) (thumb, medium []byte, err error) {
	thumb, err = generateVariant(data, variantThumbMaxDim, variantThumbQuality)
	if err != nil {
		return nil, nil, err
	}
	medium, err = generateVariant(data, variantMediumMaxDim, variantMediumQuality)
	if err != nil {
		return nil, nil, err
	}
	return thumb, medium, nil
}

// generateVariant scales image data to fit within maxDim×maxDim and re-encodes as JPEG.
// If the image already fits within maxDim in both dimensions it is still re-encoded at quality
// to reduce file size for smaller screens.
func generateVariant(data []byte, maxDim int, quality int) ([]byte, error) {
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	if w > maxDim || h > maxDim {
		ratio := math.Min(float64(maxDim)/float64(w), float64(maxDim)/float64(h))
		newW := int(math.Round(float64(w) * ratio))
		newH := int(math.Round(float64(h) * ratio))
		if newW < 1 {
			newW = 1
		}
		if newH < 1 {
			newH = 1
		}

		dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
		for y := 0; y < newH; y++ {
			for x := 0; x < newW; x++ {
				srcX := int(float64(x) / ratio)
				srcY := int(float64(y) / ratio)
				dst.Set(x, y, src.At(srcX, srcY))
			}
		}
		src = dst
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, src, &jpeg.Options{Quality: quality}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// maxVariantSourceBytes bounds the original an existing photo's variants are
// made from, so a stray huge file cannot exhaust memory while decoding.
const maxVariantSourceBytes = 40 * 1024 * 1024

// ErrVariantSourceTooLarge reports an original too large to make variants from.
var ErrVariantSourceTooLarge = errors.New("image too large to resize")

// variantNames returns the thumb and medium variant paths for an upload path:
// coin-7/123-obverse.png -> coin-7/123-obverse_thumb.jpg, ..._medium.jpg.
func variantNames(relPath string) (thumb, medium string) {
	base := strings.TrimSuffix(relPath, filepath.Ext(relPath))
	return base + variantSuffix(string(VariantSizeThumb)), base + variantSuffix(string(VariantSizeMedium))
}

// ensureImageVariants makes the thumb and medium variants of an upload when
// either is missing. Photos uploaded before variants existed, and Quick
// Capture photos, have none, and inline transports (the MCP coin viewer)
// cannot show their originals. made is false when both already existed. The
// upload directory is opened as an os.Root, so a stored path cannot reach
// outside it.
func ensureImageVariants(uploadDir, rawPath string) (made bool, err error) {
	relPath, err := normalizeUploadPath(rawPath)
	if err != nil {
		return false, err
	}
	root, err := os.OpenRoot(uploadDir)
	if err != nil {
		return false, err
	}
	defer root.Close()

	thumb, medium := variantNames(filepath.FromSlash(relPath))
	_, thumbErr := root.Stat(thumb)
	_, mediumErr := root.Stat(medium)
	if thumbErr == nil && mediumErr == nil {
		return false, nil
	}

	info, err := root.Stat(filepath.FromSlash(relPath))
	if err != nil {
		return false, err
	}
	if info.IsDir() {
		return false, ErrMediaNotFound
	}
	if info.Size() > maxVariantSourceBytes {
		return false, ErrVariantSourceTooLarge
	}
	data, err := root.ReadFile(filepath.FromSlash(relPath))
	if err != nil {
		return false, err
	}
	thumbData, mediumData, err := generateImageVariants(data)
	if err != nil {
		return false, err
	}
	if thumbErr != nil {
		if err := root.WriteFile(thumb, thumbData, 0644); err != nil {
			return false, err
		}
	}
	if mediumErr != nil {
		if err := root.WriteFile(medium, mediumData, 0644); err != nil {
			return false, err
		}
	}
	return true, nil
}

// EnsureVariants makes any missing size variants of one stored photo.
func (s *ImageService) EnsureVariants(filePath string) (bool, error) {
	return ensureImageVariants(s.uploadDir, filePath)
}
