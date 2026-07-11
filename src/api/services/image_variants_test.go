package services

import (
	"bytes"
	"image/color"
	"image/jpeg"
	"testing"

	"image"
)

// makeVariantTestJPEG creates a solid-colour JPEG image of the given dimensions.
func makeVariantTestJPEG(t *testing.T, w, h int, c color.RGBA) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatalf("makeVariantTestJPEG: encode failed: %v", err)
	}
	return buf.Bytes()
}

func TestGenerateVariant_DownscalesLargeImage(t *testing.T) {
	data := makeVariantTestJPEG(t, 1200, 900, color.RGBA{R: 100, G: 150, B: 200, A: 255})
	out, err := generateVariant(data, 300, 70)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	img, _, err := image.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("output is not a valid image: %v", err)
	}
	b := img.Bounds()
	if b.Dx() > 300 || b.Dy() > 300 {
		t.Errorf("variant %dx%d exceeds 300px limit", b.Dx(), b.Dy())
	}
	// Aspect ratio preserved within rounding
	expectedW := 300
	expectedH := 225
	if b.Dx() != expectedW || b.Dy() != expectedH {
		t.Errorf("expected %dx%d, got %dx%d", expectedW, expectedH, b.Dx(), b.Dy())
	}
}

func TestGenerateVariant_SmallImageReEncodedWithinBudget(t *testing.T) {
	data := makeVariantTestJPEG(t, 100, 100, color.RGBA{R: 50, G: 50, B: 50, A: 255})
	out, err := generateVariant(data, 300, 70)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	img, _, err := image.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("output is not a valid image: %v", err)
	}
	b := img.Bounds()
	if b.Dx() != 100 || b.Dy() != 100 {
		t.Errorf("expected 100x100, got %dx%d", b.Dx(), b.Dy())
	}
}

func TestGenerateVariant_InvalidDataReturnsError(t *testing.T) {
	_, err := generateVariant([]byte("not an image"), 300, 70)
	if err == nil {
		t.Error("expected error for invalid image data, got nil")
	}
}

func TestGenerateImageVariants_BothVariantsProduced(t *testing.T) {
	data := makeVariantTestJPEG(t, 2000, 1500, color.RGBA{R: 200, G: 100, B: 50, A: 255})
	thumb, medium, err := generateImageVariants(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tImg, _, err := image.Decode(bytes.NewReader(thumb))
	if err != nil {
		t.Fatalf("thumb is not valid image: %v", err)
	}
	tb := tImg.Bounds()
	if tb.Dx() > variantThumbMaxDim || tb.Dy() > variantThumbMaxDim {
		t.Errorf("thumb %dx%d exceeds %dpx limit", tb.Dx(), tb.Dy(), variantThumbMaxDim)
	}

	mImg, _, err := image.Decode(bytes.NewReader(medium))
	if err != nil {
		t.Fatalf("medium is not valid image: %v", err)
	}
	mb := mImg.Bounds()
	if mb.Dx() > variantMediumMaxDim || mb.Dy() > variantMediumMaxDim {
		t.Errorf("medium %dx%d exceeds %dpx limit", mb.Dx(), mb.Dy(), variantMediumMaxDim)
	}

	if len(thumb) >= len(medium) {
		t.Errorf("expected thumb (%d bytes) < medium (%d bytes)", len(thumb), len(medium))
	}
}

func TestGenerateImageVariants_InvalidDataReturnsError(t *testing.T) {
	_, _, err := generateImageVariants([]byte("garbage"))
	if err == nil {
		t.Error("expected error for invalid image data")
	}
}

func TestVariantConstants(t *testing.T) {
	if variantThumbMaxDim <= 0 {
		t.Error("variantThumbMaxDim must be positive")
	}
	if variantMediumMaxDim <= variantThumbMaxDim {
		t.Error("variantMediumMaxDim must exceed variantThumbMaxDim")
	}
	if variantThumbQuality < 1 || variantThumbQuality > 100 {
		t.Errorf("variantThumbQuality %d out of range", variantThumbQuality)
	}
	if variantMediumQuality < 1 || variantMediumQuality > 100 {
		t.Errorf("variantMediumQuality %d out of range", variantMediumQuality)
	}
}
