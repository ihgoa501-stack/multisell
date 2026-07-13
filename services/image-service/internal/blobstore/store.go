package blobstore

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"
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
	ID                string `json:"id"`
	SizeBytes         int64  `json:"size_bytes"`
	MediaType         string `json:"media_type"`
	Format            string `json:"format"`
	Width             int    `json:"width"`
	Height            int    `json:"height"`
	Sandbox           bool   `json:"sandbox"`
	Watermarked       bool   `json:"watermarked"`
	NonPublishable    bool   `json:"non_publishable"`
	RestrictionReason string `json:"restriction_reason,omitempty"`
}

type Store struct {
	root string
	mu   sync.Mutex
}

// Restriction is monotonic provenance attached to an output blob. There is no
// API for clearing it: once any source marks bytes unsafe for publication,
// every later use of the same content-addressed blob remains restricted.
type Restriction struct {
	Sandbox        bool   `json:"sandbox"`
	Watermarked    bool   `json:"watermarked"`
	NonPublishable bool   `json:"non_publishable"`
	Reason         string `json:"reason"`
}

func New(root string) (*Store, error) {
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, err
	}
	return &Store{root: root}, nil
}

// Ping verifies that the configured blob root can complete the same durable
// filesystem operations required by normal writes: create, write, sync, read
// and cleanup. The probe never enters the content-addressed blob namespace and
// must leave no file behind.
func (s *Store) Ping(ctx context.Context) (err error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	payload := make([]byte, 32)
	if _, err := rand.Read(payload); err != nil {
		return err
	}
	f, err := os.CreateTemp(s.root, ".readiness-")
	if err != nil {
		return err
	}
	name := f.Name()
	closed := false
	defer func() {
		if !closed {
			if closeErr := f.Close(); err == nil {
				err = closeErr
			}
		}
		if removeErr := os.Remove(name); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) && err == nil {
			err = removeErr
		}
	}()
	if err = f.Chmod(0o600); err != nil {
		return err
	}
	if _, err = f.Write(payload); err != nil {
		return err
	}
	if err = f.Sync(); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	closed = true
	if err = ctx.Err(); err != nil {
		return err
	}
	got, err := os.ReadFile(name)
	if err != nil {
		return err
	}
	if !bytes.Equal(got, payload) {
		return errors.New("blob readiness read-back mismatch")
	}
	if err = os.Remove(name); err != nil {
		return err
	}
	return nil
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
	meta := Metadata{ID: id, SizeBytes: int64(len(b)), MediaType: mediaType, Format: format, Width: cfg.Width, Height: cfg.Height}
	if restriction, ok, err := s.GetRestriction(id); err != nil {
		return Metadata{}, err
	} else if ok {
		meta.Sandbox, meta.Watermarked, meta.NonPublishable = restriction.Sandbox, restriction.Watermarked, restriction.NonPublishable
		meta.RestrictionReason = restriction.Reason
	}
	return meta, nil
}

// MarkRestricted atomically adds immutable safety restrictions to a blob.
// Existing true flags can never be downgraded by a subsequent caller.
func (s *Store) MarkRestricted(id string, in Restriction) error {
	if _, err := s.Get(id); err != nil {
		return err
	}
	if !in.Sandbox || !in.Watermarked || !in.NonPublishable || strings.TrimSpace(in.Reason) == "" {
		return errors.New("sandbox restriction must be watermarked and non-publishable")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	current, _, err := s.getRestriction(id)
	if err != nil {
		return err
	}
	current.Sandbox = current.Sandbox || in.Sandbox
	current.Watermarked = current.Watermarked || in.Watermarked
	current.NonPublishable = current.NonPublishable || in.NonPublishable
	if current.Reason == "" {
		current.Reason = strings.TrimSpace(in.Reason)
	}
	b, err := json.Marshal(current)
	if err != nil {
		return err
	}
	return atomicWrite(filepath.Join(s.root, ".restriction-"+id+".json"), b)
}

func (s *Store) GetRestriction(id string) (Restriction, bool, error) {
	if !validID(id) {
		return Restriction{}, false, os.ErrNotExist
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getRestriction(id)
}

func (s *Store) getRestriction(id string) (Restriction, bool, error) {
	b, err := os.ReadFile(filepath.Join(s.root, ".restriction-"+id+".json"))
	if errors.Is(err, os.ErrNotExist) {
		return Restriction{}, false, nil
	}
	if err != nil {
		return Restriction{}, false, err
	}
	var out Restriction
	if err := json.Unmarshal(b, &out); err != nil {
		return Restriction{}, false, err
	}
	if !out.Sandbox || !out.Watermarked || !out.NonPublishable || out.Reason == "" {
		return Restriction{}, false, errors.New("invalid blob restriction metadata")
	}
	return out, true, nil
}

// ClaimProviderSubmit is a durable, fail-closed once-only gate. It is claimed
// before the first network byte can be sent, so a crash or ambiguous response
// can never cause an automatic second mutation for the same job/provider.
func (s *Store) ClaimProviderSubmit(provider, jobID string) (bool, error) {
	provider, jobID = strings.TrimSpace(provider), strings.TrimSpace(jobID)
	if provider == "" || jobID == "" || strings.ContainsAny(provider+jobID, "/\\\x00") {
		return false, errors.New("provider and job ID are required")
	}
	h := sha256.Sum256([]byte(provider + "\x00" + jobID))
	f, err := os.OpenFile(filepath.Join(s.root, ".provider-submit-"+hex.EncodeToString(h[:])), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, os.ErrExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return false, err
	}
	if err := f.Close(); err != nil {
		return false, err
	}
	if err := syncDirectory(s.root); err != nil {
		return false, err
	}
	return true, nil
}

func atomicWrite(path string, b []byte) error {
	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, ".metadata-")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if err := f.Chmod(0o600); err != nil {
		_ = f.Close()
		return err
	}
	if _, err := f.Write(b); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(name, path); err != nil {
		return err
	}
	return syncDirectory(dir)
}

func syncDirectory(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
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
