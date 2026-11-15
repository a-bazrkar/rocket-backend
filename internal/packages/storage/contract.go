package storage

import (
	"context"
	"io"
)

// Storage is the file API the rest of the app depends on.
type Storage interface {

	// Upload a file to the storage.
	Put(ctx context.Context, r io.Reader) (Object, error)

	// Open a file from the storage.
	Open(ctx context.Context, key string) (io.ReadCloser, error)

	// Remove a file from the storage.
	Remove(ctx context.Context, key string) error

	// Verify a file from the storage.
	Verify(ctx context.Context, key string, sum [32]byte) error
}
