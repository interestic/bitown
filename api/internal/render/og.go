package render

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"

	"github.com/interestic/bitown/internal/citycore"
)

const (
	ogRendererVersion = "og-v1"
	OGWidth           = 1200
	OGHeight          = 630
)

// ogBackdrop is a sky fill so the isometric diamond letterboxes cleanly
// inside the 1.91:1 Open Graph frame.
var ogBackdrop = color.RGBA{R: 74, G: 126, B: 168, A: 255}

// OGEntityTag returns a strong ETag for og.png. It is derived from
// MapEntityTag plus the OG compositor version, so map.png ETags stay unchanged.
func OGEntityTag(city *citycore.City) (string, error) {
	mapTag, err := MapEntityTag(city)
	if err != nil {
		return "", err
	}
	sum := sha256.New()
	_, _ = fmt.Fprintf(sum, "%s/%s", ogRendererVersion, mapTag)
	return `"` + hex.EncodeToString(sum.Sum(nil)[:16]) + `"`, nil
}

// BuildCityOGPNG renders a 1200×630 PNG snapshot of the city map for
// Open Graph cards and GitHub README images.
func BuildCityOGPNG(city *citycore.City) ([]byte, error) {
	srcPNG, err := BuildCityMapPNG(city)
	if err != nil {
		return nil, err
	}
	src, err := png.Decode(bytes.NewReader(srcPNG))
	if err != nil {
		return nil, fmt.Errorf("decode map png: %w", err)
	}
	frame := letterboxNearest(src, OGWidth, OGHeight, ogBackdrop)
	var buf bytes.Buffer
	if err := png.Encode(&buf, frame); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func letterboxNearest(src image.Image, dw, dh int, bg color.Color) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{C: bg}, image.Point{}, draw.Src)

	sb := src.Bounds()
	sw, sh := sb.Dx(), sb.Dy()
	if sw <= 0 || sh <= 0 || dw <= 0 || dh <= 0 {
		return dst
	}

	scaleW := float64(dw) / float64(sw)
	scaleH := float64(dh) / float64(sh)
	scale := scaleW
	if scaleH < scaleW {
		scale = scaleH
	}
	nw := int(float64(sw) * scale)
	nh := int(float64(sh) * scale)
	if nw < 1 {
		nw = 1
	}
	if nh < 1 {
		nh = 1
	}
	ox := (dw - nw) / 2
	oy := (dh - nh) / 2

	for y := 0; y < nh; y++ {
		sy := sb.Min.Y + y*sh/nh
		for x := 0; x < nw; x++ {
			sx := sb.Min.X + x*sw/nw
			dst.Set(ox+x, oy+y, src.At(sx, sy))
		}
	}
	return dst
}
