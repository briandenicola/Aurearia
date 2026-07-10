package services

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/jpeg"
	"testing"
)

// makeResizeTestJPEG creates a solid-colour JPEG image of the given dimensions.
func makeResizeTestJPEG(t *testing.T, w, h int, c color.RGBA) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatalf("makeResizeTestJPEG: encode failed: %v", err)
	}
	return buf.Bytes()
}

func TestResizeForAnalysis_SmallImageReturnedUnchanged(t *testing.T) {
	data := makeResizeTestJPEG(t, 100, 100, color.RGBA{R: 200, G: 150, B: 50, A: 255})
	if len(data) >= maxAnalysisRawBytes {
		t.Fatalf("test prerequisite: image (%d bytes) >= budget (%d)", len(data), maxAnalysisRawBytes)
	}
	got, err := resizeForAnalysis(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Error("small image should be returned unchanged")
	}
}

func TestResizeForAnalysis_InvalidOversizedBytesReturnsError(t *testing.T) {
	// Non-image bytes larger than the raw byte budget should return a decode error.
	invalid := bytes.Repeat([]byte{0xAB}, maxAnalysisRawBytes+1)
	_, err := resizeForAnalysis(invalid)
	if err == nil {
		t.Error("expected decode error for invalid oversized bytes, got nil")
	}
}

func TestResizeForAnalysis_ResultWithinBase64Budget(t *testing.T) {
	data := makeResizeTestJPEG(t, 800, 600, color.RGBA{R: 128, G: 64, B: 32, A: 255})
	got, err := resizeForAnalysis(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	encoded := base64.StdEncoding.EncodeToString(got)
	if len(encoded) > maxAnalysisBase64Chars {
		t.Errorf("base64 length %d exceeds limit %d", len(encoded), maxAnalysisBase64Chars)
	}
}

func TestResizeForAnalysis_Constants(t *testing.T) {
	if maxAnalysisBase64Chars != 12_000_000 {
		t.Errorf("maxAnalysisBase64Chars = %d, want 12_000_000", maxAnalysisBase64Chars)
	}
	if maxAnalysisRawBytes != 9_000_000 {
		t.Errorf("maxAnalysisRawBytes = %d, want 9_000_000", maxAnalysisRawBytes)
	}
	if analysisMaxDim != 1920 {
		t.Errorf("analysisMaxDim = %d, want 1920", analysisMaxDim)
	}
}
