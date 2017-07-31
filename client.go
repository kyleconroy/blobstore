package blobstore

import "io"

type Client interface {
	Get(key string) (io.ReadCloser, int64, error)
	Put(key string, blob io.Reader, length int64) error
	Delete(key string) error
	Contains(key string) (bool, error)
}
