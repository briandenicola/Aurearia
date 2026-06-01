package capture

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math"
	"testing"
)

func TestCircleGuide_resolve(t *testing.T) {
	tests := []struct {
		name    string
		guide   CircleGuide
		bounds  image.Rectangle
		wantCx  float64
		wantCy  float64
		wantR   float64
	}{
		{
			name:    "1000x1000 image with cap - diameter capped at 360",
			guide:   DefaultGuide,
			bounds:  image.Rect(0, 0, 1000, 1000),
			wantCx:  500,
			wantCy:  520,
			wantR:   180,
		},
		{
			name:    "400x400 image no cap hit - diameter 74% of width",
			guide:   DefaultGuide,
			bounds:  image.Rect(0, 0, 400, 400),
			wantCx:  200,
			wantCy:  208,
			wantR:   148,
		},
		{
			name: "non-zero bounds origin",
			guide: CircleGuide{
				CenterXFrac: 0.5,
				CenterYFrac: 0.5,
				DiamFrac:    0.5,
				MaxDiamPx:   0,
			},
			bounds: image.Rect(100, 100, 300, 300),
			wantCx: 200,
			wantCy: 200,
			wantR:  50,
		},
		{
			name: "no cap",
			guide: CircleGuide{
				CenterXFrac: 0.5,
				CenterYFrac: 0.5,
				DiamFrac:    0.8,
				MaxDiamPx:   0,
			},
			bounds: image.Rect(0, 0, 500, 500),
			wantCx: 250,
			wantCy: 250,
			wantR:  200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cx, cy, r := tt.guide.resolve(tt.bounds)
			if math.Abs(cx-tt.wantCx) > 0.01 {
				t.Errorf("cx = %v, want %v", cx, tt.wantCx)
			}
			if math.Abs(cy-tt.wantCy) > 0.01 {
				t.Errorf("cy = %v, want %v", cy, tt.wantCy)
			}
			if math.Abs(r-tt.wantR) > 0.01 {
				t.Errorf("r = %v, want %v", r, tt.wantR)
			}
		})
	}
}

func TestClipToCircle_outputSize(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 500, 500))
	guide := CircleGuide{
		CenterXFrac: 0.5,
		CenterYFrac: 0.5,
		DiamFrac:    0.6,
		MaxDiamPx:   0,
	}
	result := ClipToCircle(src, guide)

	// diameter = 0.6 * 500 = 300, r = 150
	// bounding box should be 300x300
	if result.Bounds().Dx() != 300 {
		t.Errorf("output width = %d, want 300", result.Bounds().Dx())
	}
	if result.Bounds().Dy() != 300 {
		t.Errorf("output height = %d, want 300", result.Bounds().Dy())
	}
}

func TestClipToCircle_cornersTransparent(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 200, 200))
	// fill with opaque white
	for y := 0; y < 200; y++ {
		for x := 0; x < 200; x++ {
			src.Set(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}

	guide := CircleGuide{
		CenterXFrac: 0.5,
		CenterYFrac: 0.5,
		DiamFrac:    0.8,
		MaxDiamPx:   0,
	}
	result := ClipToCircle(src, guide)

	// corners of the output should be fully transparent
	corners := []image.Point{
		{0, 0},
		{result.Bounds().Dx() - 1, 0},
		{0, result.Bounds().Dy() - 1},
		{result.Bounds().Dx() - 1, result.Bounds().Dy() - 1},
	}

	for _, pt := range corners {
		_, _, _, a := result.At(pt.X, pt.Y).RGBA()
		if a != 0 {
			t.Errorf("corner at (%d, %d) has alpha %d, want 0", pt.X, pt.Y, a>>8)
		}
	}
}

func TestClipToCircle_centerOpaque(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 200, 200))
	// fill with opaque white
	for y := 0; y < 200; y++ {
		for x := 0; x < 200; x++ {
			src.Set(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}

	guide := CircleGuide{
		CenterXFrac: 0.5,
		CenterYFrac: 0.5,
		DiamFrac:    0.8,
		MaxDiamPx:   0,
	}
	result := ClipToCircle(src, guide)

	// center pixel should be fully opaque
	centerX := result.Bounds().Dx() / 2
	centerY := result.Bounds().Dy() / 2
	_, _, _, a := result.At(centerX, centerY).RGBA()
	// alpha is in 0-65535 range from RGBA(), convert to 0-255
	alphaU8 := uint8(a >> 8)
	if alphaU8 != 255 {
		t.Errorf("center pixel alpha = %d, want 255", alphaU8)
	}
}

func TestClipToCircle_antiAliasedEdge(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 200, 200))
	// fill with opaque white
	for y := 0; y < 200; y++ {
		for x := 0; x < 200; x++ {
			src.Set(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}

	guide := CircleGuide{
		CenterXFrac: 0.5,
		CenterYFrac: 0.5,
		DiamFrac:    0.5,
		MaxDiamPx:   0,
	}
	result := ClipToCircle(src, guide)

	// compute center and radius
	cx, cy, r := guide.resolve(src.Bounds())

	// sample points just inside and outside the anti-alias band
	// find a pixel at distance r-0.5 (middle of anti-alias band)
	testX := int(cx + (r-0.5)*math.Cos(0))
	testY := int(cy + (r-0.5)*math.Sin(0))

	// map to output coordinates
	x0, y0 := int(math.Floor(cx-r)), int(math.Floor(cy-r))
	outX := testX - x0
	outY := testY - y0

	if outX >= 0 && outX < result.Bounds().Dx() && outY >= 0 && outY < result.Bounds().Dy() {
		_, _, _, a := result.At(outX, outY).RGBA()
		alphaU8 := uint8(a >> 8)
		// should have partial alpha (not 0, not 255)
		if alphaU8 == 0 || alphaU8 == 255 {
			t.Errorf("anti-alias band pixel alpha = %d, want partial alpha (1-254)", alphaU8)
		}
	}

	// pixel just outside r should be transparent
	testX2 := int(cx + (r+0.1)*math.Cos(math.Pi/4))
	testY2 := int(cy + (r+0.1)*math.Sin(math.Pi/4))
	outX2 := testX2 - x0
	outY2 := testY2 - y0
	if outX2 >= 0 && outX2 < result.Bounds().Dx() && outY2 >= 0 && outY2 < result.Bounds().Dy() {
		_, _, _, a := result.At(outX2, outY2).RGBA()
		if a != 0 {
			t.Errorf("pixel outside r has alpha %d, want 0", a>>8)
		}
	}
}

func TestClipToCircle_boundingBoxIntersect(t *testing.T) {
	// small image, guide circle extends beyond bounds
	src := image.NewRGBA(image.Rect(0, 0, 50, 50))
	for y := 0; y < 50; y++ {
		for x := 0; x < 50; x++ {
			src.Set(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}

	guide := CircleGuide{
		CenterXFrac: 0.5,
		CenterYFrac: 0.5,
		DiamFrac:    2.0, // circle diameter would be 100px, bigger than 50px image
		MaxDiamPx:   0,
	}

	// should not panic
	result := ClipToCircle(src, guide)

	// result should be at most 50x50
	if result.Bounds().Dx() > 50 || result.Bounds().Dy() > 50 {
		t.Errorf("result size %dx%d exceeds source bounds 50x50", result.Bounds().Dx(), result.Bounds().Dy())
	}
}

func TestClipBytesToCirclePNG_JPEG(t *testing.T) {
	// create synthetic opaque JPEG
	src := image.NewRGBA(image.Rect(0, 0, 200, 200))
	for y := 0; y < 200; y++ {
		for x := 0; x < 200; x++ {
			src.Set(x, y, color.RGBA{R: 200, G: 150, B: 100, A: 255})
		}
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, src, nil); err != nil {
		t.Fatal(err)
	}

	guide := CircleGuide{
		CenterXFrac: 0.5,
		CenterYFrac: 0.5,
		DiamFrac:    0.6,
		MaxDiamPx:   0,
	}
	result, err := ClipBytesToCirclePNG(buf.Bytes(), guide)
	if err != nil {
		t.Fatalf("ClipBytesToCirclePNG failed: %v", err)
	}

	// decode result to verify it's valid PNG with transparent corners
	img, err := png.Decode(bytes.NewReader(result))
	if err != nil {
		t.Fatalf("result is not valid PNG: %v", err)
	}

	// check corners are transparent
	rgbaImg, ok := img.(*image.RGBA)
	if !ok {
		// might be NRGBA or similar, check corners anyway
		_, _, _, a := img.At(0, 0).RGBA()
		if a != 0 {
			t.Errorf("corner alpha = %d, want 0", a>>8)
		}
	} else {
		_, _, _, a := rgbaImg.At(0, 0).RGBA()
		if a != 0 {
			t.Errorf("corner alpha = %d, want 0", a>>8)
		}
	}
}

func TestClipBytesToCirclePNG_PNG(t *testing.T) {
	// create synthetic opaque PNG
	src := image.NewRGBA(image.Rect(0, 0, 200, 200))
	for y := 0; y < 200; y++ {
		for x := 0; x < 200; x++ {
			src.Set(x, y, color.RGBA{R: 100, G: 200, B: 150, A: 255})
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, src); err != nil {
		t.Fatal(err)
	}

	guide := CircleGuide{
		CenterXFrac: 0.5,
		CenterYFrac: 0.5,
		DiamFrac:    0.8,
		MaxDiamPx:   0,
	}
	result, err := ClipBytesToCirclePNG(buf.Bytes(), guide)
	if err != nil {
		t.Fatalf("ClipBytesToCirclePNG failed: %v", err)
	}

	// decode result
	img, err := png.Decode(bytes.NewReader(result))
	if err != nil {
		t.Fatalf("result is not valid PNG: %v", err)
	}

	// check corners are transparent
	_, _, _, a := img.At(0, 0).RGBA()
	if a != 0 {
		t.Errorf("corner alpha = %d, want 0", a>>8)
	}
}

func TestClipBytesToCirclePNG_zeroValueGuide(t *testing.T) {
	// create synthetic image
	src := image.NewRGBA(image.Rect(0, 0, 200, 200))
	for y := 0; y < 200; y++ {
		for x := 0; x < 200; x++ {
			src.Set(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, src); err != nil {
		t.Fatal(err)
	}

	// zero value guide should fall back to DefaultGuide
	result, err := ClipBytesToCirclePNG(buf.Bytes(), CircleGuide{})
	if err != nil {
		t.Fatalf("ClipBytesToCirclePNG with zero guide failed: %v", err)
	}

	// verify valid PNG returned
	img, err := png.Decode(bytes.NewReader(result))
	if err != nil {
		t.Fatalf("result is not valid PNG: %v", err)
	}

	if img.Bounds().Dx() == 0 || img.Bounds().Dy() == 0 {
		t.Error("zero-value guide fallback produced empty image")
	}
}

func TestClipBytesToCirclePNG_invalidData(t *testing.T) {
	guide := DefaultGuide
	_, err := ClipBytesToCirclePNG([]byte("not an image"), guide)
	if err == nil {
		t.Error("expected error for invalid image data, got nil")
	}
}
