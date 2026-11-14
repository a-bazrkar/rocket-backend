package storage

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"

	"github.com/gabriel-vasile/mimetype"
	"github.com/spf13/afero"
)

var (
	ErrEmpty    = errors.New("storage: file is empty")
	ErrTooLarge = errors.New("storage: file is too large")
	ErrMIME     = errors.New("storage: mime type is not allowed")
	ErrNotFound = errors.New("storage: file not found")
	ErrKey      = errors.New("storage: invalid key")
)

// Object is a stored file. Width and Height are 0 when the file is not a jpeg, png, or gif.
type Object struct {
	Key    string
	Size   int64
	SHA256 [32]byte
	MIME   string
	Width  int
	Height int
}

type Store struct {
	fs  afero.Fs
	max int64
}

// New stores files under cfg.Root. The key is the hex SHA-256, so the same bytes land on one path.
func New(cfg Config) (*Store, error) {
	if cfg.Root == "" {
		return nil, fmt.Errorf("storage: root is required")
	}
	if cfg.MaxBytes <= 0 {
		return nil, fmt.Errorf("storage: max bytes must be positive")
	}
	if err := os.MkdirAll(cfg.Root, 0o755); err != nil {
		return nil, fmt.Errorf("storage: root: %w", err)
	}
	return newStore(afero.NewBasePathFs(afero.NewOsFs(), cfg.Root), cfg.MaxBytes), nil
}

func newStore(fs afero.Fs, max int64) *Store {
	return &Store{fs: fs, max: max}
}

// Put hashes the body, checks its sniffed type, and writes it.
// A second put of the same bytes returns the existing key.
func (s *Store) Put(ctx context.Context, r io.Reader) (Object, error) {
	if err := ctx.Err(); err != nil {
		return Object{}, err
	}

	tmpName, tmp, err := s.temp()
	if err != nil {
		return Object{}, err
	}
	keep := false
	defer func() {
		_ = tmp.Close()
		if !keep {
			_ = s.fs.Remove(tmpName)
		}
	}()

	hash := sha256.New()
	n, err := io.Copy(io.MultiWriter(tmp, hash), ctxReader{ctx: ctx, r: io.LimitReader(r, s.max+1)})
	if err != nil {
		return Object{}, fmt.Errorf("storage: write: %w", err)
	}
	if n == 0 {
		return Object{}, ErrEmpty
	}
	if n > s.max {
		return Object{}, ErrTooLarge
	}

	var sum [32]byte
	copy(sum[:], hash.Sum(nil))

	mt, err := sniff(tmp)
	if err != nil {
		return Object{}, err
	}
	if !allowedMIME(mt) {
		return Object{}, ErrMIME
	}
	mime := mt.String()
	width, height := imageSize(tmp, mime)

	hexSum := hex.EncodeToString(sum[:])
	key := hexSum[:2] + "/" + hexSum
	if err := s.fs.MkdirAll(hexSum[:2], 0o755); err != nil {
		return Object{}, fmt.Errorf("storage: mkdir: %w", err)
	}
	obj := Object{Key: key, Size: n, SHA256: sum, MIME: mime, Width: width, Height: height}
	if _, err := s.fs.Stat(key); err == nil {
		return obj, nil
	}

	_ = tmp.Close()
	if err := s.fs.Rename(tmpName, key); err != nil {
		return Object{}, fmt.Errorf("storage: rename: %w", err)
	}
	keep = true
	return obj, nil
}

// Open returns the stored body. The caller closes it.
func (s *Store) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !validKey(key) {
		return nil, ErrKey
	}
	f, err := s.fs.Open(key)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("storage: open: %w", err)
	}
	return f, nil
}

// Remove deletes the stored body. A missing file is not an error.
func (s *Store) Remove(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !validKey(key) {
		return ErrKey
	}
	if err := s.fs.Remove(key); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("storage: remove: %w", err)
	}
	return nil
}

// Verify re-reads the file and checks that its SHA-256 matches.
func (s *Store) Verify(ctx context.Context, key string, sum [32]byte) error {
	f, err := s.Open(ctx, key)
	if err != nil {
		return err
	}
	defer f.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, ctxReader{ctx: ctx, r: f}); err != nil {
		return fmt.Errorf("storage: verify: %w", err)
	}
	var got [32]byte
	copy(got[:], hash.Sum(nil))
	if got != sum {
		return fmt.Errorf("storage: sha256 mismatch")
	}
	return nil
}

func (s *Store) temp() (string, afero.File, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", nil, fmt.Errorf("storage: temp: %w", err)
	}
	name := "tmp/" + hex.EncodeToString(b[:])
	if err := s.fs.MkdirAll("tmp", 0o755); err != nil {
		return "", nil, fmt.Errorf("storage: temp: %w", err)
	}
	f, err := s.fs.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
	if err != nil {
		return "", nil, fmt.Errorf("storage: temp: %w", err)
	}
	return name, f, nil
}

func sniff(f afero.File) (*mimetype.MIME, error) {
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("storage: rewind: %w", err)
	}
	mt, err := mimetype.DetectReader(f)
	if err != nil {
		return nil, fmt.Errorf("storage: mime: %w", err)
	}
	return mt, nil
}

func imageSize(f afero.File, mime string) (int, int) {
	switch mime {
	case "image/jpeg", "image/png", "image/gif":
	default:
		return 0, 0
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return 0, 0
	}
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0, 0
	}
	return cfg.Width, cfg.Height
}

func allowedMIME(mt *mimetype.MIME) bool {
	for _, kind := range []string{
		"image/jpeg", "image/png", "image/gif", "image/webp",
		"video/mp4", "video/webm", "video/ogg",
		"audio/mpeg", "audio/mp4", "audio/ogg", "audio/wav", "audio/webm",
		"application/ogg", "application/pdf",
	} {
		if mt.Is(kind) {
			return true
		}
	}
	return false
}

func validKey(key string) bool {
	if len(key) != 2+1+64 || key[2] != '/' {
		return false
	}
	for i := 0; i < len(key); i++ {
		if i == 2 {
			continue
		}
		c := key[i]
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return key[:2] == key[3:5]
}

type ctxReader struct {
	ctx context.Context
	r   io.Reader
}

func (c ctxReader) Read(p []byte) (int, error) {
	if err := c.ctx.Err(); err != nil {
		return 0, err
	}
	return c.r.Read(p)
}
