package services

import (
	"bytes"
	"image"
	_ "image/jpeg" // JPEG decoder
	"image/jpeg"
	_ "image/png" // PNG decoder
	"math"
)

// maxAnalysisBase64Chars mirrors the BoundedImageBase64 constraint in the Python agent.
const maxAnalysisBase64Chars = 12_000_000

// maxAnalysisRawBytes is the raw byte budget that fits inside maxAnalysisBase64Chars.
// base64 encodes 3 bytes as 4 chars: 12_000_000 * 3 / 4 = 9_000_000.
const maxAnalysisRawBytes = maxAnalysisBase64Chars * 3 / 4

// analysisMaxDim is the maximum pixel dimension (width or height) used when resizing.
const analysisMaxDim = 1920

// analysisJPEGQuality is the JPEG quality applied when re-encoding a resized image.
const analysisJPEGQuality = 85

// resizeForAnalysis returns image bytes safe for AI analysis.
// If the raw byte length is already within the base64 character budget it returns data unchanged.
// Otherwise the image is decoded, scaled down to fit within analysisMaxDim×analysisMaxDim
// (preserving aspect ratio), and re-encoded as JPEG at analysisJPEGQuality.
func resizeForAnalysis(data []byte) ([]byte, error) {
	if len(data) <= maxAnalysisRawBytes {
		return data, nil
	}

	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	if w > analysisMaxDim || h > analysisMaxDim {
		ratio := math.Min(float64(analysisMaxDim)/float64(w), float64(analysisMaxDim)/float64(h))
		newW := int(math.Round(float64(w) * ratio))
		newH := int(math.Round(float64(h) * ratio))

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
	if err := jpeg.Encode(&buf, src, &jpeg.Options{Quality: analysisJPEGQuality}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
