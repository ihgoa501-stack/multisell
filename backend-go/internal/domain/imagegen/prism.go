package imagegen

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"os"
	"sync"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// PlatformSpec defines the output spec for a single e-commerce platform.
type PlatformSpec struct {
	Name    string
	Width   int
	Height  int
	Format  string // "jpeg" or "png"
	Quality int    // 1-100 for jpeg
}

// DefaultPlatformSpecs returns the standard specs for supported platforms.
func DefaultPlatformSpecs() []PlatformSpec {
	return []PlatformSpec{
		{Name: "amazon", Width: 1000, Height: 1000, Format: "jpeg", Quality: 90},
		{Name: "shopee", Width: 800, Height: 800, Format: "jpeg", Quality: 85},
		{Name: "lazada", Width: 1200, Height: 1200, Format: "jpeg", Quality: 90},
		{Name: "ozon", Width: 900, Height: 1200, Format: "jpeg", Quality: 85},
	}
}

// WatermarkPosition defines where to place the watermark.
type WatermarkPosition int

const (
	WatermarkBottomRight WatermarkPosition = iota
	WatermarkBottomLeft
	WatermarkTopRight
	WatermarkTopLeft
	WatermarkCenter
)

// WatermarkConfig configures a watermark overlay.
type WatermarkConfig struct {
	Type     string            // "text" or "image"
	Text     string            // text for text watermark
	ImagePath string           // path for image watermark
	Position WatermarkPosition // placement
	Margin   int               // margin from edges, default 10
}

// DefaultTextWatermark returns a simple text watermark config.
func DefaultTextWatermark(text string) *WatermarkConfig {
	return &WatermarkConfig{
		Type:     "text",
		Text:     text,
		Position: WatermarkBottomRight,
		Margin:   10,
	}
}

// Prism is the image processing pipeline engine.
type Prism struct {
	specs     []PlatformSpec
	watermark *WatermarkConfig
	cache     *PrismCache
}

// NewPrism creates a new Prism engine with the given specs and watermark config.
func NewPrism(specs []PlatformSpec, wm *WatermarkConfig) *Prism {
	if specs == nil {
		specs = DefaultPlatformSpecs()
	}
	return &Prism{
		specs:     specs,
		watermark: wm,
		cache:     NewPrismCache(),
	}
}

// SetWatermark updates the watermark config.
func (p *Prism) SetWatermark(cfg *WatermarkConfig) {
	p.watermark = cfg
}

// ProcessSpecs decodes the input image once and produces all platform-specific outputs.
// Returns a map keyed by platform name. Each spec is cached by SHA256 hash.
func (p *Prism) ProcessSpecs(imgBytes []byte) (map[string][]byte, error) {
	result := make(map[string][]byte, len(p.specs))
	hash := sha256.Sum256(imgBytes)
	baseKey := fmt.Sprintf("%x", hash)

	// Decode the source image once.
	src, format, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		return nil, fmt.Errorf("imagegen: decode source: %w", err)
	}
	_ = format

	for _, spec := range p.specs {
		key := baseKey + ":" + spec.Name

		// Check cache.
		if data, ok := p.cache.Get(key); ok {
			result[spec.Name] = data
			continue
		}

		// Resize.
		dst := image.NewRGBA(image.Rect(0, 0, spec.Width, spec.Height))
		xdraw.BiLinear.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)

		// Ensure white background.
		dst = p.ensureWhiteBackground(dst)

		// Apply watermark.
		if p.watermark != nil {
			p.applyWatermark(dst, spec.Width, spec.Height)
		}

		// Encode.
		var buf bytes.Buffer
		switch spec.Format {
		case "jpeg":
			err = jpeg.Encode(&buf, dst, &jpeg.Options{Quality: spec.Quality})
		case "png":
			err = png.Encode(&buf, dst)
		default:
			err = fmt.Errorf("unsupported format: %s", spec.Format)
		}
		if err != nil {
			return nil, fmt.Errorf("imagegen: encode %s: %w", spec.Name, err)
		}

		p.cache.Set(key, buf.Bytes())
		result[spec.Name] = buf.Bytes()
	}

	return result, nil
}

// ensureWhiteBackground replaces near-white pixels with pure white.
func (p *Prism) ensureWhiteBackground(img *image.RGBA) *image.RGBA {
	bounds := img.Bounds()
	out := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			// Convert to 8-bit.
			r8, g8, b8 := r>>8, g>>8, b>>8
			if r8 >= 230 && g8 >= 230 && b8 >= 230 {
				out.Set(x, y, color.RGBA{R: 255, G: 255, B: 255, A: uint8(a >> 8)})
			} else {
				out.Set(x, y, color.RGBA{R: uint8(r8), G: uint8(g8), B: uint8(b8), A: uint8(a >> 8)})
			}
		}
	}
	return out
}

// applyWatermark overlays the configured watermark onto the image.
func (p *Prism) applyWatermark(img *image.RGBA, imgW, imgH int) {
	if p.watermark == nil {
		return
	}
	margin := p.watermark.Margin
	if margin <= 0 {
		margin = 10
	}

	switch p.watermark.Type {
	case "text":
		if p.watermark.Text == "" {
			return
		}
		face := basicfont.Face7x13
		textWidth := font.MeasureString(face, p.watermark.Text).Round()
		textHeight := face.Metrics().Height.Round()

		var x, y int
		switch p.watermark.Position {
		case WatermarkBottomRight:
			x, y = imgW-textWidth-margin, imgH-margin
		case WatermarkBottomLeft:
			x, y = margin, imgH-margin
		case WatermarkTopRight:
			x, y = imgW-textWidth-margin, textHeight+margin
		case WatermarkTopLeft:
			x, y = margin, textHeight+margin
		case WatermarkCenter:
			x, y = (imgW-textWidth)/2, (imgH+textHeight)/2
		}

		d := &font.Drawer{
			Dst:  img,
			Src:  image.NewUniform(color.RGBA{R: 255, G: 255, B: 255, A: 180}),
			Face: face,
			Dot:  fixed.P(x, y),
		}
		d.DrawString(p.watermark.Text)

	case "image":
		if p.watermark.ImagePath == "" {
			return
		}
		wmImg, err := p.loadWatermarkImage(p.watermark.ImagePath)
		if err != nil {
			return
		}
		wmBounds := wmImg.Bounds()
		wmW, wmH := wmBounds.Dx(), wmBounds.Dy()

		var x, y int
		switch p.watermark.Position {
		case WatermarkBottomRight:
			x, y = imgW-wmW-margin, imgH-wmH-margin
		case WatermarkBottomLeft:
			x, y = margin, imgH-wmH-margin
		case WatermarkTopRight:
			x, y = imgW-wmW-margin, margin
		case WatermarkTopLeft:
			x, y = margin, margin
		case WatermarkCenter:
			x, y = (imgW-wmW)/2, (imgH-wmH)/2
		}

		draw.Draw(img, image.Rect(x, y, x+wmW, y+wmH), wmImg, wmBounds.Min, draw.Over)
	}
}

func (p *Prism) loadWatermarkImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	return img, err
}

// DetectWhiteBackground returns true if at least 90% of perimeter pixels
// have RGB values >= 230 (near-white).
func DetectWhiteBackground(img image.Image) bool {
	bounds := img.Bounds()
	var total, white int
	// Top and bottom rows.
	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		total++
		r, g, b, _ := img.At(x, bounds.Min.Y).RGBA()
		if r>>8 >= 230 && g>>8 >= 230 && b>>8 >= 230 {
			white++
		}
		total++
		r, g, b, _ = img.At(x, bounds.Max.Y-1).RGBA()
		if r>>8 >= 230 && g>>8 >= 230 && b>>8 >= 230 {
			white++
		}
	}
	// Left and right columns (skip corners already counted).
	for y := bounds.Min.Y + 1; y < bounds.Max.Y-1; y++ {
		total++
		r, g, b, _ := img.At(bounds.Min.X, y).RGBA()
		if r>>8 >= 230 && g>>8 >= 230 && b>>8 >= 230 {
			white++
		}
		total++
		r, g, b, _ = img.At(bounds.Max.X-1, y).RGBA()
		if r>>8 >= 230 && g>>8 >= 230 && b>>8 >= 230 {
			white++
		}
	}
	if total == 0 {
		return false
	}
	return float64(white)/float64(total) >= 0.9
}

// EnsureWhiteBackground replaces near-white pixels with pure white in any image.
func EnsureWhiteBackground(img image.Image) image.Image {
	bounds := img.Bounds()
	out := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			r8, g8, b8 := r>>8, g>>8, b>>8
			if r8 >= 230 && g8 >= 230 && b8 >= 230 {
				out.Set(x, y, color.RGBA{R: 255, G: 255, B: 255, A: uint8(a >> 8)})
			} else {
				out.Set(x, y, color.RGBA{R: uint8(r8), G: uint8(g8), B: uint8(b8), A: uint8(a >> 8)})
			}
		}
	}
	return out
}

// --- Cache ---

// cacheEntry stores cached image bytes.
type cacheEntry struct {
	data []byte
}

// PrismCache is a thread-safe in-memory cache for processed images.
type PrismCache struct {
	mu    sync.RWMutex
	store map[string]*cacheEntry
}

// NewPrismCache creates a new cache.
func NewPrismCache() *PrismCache {
	return &PrismCache{store: make(map[string]*cacheEntry)}
}

// Get returns cached data by key.
func (c *PrismCache) Get(key string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.store[key]
	if !ok {
		return nil, false
	}
	return entry.data, true
}

// Set stores data by key.
func (c *PrismCache) Set(key string, data []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store[key] = &cacheEntry{data: data}
}

// CacheSize returns the number of cached entries.
func (c *PrismCache) CacheSize() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.store)
}

// CacheClear removes all cached entries.
func (c *PrismCache) CacheClear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store = make(map[string]*cacheEntry)
}
