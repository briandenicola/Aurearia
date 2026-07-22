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

func makeResizeTestPatternJPEG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, color.RGBA{
				R: uint8((x*37 + y*17) % 256),
				G: uint8((x*13 + y*29) % 256),
				B: uint8((x*7 + y*11) % 256),
				A: 255,
			})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatalf("makeResizeTestPatternJPEG: encode failed: %v", err)
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

func TestResizeForAnalysis_RawBytesUnderOldBudgetButBase64OverProviderLimit(t *testing.T) {
	data := makeResizeTestPatternJPEG(t, 2800, 2800)
	const oldRawBudget = 12_000_000 * 3 / 4
	if len(data) >= oldRawBudget {
		t.Fatalf("test prerequisite: image (%d bytes) >= old budget (%d)", len(data), oldRawBudget)
	}
	if encodedBase64Len(len(data)) <= maxAnalysisBase64Chars {
		t.Fatalf("test prerequisite: base64 length %d <= provider limit %d", encodedBase64Len(len(data)), maxAnalysisBase64Chars)
	}

	got, err := resizeForAnalysis(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if encodedBase64Len(len(got)) > maxAnalysisBase64Chars {
		t.Errorf("base64 length %d exceeds limit %d", encodedBase64Len(len(got)), maxAnalysisBase64Chars)
	}
	if bytes.Equal(got, data) {
		t.Error("image over provider base64 limit should be resized")
	}
}

func TestResizeForAnalysis_ResultWithinBase64Budget(t *testing.T) {
	data := makeResizeTestPatternJPEG(t, 3000, 3000)
	got, err := resizeForAnalysis(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(base64.StdEncoding.EncodeToString(got)) > maxAnalysisBase64Chars {
		t.Errorf("base64 length %d exceeds limit %d", encodedBase64Len(len(got)), maxAnalysisBase64Chars)
	}
}

func TestResizeForAnalysis_Constants(t *testing.T) {
	if maxAnalysisBase64Chars != 10*1024*1024 {
		t.Errorf("maxAnalysisBase64Chars = %d, want %d", maxAnalysisBase64Chars, 10*1024*1024)
	}
	if maxAnalysisRawBytes != 7_864_320 {
		t.Errorf("maxAnalysisRawBytes = %d, want 7_864_320", maxAnalysisRawBytes)
	}
	if analysisMaxDim != 1920 {
		t.Errorf("analysisMaxDim = %d, want 1920", analysisMaxDim)
	}
}
