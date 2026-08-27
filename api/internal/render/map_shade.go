package render

import (
	"image"
	"math"
	"os"
)

const groundShadeVariant = "ground-shade-v1"

func groundShadeEnabled() bool {
	return os.Getenv("BITOWN_MAP_GROUND_SHADE") == "1"
}

// groundShadeIdentity is appended to the map ETag hash when shade is on.
// Empty when off so default mapRendererVersion tags stay byte-identical.
func groundShadeIdentity() string {
	if groundShadeEnabled() {
		return groundShadeVariant
	}
	return ""
}

// applyGroundShade multiplies opaque ground pixels with a fixed NW directional
// gradient. Call after floor + road underlay and before object sprites.
func applyGroundShade(img *image.RGBA) {
	b := img.Bounds()
	denom := float64(mapWidth + mapHeight)
	if denom <= 0 {
		return
	}
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			i := img.PixOffset(x, y)
			a := img.Pix[i+3]
			if a == 0 {
				continue
			}
			// Higher x+y = SE (more shadow); lower x+y = NW (brighter).
			t := float64(x+y) / denom
			factor := 1.15 - 0.35*t
			if factor < 0.75 {
				factor = 0.75
			} else if factor > 1.15 {
				factor = 1.15
			}
			// Subtle warm highlight / cool shadow.
			rBoost := 1.0 + 0.04*(factor-1.0)
			bBoost := 1.0 + 0.04*(1.0-factor)
			img.Pix[i+0] = mulShadeChannel(img.Pix[i+0], factor*rBoost)
			img.Pix[i+1] = mulShadeChannel(img.Pix[i+1], factor)
			img.Pix[i+2] = mulShadeChannel(img.Pix[i+2], factor*bBoost)
		}
	}
}

func mulShadeChannel(c uint8, factor float64) uint8 {
	v := float64(c) * factor
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(math.Round(v))
}
