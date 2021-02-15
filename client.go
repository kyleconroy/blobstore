package blobstore

import (
	"context"
	"io"
)

type Client interface {
	// Return the contents at a given key. Returns an error if the key doesn't
	// exist.
	Get(ctx context.Context, key string) (io.ReadCloser, int64, error)

	// Store the contents of the input reader in the blobstore under the given key.
	Put(ctx context.Context, key string, blob io.Reader, length int64) error

	// Delete the contents stored at the given key
	Delete(ctx context.Context, key string) error

	// Returns true if the given key is already in the store. May return a
	// error if the store in unavailable
	Contains(ctx context.Context, key string) (bool, error)
}
