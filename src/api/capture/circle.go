package capture

import (
	"bytes"
	"image"
	"image/color"
	_ "image/jpeg" // register JPEG decoder
	_ "image/png"  // register PNG decoder
	"image/png"
	"math"
)

// CircleGuide defines the geometry of a circular clipping region as fractions of the source image dimensions.
// CenterXFrac and CenterYFrac specify the center position as fractions (0.0 to 1.0) of width and height.
// DiamFrac specifies the circle diameter as a fraction (0.0 to 1.0) of the source width.
// MaxDiamPx caps the absolute diameter in pixels; 0 means no cap.
type CircleGuide struct {
	CenterXFrac float64
	CenterYFrac float64
	DiamFrac    float64
	MaxDiamPx   float64
}

// DefaultGuide mirrors the frontend viewfinder overlay geometry:
// center at 50% horizontal, 52% vertical; diameter 74% of width, capped at 360px.
var DefaultGuide = CircleGuide{
	CenterXFrac: 0.50,
	CenterYFrac: 0.52,
	DiamFrac:    0.74,
	MaxDiamPx:   360,
}

// resolve computes the absolute center coordinates (cx, cy) and radius (r) in pixels
// for the given image bounds. The diameter is DiamFrac * width, capped at MaxDiamPx if MaxDiamPx > 0.
// The center is positioned at Min + frac * dimension for each axis.
func (g CircleGuide) resolve(b image.Rectangle) (cx, cy, r float64) {
	w, h := float64(b.Dx()), float64(b.Dy())
	diam := g.DiamFrac * w
	if g.MaxDiamPx > 0 && diam > g.MaxDiamPx {
		diam = g.MaxDiamPx
	}
	cx = float64(b.Min.X) + g.CenterXFrac*w
	cy = float64(b.Min.Y) + g.CenterYFrac*h
	r = diam / 2
	return
}

// ClipToCircle crops the source image to the circular region defined by the guide,
// returning a new RGBA image sized to the circle's bounding box (intersected with source bounds).
// Pixels outside the circle are fully transparent. The circle edge is anti-aliased over the
// last 1px ring (alpha linearly decreases from 1.0 at r-1 to 0.0 at r).
// Color channels are premultiplied by the computed alpha to produce correct compositing.
func ClipToCircle(src image.Image, g CircleGuide) *image.RGBA {
	cx, cy, r := g.resolve(src.Bounds())
	x0, y0 := int(math.Floor(cx-r)), int(math.Floor(cy-r))
	x1, y1 := int(math.Ceil(cx+r)), int(math.Ceil(cy+r))
	bb := image.Rect(x0, y0, x1, y1).Intersect(src.Bounds())
	out := image.NewRGBA(image.Rect(0, 0, bb.Dx(), bb.Dy()))
	for y := bb.Min.Y; y < bb.Max.Y; y++ {
		for x := bb.Min.X; x < bb.Max.X; x++ {
			dx := (float64(x) + 0.5) - cx
			dy := (float64(y) + 0.5) - cy
			dist := math.Hypot(dx, dy)
			alpha := 1.0
			if dist > r {
				alpha = 0
			} else if dist > r-1 {
				alpha = r - dist
			}
			if alpha <= 0 {
				continue
			}
			sr, sg, sb, sa := src.At(x, y).RGBA()
			scale := func(c uint32) uint8 { return uint8(uint32(alpha*float64(c)) >> 8) }
			out.SetRGBA(x-bb.Min.X, y-bb.Min.Y, color.RGBA{
				R: scale(sr),
				G: scale(sg),
				B: scale(sb),
				A: uint8(uint32(alpha*float64(sa)+0.5) >> 8),
			})
		}
	}
	return out
}

// ClipBytesToCirclePNG decodes image bytes (JPEG or PNG), applies circular clipping with the given guide,
// and encodes the result as PNG. Returns the PNG bytes or an error if decoding fails.
// If g is the zero value, DefaultGuide is used.
func ClipBytesToCirclePNG(data []byte, g CircleGuide) ([]byte, error) {
	if g == (CircleGuide{}) {
		g = DefaultGuide
	}
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	clipped := ClipToCircle(src, g)
	var buf bytes.Buffer
	if err := png.Encode(&buf, clipped); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
