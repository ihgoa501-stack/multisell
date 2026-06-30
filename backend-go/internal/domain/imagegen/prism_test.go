package imagegen

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// =============================================================
// Helper: create a test image
// =============================================================

func testImage(w, h int, fill func(x, y int) color.Color) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, fill(x, y))
		}
	}
	return img
}

func encodePNG(img image.Image) []byte {
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func allWhite(x, y int) color.Color {
	return color.RGBA{R: 255, G: 255, B: 255, A: 255}
}

func allRed(x, y int) color.Color {
	return color.RGBA{R: 255, G: 0, B: 0, A: 255}
}

func nearWhite(x, y int) color.Color {
	return color.RGBA{R: 240, G: 241, B: 242, A: 255}
}

func mixedColors(x, y int) color.Color {
	if x < 50 && y < 50 {
		return color.RGBA{R: 255, G: 0, B: 0, A: 255}
	}
	return color.RGBA{R: 255, G: 255, B: 255, A: 255}
}

// =============================================================
// White background detection
// =============================================================

func TestDetectWhiteBackground_FullWhite(t *testing.T) {
	img := testImage(100, 100, allWhite)
	if !DetectWhiteBackground(img) {
		t.Fatal("expected white background detection for all-white image")
	}
}

func TestDetectWhiteBackground_ColoredBorder(t *testing.T) {
	img := testImage(100, 100, mixedColors)
	if DetectWhiteBackground(img) {
		t.Fatal("expected non-white detection for image with colored border")
	}
}

func TestDetectWhiteBackground_NearWhite(t *testing.T) {
	img := testImage(100, 100, nearWhite)
	if !DetectWhiteBackground(img) {
		t.Fatal("expected white detection for near-white image")
	}
}

func TestDetectWhiteBackground_SinglePixel(t *testing.T) {
	img := testImage(1, 1, allWhite)
	if !DetectWhiteBackground(img) {
		t.Fatal("expected white detection for single pixel white")
	}
}

func TestDetectWhiteBackground_Empty(t *testing.T) {
	img := testImage(0, 0, allWhite)
	if DetectWhiteBackground(img) {
		t.Fatal("expected false for empty image")
	}
}

// =============================================================
// White background replacement
// =============================================================

func TestEnsureWhiteBackground_NearWhite(t *testing.T) {
	img := testImage(10, 10, nearWhite)
	out := EnsureWhiteBackground(img)
	bounds := out.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := out.At(x, y).RGBA()
			if r>>8 != 255 || g>>8 != 255 || b>>8 != 255 {
				t.Fatalf("expected pure white at (%d,%d), got %d,%d,%d", x, y, r>>8, g>>8, b>>8)
			}
		}
	}
}

func TestEnsureWhiteBackground_PreservesColor(t *testing.T) {
	img := testImage(10, 10, func(x, y int) color.Color {
		if x < 5 {
			return color.RGBA{R: 100, G: 100, B: 100, A: 255}
		}
		return nearWhite(x, y)
	})
	out := EnsureWhiteBackground(img)
	r, g, b, _ := out.At(2, 2).RGBA()
	if r>>8 != 100 {
		t.Fatalf("expected preserved color 100 at (2,2), got %d", r>>8)
	}
	// Near-white area should be pure white.
	r, g, b, _ = out.At(7, 2).RGBA()
	if r>>8 != 255 || g>>8 != 255 || b>>8 != 255 {
		t.Fatalf("expected white at (7,2), got %d,%d,%d", r>>8, g>>8, b>>8)
	}
}

func TestEnsureWhiteBackground_PreservesAlpha(t *testing.T) {
	img := testImage(5, 5, func(x, y int) color.Color {
		return color.RGBA{R: 200, G: 200, B: 200, A: 128}
	})
	out := EnsureWhiteBackground(img)
	_, _, _, a := out.At(2, 2).RGBA()
	if a>>8 != 128 {
		t.Fatalf("expected alpha 128, got %d", a>>8)
	}
}

// =============================================================
// Text watermark
// =============================================================

func TestTextWatermark_AllPositions(t *testing.T) {
	positions := []struct {
		pos WatermarkPosition
		name string
	}{
		{WatermarkBottomRight, "bottom-right"},
		{WatermarkBottomLeft, "bottom-left"},
		{WatermarkTopRight, "top-right"},
		{WatermarkTopLeft, "top-left"},
		{WatermarkCenter, "center"},
	}
	for _, p := range positions {
		prism := NewPrism([]PlatformSpec{
			{Name: "test", Width: 200, Height: 200, Format: "png"},
		}, &WatermarkConfig{
			Type:     "text",
			Text:     "Test Watermark",
			Position: p.pos,
			Margin:   5,
		})
		imgBytes := encodePNG(testImage(200, 200, allWhite))
		result, err := prism.ProcessSpecs(imgBytes)
		if err != nil {
			t.Fatalf("watermark %s: %v", p.name, err)
		}
		if len(result["test"]) == 0 {
			t.Fatalf("watermark %s: empty result", p.name)
		}
	}
}

func TestTextWatermark_EmptyText(t *testing.T) {
	prism := NewPrism(nil, &WatermarkConfig{Type: "text", Text: ""})
	imgBytes := encodePNG(testImage(100, 100, allWhite))
	result, err := prism.ProcessSpecs(imgBytes)
	if err != nil {
		t.Fatal("empty text should not error")
	}
	if len(result) == 0 {
		t.Fatal("expected results even without text")
	}
}

// =============================================================
// Image watermark
// =============================================================

func TestImageWatermark_FileNotFound(t *testing.T) {
	prism := NewPrism(nil, &WatermarkConfig{
		Type:      "image",
		ImagePath: "/nonexistent/watermark.png",
	})
	imgBytes := encodePNG(testImage(100, 100, allWhite))
	result, err := prism.ProcessSpecs(imgBytes)
	if err != nil {
		t.Fatal("missing watermark file should not cause error (silent skip)")
	}
	if len(result) == 0 {
		t.Fatal("expected results even without watermark")
	}
}

// =============================================================
// Batch processing
// =============================================================

func TestPrism_DefaultSpecs(t *testing.T) {
	p := NewPrism(nil, nil)
	if len(p.specs) != 4 {
		t.Fatalf("expected 4 default specs, got %d", len(p.specs))
	}
}

func TestPrism_BatchAllFormats(t *testing.T) {
	specs := []PlatformSpec{
		{Name: "amazon", Width: 100, Height: 100, Format: "jpeg", Quality: 90},
		{Name: "shopee", Width: 80, Height: 80, Format: "jpeg", Quality: 85},
	}
	p := NewPrism(specs, nil)
	imgBytes := encodePNG(testImage(200, 200, allWhite))
	result, err := p.ProcessSpecs(imgBytes)
	if err != nil {
		t.Fatalf("batch processing failed: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result))
	}
	for _, name := range []string{"amazon", "shopee"} {
		if _, ok := result[name]; !ok {
			t.Fatalf("missing result for %s", name)
		}
		if len(result[name]) == 0 {
			t.Fatalf("empty result for %s", name)
		}
	}
}

func TestPrism_InvalidInput(t *testing.T) {
	p := NewPrism(nil, nil)
	_, err := p.ProcessSpecs([]byte("not-an-image"))
	if err == nil {
		t.Fatal("expected error for invalid image input")
	}
}

func TestPrism_BatchWithTextWatermark(t *testing.T) {
	p := NewPrism([]PlatformSpec{
		{Name: "test", Width: 100, Height: 100, Format: "png"},
	}, DefaultTextWatermark("Test"))
	imgBytes := encodePNG(testImage(100, 100, allWhite))
	result, err := p.ProcessSpecs(imgBytes)
	if err != nil {
		t.Fatalf("batch with watermark: %v", err)
	}
	if len(result["test"]) == 0 {
		t.Fatal("expected result with watermark")
	}
}

func TestPrism_BatchWithImageWatermark(t *testing.T) {
	// Create a temporary watermark image.
	dir := t.TempDir()
	wmPath := filepath.Join(dir, "wm.png")
	wmImg := testImage(20, 20, func(x, y int) color.Color {
		return color.RGBA{R: 0, G: 0, B: 0, A: 200}
	})
	wmBytes := encodePNG(wmImg)
	if err := os.WriteFile(wmPath, wmBytes, 0644); err != nil {
		t.Fatal(err)
	}

	p := NewPrism(nil, &WatermarkConfig{
		Type:      "image",
		ImagePath: wmPath,
		Position:  WatermarkTopLeft,
	})
	imgBytes := encodePNG(testImage(100, 100, allWhite))
	result, err := p.ProcessSpecs(imgBytes)
	if err != nil {
		t.Fatalf("batch with image watermark: %v", err)
	}
	if len(result) == 0 {
		t.Fatal("expected results with image watermark")
	}
}

// =============================================================
// Cache
// =============================================================

func TestCache_Basic(t *testing.T) {
	c := NewPrismCache()
	if c.CacheSize() != 0 {
		t.Fatal("expected empty cache")
	}
	c.Set("key1", []byte("data1"))
	if c.CacheSize() != 1 {
		t.Fatal("expected cache size 1")
	}
	data, ok := c.Get("key1")
	if !ok || string(data) != "data1" {
		t.Fatal("cache get failed")
	}
}

func TestCache_Miss(t *testing.T) {
	c := NewPrismCache()
	_, ok := c.Get("nonexistent")
	if ok {
		t.Fatal("expected miss for nonexistent key")
	}
}

func TestCache_Clear(t *testing.T) {
	c := NewPrismCache()
	c.Set("a", []byte("1"))
	c.Set("b", []byte("2"))
	if c.CacheSize() != 2 {
		t.Fatal("expected size 2")
	}
	c.CacheClear()
	if c.CacheSize() != 0 {
		t.Fatal("expected empty after clear")
	}
}

func TestCache_ConcurrentSafety(t *testing.T) {
	c := NewPrismCache()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := string(rune('A' + n%26))
			c.Set(key, []byte{byte(n)})
			c.Get(key)
			c.CacheSize()
		}(i)
	}
	wg.Wait()
}

func TestPrism_CacheHit(t *testing.T) {
	p := NewPrism(nil, nil)
	imgBytes := encodePNG(testImage(100, 100, allWhite))

	// First call - cache miss.
	result1, err := p.ProcessSpecs(imgBytes)
	if err != nil {
		t.Fatal(err)
	}

	// Second call - cache hit.
	result2, err := p.ProcessSpecs(imgBytes)
	if err != nil {
		t.Fatal(err)
	}

	for name, data1 := range result1 {
		data2, ok := result2[name]
		if !ok {
			t.Fatalf("missing %s in second result", name)
		}
		if !bytes.Equal(data1, data2) {
			t.Fatalf("cache hit returned different data for %s", name)
		}
	}
}

func TestPrism_CacheAfterClear(t *testing.T) {
	p := NewPrism(nil, nil)
	imgBytes := encodePNG(testImage(100, 100, allWhite))
	p.ProcessSpecs(imgBytes)
	size := p.cache.CacheSize()
	if size == 0 {
		t.Fatal("expected cache entries")
	}
	p.cache.CacheClear()
	if p.cache.CacheSize() != 0 {
		t.Fatal("expected empty cache after clear")
	}
}

func TestPrism_NoCache(t *testing.T) {
	p := NewPrism(nil, nil) // cache always on
	p.ProcessSpecs(encodePNG(testImage(50, 50, allWhite)))
	if p.cache.CacheSize() == 0 {
		t.Fatal("expected cache to be populated")
	}
}
