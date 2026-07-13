package photoroom

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/png"
)

const sandboxWord = "SANDBOX"

var sandboxGlyphs = map[byte][7]byte{
	'S': {0x1f, 0x10, 0x10, 0x1f, 0x01, 0x01, 0x1f},
	'A': {0x0e, 0x11, 0x11, 0x1f, 0x11, 0x11, 0x11},
	'N': {0x11, 0x19, 0x19, 0x15, 0x13, 0x13, 0x11},
	'D': {0x1e, 0x11, 0x11, 0x11, 0x11, 0x11, 0x1e},
	'B': {0x1e, 0x11, 0x11, 0x1e, 0x11, 0x11, 0x1e},
	'O': {0x0e, 0x11, 0x11, 0x11, 0x11, 0x11, 0x0e},
	'X': {0x11, 0x11, 0x0a, 0x04, 0x0a, 0x11, 0x11},
}

// applyAndVerifySandboxWatermark adds an opaque, deterministic SANDBOX banner,
// encodes a fresh PNG, then decodes and checks every marker pixel. Watermarked
// metadata is only persisted after this byte-level verification succeeds.
func applyAndVerifySandboxWatermark(raw []byte) ([]byte, error) {
	source, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	b := source.Bounds()
	const glyphWidth, glyphHeight, spacing, margin = 5, 7, 1, 2
	unitsWide := len(sandboxWord)*(glyphWidth+spacing) - spacing + 2*margin
	unitsHigh := glyphHeight + 2*margin
	if b.Dx() < unitsWide || b.Dy() < unitsHigh {
		return nil, errors.New("image is too small for SANDBOX watermark")
	}
	scale := b.Dx() / unitsWide
	if byHeight := b.Dy() / (unitsHigh * 3); byHeight < scale {
		scale = byHeight
	}
	if scale < 1 {
		scale = 1
	}
	if scale > 12 {
		scale = 12
	}
	width, height := unitsWide*scale, unitsHigh*scale
	x0 := b.Min.X + (b.Dx()-width)/2
	y0 := b.Max.Y - height
	dst := image.NewNRGBA(b)
	draw.Draw(dst, b, source, b.Min, draw.Src)
	red := color.NRGBA{R: 180, G: 0, B: 0, A: 255}
	white := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	draw.Draw(dst, image.Rect(x0, y0, x0+width, y0+height), &image.Uniform{C: red}, image.Point{}, draw.Src)
	for i := range sandboxWord {
		glyph := sandboxGlyphs[sandboxWord[i]]
		gx := x0 + (margin+i*(glyphWidth+spacing))*scale
		for row, bits := range glyph {
			for col := 0; col < glyphWidth; col++ {
				if bits&(1<<uint(glyphWidth-1-col)) == 0 {
					continue
				}
				draw.Draw(dst, image.Rect(gx+col*scale, y0+(margin+row)*scale, gx+(col+1)*scale, y0+(margin+row+1)*scale), &image.Uniform{C: white}, image.Point{}, draw.Src)
			}
		}
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, dst); err != nil {
		return nil, err
	}
	verified, err := png.Decode(bytes.NewReader(encoded.Bytes()))
	if err != nil || !hasExactSandboxWatermark(verified, x0, y0, scale) {
		return nil, errors.New("SANDBOX watermark verification failed")
	}
	return encoded.Bytes(), nil
}

func hasExactSandboxWatermark(img image.Image, x0, y0, scale int) bool {
	const glyphWidth, glyphHeight, spacing, margin = 5, 7, 1, 2
	red := color.NRGBAModel.Convert(color.NRGBA{R: 180, A: 255}).(color.NRGBA)
	white := color.NRGBAModel.Convert(color.NRGBA{R: 255, G: 255, B: 255, A: 255}).(color.NRGBA)
	if color.NRGBAModel.Convert(img.At(x0, y0)).(color.NRGBA) != red {
		return false
	}
	for i := range sandboxWord {
		glyph := sandboxGlyphs[sandboxWord[i]]
		gx := x0 + (margin+i*(glyphWidth+spacing))*scale
		for row, bits := range glyph {
			for col := 0; col < glyphWidth; col++ {
				if bits&(1<<uint(glyphWidth-1-col)) != 0 && color.NRGBAModel.Convert(img.At(gx+col*scale, y0+(margin+row)*scale)).(color.NRGBA) != white {
					return false
				}
			}
		}
	}
	return true
}
