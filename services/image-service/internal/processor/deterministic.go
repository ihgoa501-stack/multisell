package processor

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
)

func Process(input []byte, width, height int, format string) ([]byte, error) {
	if len(input) == 0 || len(input) > 10<<20 || width < 100 || height < 100 || width > 4000 || height > 4000 {
		return nil, errors.New("invalid image request")
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(input))
	if err != nil || int64(cfg.Width)*int64(cfg.Height) > 40_000_000 {
		return nil, errors.New("unsafe image")
	}
	src, _, err := image.Decode(bytes.NewReader(input))
	if err != nil {
		return nil, err
	}
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(dst, dst.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	// Nearest-neighbour scaling is deterministic and dependency-free; higher quality resampling is a later compatible processor version.
	sb := src.Bounds()
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			sx := sb.Min.X + x*sb.Dx()/width
			sy := sb.Min.Y + y*sb.Dy()/height
			dst.Set(x, y, src.At(sx, sy))
		}
	}
	var out bytes.Buffer
	switch format {
	case "jpeg":
		err = jpeg.Encode(&out, dst, &jpeg.Options{Quality: 90})
	case "png":
		err = png.Encode(&out, dst)
	default:
		return nil, errors.New("unsupported format")
	}
	if err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
