package services

import (
	"bytes"
	"image"
	_ "image/gif"  // GIF decoder
	_ "image/jpeg" // JPEG decoder
	"image/jpeg"
	_ "image/png" // PNG decoder
	"math"
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
