package storage

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/spf13/afero"
)

// 1x1 PNG.
var png1x1 = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
	0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
	0xde, 0x00, 0x00, 0x00, 0x0c, 0x49, 0x44, 0x41,
	0x54, 0x08, 0xd7, 0x63, 0xf8, 0xcf, 0xc0, 0x00,
	0x00, 0x00, 0x03, 0x00, 0x01, 0x00, 0x05, 0xfe,
	0xd4, 0xef, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45,
	0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
}

func TestPutStoresAnyFile(t *testing.T) {
	store := newStore(afero.NewMemMapFs(), 1024)
	ctx := context.Background()

	text, err := store.Put(ctx, bytes.NewReader([]byte("hello this is plain text")))
	if err != nil {
		t.Fatal(err)
	}
	if text.MIME != "text/plain; charset=utf-8" || text.Width != 0 || text.Height != 0 {
		t.Fatalf("text = %+v", text)
	}

	raw, err := store.Put(ctx, bytes.NewReader([]byte{0x00, 0x01, 0x02, 0x03, 0xff, 0xfe}))
	if err != nil {
		t.Fatal(err)
	}
	if raw.MIME != "application/octet-stream" {
		t.Fatalf("raw mime = %s", raw.MIME)
	}
}

func TestPutPNGDimensions(t *testing.T) {
	store := newStore(afero.NewMemMapFs(), 1024)
	obj, err := store.Put(context.Background(), bytes.NewReader(png1x1))
	if err != nil {
		t.Fatal(err)
	}
	if obj.MIME != "image/png" || obj.Width != 1 || obj.Height != 1 {
		t.Fatalf("object = %+v", obj)
	}
}

func TestPutRejectsEmptyAndTooLarge(t *testing.T) {
	ctx := context.Background()
	store := newStore(afero.NewMemMapFs(), 8)

	if _, err := store.Put(ctx, bytes.NewReader(nil)); !errors.Is(err, ErrEmpty) {
		t.Fatalf("empty: %v", err)
	}
	if _, err := store.Put(ctx, bytes.NewReader(bytes.Repeat([]byte{0xab}, 9))); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("too large: %v", err)
	}
}
