package blobstore

import (
	"crypto"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
)

type fsStore struct {
	path string
	hash crypto.Hash
}

func NewFileSystem(path string) (Client, error) {
	err := os.RemoveAll(path)
	if err != nil {
		return nil, err
	}

	err = os.MkdirAll(path, 0755)
	if err != nil {
		return nil, err
	}

	return &fsStore{path, crypto.SHA256}, nil
}

// Each file has a header with metadata
type fsHeader struct {
	Key    string
	Length int64
}

var fsHeaderEndianness = binary.BigEndian

func (s *fsStore) Put(key string, blob io.Reader, length int64) error {
	finalPath := s.makePath(key)
	tempPath := finalPath + "-temp"
	f, err := os.Create(tempPath)
	if err != nil {
		return err
	}
	defer os.Remove(tempPath)
	defer f.Close()

	buf, err := json.MarshalIndent(fsHeader{Key: key, Length: length}, "", "    ")
	if err != nil {
		return err
	}

	// write header
	var headerLength int64 = int64(len(buf))
	err = binary.Write(f, fsHeaderEndianness, headerLength)
	if err != nil {
		return err
	}
	_, err = f.Write(buf)
	if err != nil {
		return err
	}

	// write payload
	_, err = io.CopyN(f, blob, length)
	if err != nil {
		return err
	}

	// for windows, the rename won't succeed if the file handle is open
	f.Close()

	return os.Rename(tempPath, finalPath)
}

func (s *fsStore) Get(key string) (io.ReadCloser, int64, error) {
	f, err := os.Open(s.makePath(key))
	if err != nil {
		return nil, 0, err
	}
	var headerLength int64
	err = binary.Read(f, fsHeaderEndianness, &headerLength)
	if err != nil {
		return nil, 0, err
	}
	headerBuf := make([]byte, headerLength)
	_, err = io.ReadFull(f, headerBuf)
	if err != nil {
		return nil, 0, err
	}
	var hdr fsHeader
	err = json.Unmarshal(headerBuf, &hdr)
	if err != nil {
		return nil, 0, err
	}
	return f, hdr.Length, nil
}

func (s *fsStore) Delete(key string) error {
	return os.Remove(s.makePath(key))
}

func (s *fsStore) Contains(key string) (bool, error) {
	_, err := os.Lstat(s.makePath(key))
	switch {
	case err == nil:
		return true, nil
	case os.IsNotExist(err):
		return false, nil
	default:
		return false, err
	}
}

func (s *fsStore) makePath(key string) string {
	h := s.hash.New()
	h.Write([]byte(key))
	return filepath.Join(s.path, hex.EncodeToString(h.Sum(nil)))
}
