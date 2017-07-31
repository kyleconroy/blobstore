package blobstore

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"sync"
)

// useful for testing
type Map struct {
	sync.Mutex
	Values map[string][]byte
}

func NewMap() *Map {
	return &Map{
		Values: make(map[string][]byte),
	}
}

func (m *Map) Put(key string, blob io.Reader, length int64) error {
	var buf bytes.Buffer
	_, err := io.CopyN(&buf, blob, length)
	if err != nil {
		return err
	}
	m.Lock()
	defer m.Unlock()
	m.Values[key] = buf.Bytes()
	return nil
}

func (m *Map) Delete(key string) error {
	m.Lock()
	delete(m.Values, key)
	m.Unlock()
	return nil
}

func (m *Map) Get(key string) (io.ReadCloser, int64, error) {
	m.Lock()
	defer m.Unlock()
	val, ok := m.Values[key]
	if !ok {
		return nil, 0, fmt.Errorf("Map has no key: %v", key)
	}
	return ioutil.NopCloser(bytes.NewReader(val)), int64(len(val)), nil
}

func (m *Map) Contains(key string) (bool, error) {
	m.Lock()
	defer m.Unlock()
	_, ok := m.Values[key]
	return ok, nil
}
