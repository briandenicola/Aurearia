package services

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png" // PNG decoder
	"math"
)

// maxAnalysisBase64Chars mirrors Anthropic's per-image base64 limit and the
// BoundedImageBase64 constraint in the Python agent.
const maxAnalysisBase64Chars = 10 * 1024 * 1024

// maxAnalysisRawBytes is the raw byte budget that fits inside maxAnalysisBase64Chars.
// base64 encodes complete 3-byte groups as 4 chars.
const maxAnalysisRawBytes = (maxAnalysisBase64Chars / 4) * 3

// analysisMaxDim is the maximum pixel dimension (width or height) used when resizing.
const analysisMaxDim = 1920

// analysisMinDim is the smallest maximum dimension attempted before giving up.
const analysisMinDim = 512

// analysisJPEGQuality is the JPEG quality applied when re-encoding a resized image.
const analysisJPEGQuality = 85

// analysisMinJPEGQuality is the lowest JPEG quality attempted before reducing dimensions further.
const analysisMinJPEGQuality = 55

// resizeForAnalysis returns image bytes safe for AI analysis.
// If the encoded byte length is already within the base64 character budget it returns data unchanged.
// Otherwise the image is decoded, scaled down to fit within analysisMaxDim×analysisMaxDim
// (preserving aspect ratio), and re-encoded as JPEG until it fits the provider limit.
func resizeForAnalysis(data []byte) ([]byte, error) {
	if encodedBase64Len(len(data)) <= maxAnalysisBase64Chars {
		return data, nil
	}

	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	maxDim := analysisMaxDim
	quality := analysisJPEGQuality
	for {
		resized := resizeToMaxDimension(src, maxDim)
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, resized, &jpeg.Options{Quality: quality}); err != nil {
			return nil, err
		}
		if encodedBase64Len(buf.Len()) <= maxAnalysisBase64Chars {
			return buf.Bytes(), nil
		}

		if quality > analysisMinJPEGQuality {
			quality -= 10
			if quality < analysisMinJPEGQuality {
				quality = analysisMinJPEGQuality
			}
			continue
		}

		if maxDim <= analysisMinDim {
			return nil, fmt.Errorf("resized image exceeds AI image size limit")
		}
		maxDim = int(math.Round(float64(maxDim) * 0.8))
		if maxDim < analysisMinDim {
			maxDim = analysisMinDim
		}
		quality = analysisJPEGQuality
	}
}

func encodedBase64Len(rawBytes int) int {
	if rawBytes <= 0 {
		return 0
	}
	return ((rawBytes + 2) / 3) * 4
}

func resizeToMaxDimension(src image.Image, maxDim int) image.Image {
	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	if w <= maxDim && h <= maxDim {
		return src
	}

	ratio := math.Min(float64(maxDim)/float64(w), float64(maxDim)/float64(h))
	newW := max(1, int(math.Round(float64(w)*ratio)))
	newH := max(1, int(math.Round(float64(h)*ratio)))

	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	for y := 0; y < newH; y++ {
		for x := 0; x < newW; x++ {
			srcX := bounds.Min.X + min(w-1, int(float64(x)/ratio))
			srcY := bounds.Min.Y + min(h-1, int(float64(y)/ratio))
			dst.Set(x, y, src.At(srcX, srcY))
		}
	}
	return dst
}
