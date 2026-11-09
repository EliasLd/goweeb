package metadata

import (
	"bytes"
	"hash/fnv"
	"image"
	"image/color"
	"image/png"
	"strings"
	"unicode"

	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// The placeholder canvas matches the ~2:3 portrait of manga covers;
// the small bitmap font is magnified nearest-neighbor so the
// initials stay readable at thumbnail size.
const (
	placeholderWidth   = 600
	placeholderHeight  = 900
	placeholderScale   = 10
	placeholderPadding = 40
)

// CoverPlaceholderPNG renders the cover consumers show when no real
// one matched: a deterministic vertical gradient seeded by the
// title, with the title's initials drawn over it. Lookups never
// persist a placeholder — it is produced on demand at render time,
// so a later real cover can still win.
func CoverPlaceholderPNG(title string) ([]byte, error) {
	canvas := image.NewRGBA(
		image.Rect(0, 0, placeholderWidth, placeholderHeight),
	)

	paintGradient(
		canvas,
		gradientColors(title),
	)

	drawInitials(
		canvas,
		initialsFor(title),
	)

	var buffer bytes.Buffer

	if err := png.Encode(&buffer, canvas); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

// gradientColors hashes the title into two pole colors: the pair is
// deterministic and the tones stay mid-light so initials remain
// readable on every possible hue.
func gradientColors(title string) [2]color.RGBA {
	return [2]color.RGBA{
		tone(hashTitle(title, "top")),
		tone(hashTitle(title, "bottom")),
	}
}

func hashTitle(title string, flavor string) uint32 {
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(title))
	_, _ = hash.Write([]byte{':'})
	_, _ = hash.Write([]byte(flavor))

	return hash.Sum32()
}

// tone maps a hash to a color kept away from pure black and pure
// white, so contrast works on both ends of the gradient.
func tone(seed uint32) color.RGBA {
	return color.RGBA{
		R: byte(110 + seed&0x7f),
		G: byte(110 + (seed>>7)&0x5f),
		B: byte(130 + (seed>>14)&0x6f),
		A: 0xFF,
	}
}

// paintGradient fills the canvas vertically between the two poles.
func paintGradient(
	canvas *image.RGBA,
	poles [2]color.RGBA,
) {
	bounds := canvas.Bounds()

	height := bounds.Dy()
	if height <= 1 {
		fillUniform(canvas, poles[0])

		return
	}

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		progress := float64(y-bounds.Min.Y) / float64(height-1)

		blend := color.RGBA{
			R: lerpChannel(poles[0].R, poles[1].R, progress),
			G: lerpChannel(poles[0].G, poles[1].G, progress),
			B: lerpChannel(poles[0].B, poles[1].B, progress),
			A: 0xFF,
		}

		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			canvas.SetRGBA(x, y, blend)
		}
	}
}

func lerpChannel(from, to byte, progress float64) byte {
	return byte(
		float64(from) + (float64(to)-float64(from))*progress,
	)
}

func fillUniform(canvas *image.RGBA, uniform color.RGBA) {
	bounds := canvas.Bounds()

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			canvas.SetRGBA(x, y, uniform)
		}
	}
}

// initialsFor takes the first letter of up to two words, uppercased;
// a title without letters falls back to a fixed glyph.
func initialsFor(title string) string {
	words := strings.FieldsFunc(
		strings.TrimSpace(title),
		func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsDigit(r)
		},
	)

	var letters []rune

	for _, word := range words {
		for _, r := range word {
			letters = append(letters, unicode.ToUpper(r))

			break
		}

		if len(letters) == 2 {
			break
		}
	}

	if len(letters) == 0 {
		return "#"
	}

	return string(letters)
}

// drawInitials renders the letters over the gradient, centered
// above the middle. The bitmap font renders small first, then
// nearest-neighbor magnifies it — blocky but readable, matching the
// placeholder idea.
func drawInitials(
	canvas *image.RGBA,
	initials string,
) {
	face := basicfont.Face7x13

	textWidth := face.Advance * len(initials) * placeholderScale
	textHeight := face.Height * placeholderScale

	targetRect := image.Rect(
		(placeholderWidth-textWidth)/2,
		(placeholderHeight-textHeight)/2-placeholderPadding,
		(placeholderWidth+textWidth)/2,
		(placeholderHeight-textHeight)/2+textHeight-placeholderPadding,
	)

	bitmap := image.NewNRGBA(
		image.Rect(
			0,
			0,
			face.Advance*len(initials),
			face.Height,
		),
	)

	drawer := font.Drawer{
		Dst:  bitmap,
		Src:  &image.Uniform{C: color.RGBA{R: 255, G: 255, B: 255, A: 0xEE}},
		Face: face,
		Dot:  fixed.P(0, face.Height-face.Descent),
	}

	drawer.DrawString(initials)

	draw.NearestNeighbor.Scale(
		canvas,
		targetRect,
		bitmap,
		bitmap.Bounds(),
		draw.Over,
		nil,
	)
}
