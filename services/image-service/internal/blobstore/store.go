package blobstore

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
)

const (
	MaxBlobBytes = 10 << 20
	MaxPixels    = 40_000_000
)

var (
	ErrInvalidImage  = errors.New("invalid image blob")
	ErrTooLarge      = errors.New("image blob exceeds size limit")
	ErrTooManyPixels = errors.New("image exceeds pixel limit")
	ErrHashMismatch  = errors.New("blob hash mismatch")
)

type Metadata struct {
	ID        string `json:"id"`
	SizeBytes int64  `json:"size_bytes"`
	MediaType string `json:"media_type"`
	Format    string `json:"format"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
}

type Store struct{ root string }

func New(root string) (*Store, error) {
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, err
	}
	return &Store{root: root}, nil
}

// Put validates and decodes a static raster image, then re-encodes it before
// persistence. Re-encoding strips EXIF, comments, profiles and other metadata;
// the returned ID addresses the sanitized bytes, not the untrusted upload.
func (s *Store) Put(data []byte) (string, error) {
	clean, _, err := sanitize(data)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(clean)
	id := hex.EncodeToString(h[:])
	path := filepath.Join(s.root, id)
	if _, err := os.Stat(path); err == nil {
		return id, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	tmp, err := os.CreateTemp(s.root, ".staged-")
	if err != nil {
		return "", err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return "", err
	}
	if _, err := tmp.Write(clean); err != nil {
		tmp.Close()
		return "", err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	if err := os.Rename(name, path); err != nil {
		return "", err
	}
	return id, nil
}

func (s *Store) Get(id string) ([]byte, error) {
	if !validID(id) {
		return nil, os.ErrNotExist
	}
	b, err := os.ReadFile(filepath.Join(s.root, id))
	if err != nil {
		return nil, err
	}
	h := sha256.Sum256(b)
	if hex.EncodeToString(h[:]) != id {
		return nil, ErrHashMismatch
	}
	return b, nil
}

func (s *Store) GetMetadata(id string) (Metadata, error) {
	b, err := s.Get(id)
	if err != nil {
		return Metadata{}, err
	}
	cfg, format, mediaType, err := inspect(b)
	if err != nil {
		return Metadata{}, err
	}
	return Metadata{ID: id, SizeBytes: int64(len(b)), MediaType: mediaType, Format: format, Width: cfg.Width, Height: cfg.Height}, nil
}

func validID(id string) bool {
	if len(id) != 64 || strings.ContainsAny(id, "/\\") || strings.ToLower(id) != id {
		return false
	}
	_, err := hex.DecodeString(id)
	return err == nil
}

func sanitize(data []byte) ([]byte, Metadata, error) {
	if len(data) == 0 {
		return nil, Metadata{}, fmt.Errorf("%w: empty file", ErrInvalidImage)
	}
	if len(data) > MaxBlobBytes {
		return nil, Metadata{}, ErrTooLarge
	}
	cfg, format, mediaType, err := inspect(data)
	if err != nil {
		return nil, Metadata{}, err
	}
	if int64(cfg.Width)*int64(cfg.Height) > MaxPixels {
		return nil, Metadata{}, ErrTooManyPixels
	}
	decoded, decodedFormat, err := image.Decode(bytes.NewReader(data))
	if err != nil || decodedFormat != format {
		return nil, Metadata{}, fmt.Errorf("%w: decode failed", ErrInvalidImage)
	}
	var out bytes.Buffer
	switch format {
	case "png":
		err = png.Encode(&out, decoded)
	case "jpeg":
		err = jpeg.Encode(&out, decoded, &jpeg.Options{Quality: 92})
	default:
		err = ErrInvalidImage
	}
	if err != nil {
		return nil, Metadata{}, fmt.Errorf("%w: encode failed", ErrInvalidImage)
	}
	meta := Metadata{SizeBytes: int64(out.Len()), MediaType: mediaType, Format: format, Width: cfg.Width, Height: cfg.Height}
	return out.Bytes(), meta, nil
}

func inspect(data []byte) (image.Config, string, string, error) {
	var expected, mediaType string
	switch {
	case len(data) >= 8 && bytes.Equal(data[:8], []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}):
		expected, mediaType = "png", "image/png"
		if animatedPNG(data) {
			return image.Config{}, "", "", fmt.Errorf("%w: animated PNG is unsupported", ErrInvalidImage)
		}
	case len(data) >= 3 && data[0] == 0xff && data[1] == 0xd8 && data[2] == 0xff:
		expected, mediaType = "jpeg", "image/jpeg"
	default:
		// GIF, SVG, WebP and MIME-spoofed/non-image input are deliberately unsupported.
		return image.Config{}, "", "", fmt.Errorf("%w: only static JPEG and PNG are supported", ErrInvalidImage)
	}
	cfg, actual, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || actual != expected || cfg.Width <= 0 || cfg.Height <= 0 {
		return image.Config{}, "", "", fmt.Errorf("%w: invalid %s payload", ErrInvalidImage, expected)
	}
	if int64(cfg.Width)*int64(cfg.Height) > MaxPixels {
		return image.Config{}, "", "", ErrTooManyPixels
	}
	return cfg, expected, mediaType, nil
}

func animatedPNG(data []byte) bool {
	// PNG chunks begin after the 8-byte signature. Detect acTL before decoding;
	// malformed lengths are left for the standard decoder to reject.
	for pos := 8; pos+12 <= len(data); {
		length := int64(data[pos])<<24 | int64(data[pos+1])<<16 | int64(data[pos+2])<<8 | int64(data[pos+3])
		if length < 0 || length > int64(len(data)) || int64(pos)+12+length > int64(len(data)) {
			return false
		}
		kind := string(data[pos+4 : pos+8])
		if kind == "acTL" {
			return true
		}
		pos += 12 + int(length)
		if kind == "IEND" {
			return false
		}
	}
	return false
}
