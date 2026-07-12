package blobstore

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestPutAcceptsSanitizesAndDescribesStaticRaster(t *testing.T) {
	store, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, format, mediaType string
		encode                  func(*bytes.Buffer) error
	}{
		{name: "png", format: "png", mediaType: "image/png", encode: func(out *bytes.Buffer) error { return png.Encode(out, testImage(3, 2)) }},
		{name: "jpeg", format: "jpeg", mediaType: "image/jpeg", encode: func(out *bytes.Buffer) error { return jpeg.Encode(out, testImage(3, 2), &jpeg.Options{Quality: 75}) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var input bytes.Buffer
			if err := tc.encode(&input); err != nil {
				t.Fatal(err)
			}
			// Trailing metadata-like bytes are accepted but cannot survive canonical re-encoding.
			input.WriteString("secret-metadata-marker")
			id, err := store.Put(input.Bytes())
			if err != nil {
				t.Fatal(err)
			}
			got, err := store.Get(id)
			if err != nil {
				t.Fatal(err)
			}
			if bytes.Contains(got, []byte("secret-metadata-marker")) {
				t.Fatal("untrusted metadata survived sanitization")
			}
			h := sha256.Sum256(got)
			if id != hex.EncodeToString(h[:]) {
				t.Fatal("ID does not address sanitized content")
			}
			meta, err := store.GetMetadata(id)
			if err != nil {
				t.Fatal(err)
			}
			if meta.ID != id || meta.Format != tc.format || meta.MediaType != tc.mediaType || meta.Width != 3 || meta.Height != 2 || meta.SizeBytes != int64(len(got)) {
				t.Fatalf("unexpected metadata: %#v", meta)
			}
		})
	}
}

func TestPutIsContentAddressedAndIdempotent(t *testing.T) {
	store, _ := New(t.TempDir())
	var input bytes.Buffer
	_ = png.Encode(&input, testImage(2, 2))
	first, err := store.Put(input.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.Put(input.Bytes())
	if err != nil || second != first {
		t.Fatalf("second put = %q, %v; first = %q", second, err, first)
	}
}

func TestPutRejectsUnsafeOrUnsupportedInput(t *testing.T) {
	store, _ := New(t.TempDir())
	var gif bytes.Buffer
	gif.WriteString("GIF89a")
	var fakePNG bytes.Buffer
	fakePNG.Write([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'})
	fakePNG.WriteString("<svg>not a png</svg>")
	animatedPNG := append(pngHeader(1, 1), pngChunk("acTL", make([]byte, 8))...)
	cases := map[string][]byte{
		"empty":        nil,
		"svg":          []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`),
		"gif":          gif.Bytes(),
		"webp":         append([]byte("RIFF\x00\x00\x00\x00WEBPVP8 "), make([]byte, 16)...),
		"fake mime":    fakePNG.Bytes(),
		"animated png": animatedPNG,
		"random":       []byte("definitely not an image"),
		"over 10 MiB":  make([]byte, MaxBlobBytes+1),
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := store.Put(input); err == nil {
				t.Fatal("unsafe input accepted")
			}
		})
	}
}

func TestPutStripsRealPNGTextChunk(t *testing.T) {
	store, _ := New(t.TempDir())
	var encoded bytes.Buffer
	_ = png.Encode(&encoded, testImage(2, 2))
	b := encoded.Bytes()
	// Insert a valid tEXt chunk immediately after IHDR.
	withMetadata := append([]byte(nil), b[:33]...)
	withMetadata = append(withMetadata, pngChunk("tEXt", []byte("Comment\x00supplier-secret"))...)
	withMetadata = append(withMetadata, b[33:]...)
	id, err := store.Put(withMetadata)
	if err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(id)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(got, []byte("supplier-secret")) {
		t.Fatal("PNG text metadata survived sanitization")
	}
}

func TestPutRejectsPixelBombBeforeFullDecode(t *testing.T) {
	store, _ := New(t.TempDir())
	b := pngHeader(10_000, 10_000)
	_, err := store.Put(b)
	if !errors.Is(err, ErrTooManyPixels) {
		t.Fatalf("got %v, want ErrTooManyPixels", err)
	}
}

func TestGetDetectsTamperingAndRejectsInvalidIDs(t *testing.T) {
	root := t.TempDir()
	store, _ := New(root)
	var input bytes.Buffer
	_ = png.Encode(&input, testImage(2, 2))
	id, err := store.Put(input.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, id), []byte("tampered"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(id); !errors.Is(err, ErrHashMismatch) {
		t.Fatalf("got %v, want ErrHashMismatch", err)
	}
	for _, bad := range []string{"../escape", "ABCDEF" + id[6:], stringsOf('z', 64), id[:63]} {
		if _, err := store.Get(bad); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("Get(%q) = %v, want not exist", bad, err)
		}
	}
}

func testImage(width, height int) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.NRGBA{R: uint8(20 + x), G: uint8(40 + y), B: 60, A: 255})
		}
	}
	return img
}

func stringsOf(ch byte, count int) string { return string(bytes.Repeat([]byte{ch}, count)) }

func pngHeader(width, height uint32) []byte {
	out := append([]byte(nil), []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}...)
	payload := make([]byte, 13)
	binary.BigEndian.PutUint32(payload[0:4], width)
	binary.BigEndian.PutUint32(payload[4:8], height)
	payload[8], payload[9], payload[10], payload[11], payload[12] = 8, 6, 0, 0, 0
	chunk := append([]byte("IHDR"), payload...)
	length := make([]byte, 4)
	binary.BigEndian.PutUint32(length, uint32(len(payload)))
	out = append(out, length...)
	out = append(out, chunk...)
	crc := make([]byte, 4)
	binary.BigEndian.PutUint32(crc, crc32.ChecksumIEEE(chunk))
	return append(out, crc...)
}

func pngChunk(kind string, payload []byte) []byte {
	length := make([]byte, 4)
	binary.BigEndian.PutUint32(length, uint32(len(payload)))
	chunk := append([]byte(kind), payload...)
	crc := make([]byte, 4)
	binary.BigEndian.PutUint32(crc, crc32.ChecksumIEEE(chunk))
	out := append(length, chunk...)
	return append(out, crc...)
}
