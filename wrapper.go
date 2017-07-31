package blobstore

import (
	"bytes"
	"container/list"
	"io"
	"io/ioutil"
	"log"
	"path"
	"sync"
)

type lru struct {
	maxSize     int64
	currentSize int64
	ll          *list.List
	lookup      map[string]*list.Element

	Client
}

// Create a new LRU cache. Size is in bytes
func LRU(size int64, i Client) Client {
	return &lru{
		maxSize: size,
		lookup:  make(map[string]*list.Element),
		ll:      list.New(),
		Client:  i,
	}
}

type entry struct {
	key  string
	size int64
}

func (l *lru) Put(key string, blob io.Reader, length int64) error {
	if err := l.Client.Put(key, blob, length); err != nil {
		return err
	}

	if ee, ok := l.lookup[key]; ok {
		l.ll.MoveToFront(ee)
		return nil
	} else {
		ele := l.ll.PushFront(entry{key, length})
		l.currentSize += length
		l.lookup[key] = ele
		for l.maxSize != 0 && l.currentSize > l.maxSize {
			if ele := l.ll.Back(); ele != nil {
				l.Delete(ele.Value.(entry).key)
			}
		}
	}
	return nil
}

func (l *lru) Get(key string) (io.ReadCloser, int64, error) {
	if ele, hit := l.lookup[key]; hit {
		l.ll.MoveToFront(ele)
	}
	return l.Client.Get(key)
}

func (l *lru) Delete(key string) error {
	if ele, hit := l.lookup[key]; hit {
		l.ll.Remove(ele)
		l.currentSize -= ele.Value.(entry).size
		delete(l.lookup, key)
	}
	return l.Client.Delete(key)
}

type synchronized struct {
	mu sync.RWMutex
	Client
}

func NewSynchronized(c Client) Client {
	return &synchronized{Client: c}
}

func (s *synchronized) Put(key string, blob io.Reader, length int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Client.Put(key, blob, length)
}

func (s *synchronized) Get(key string) (io.ReadCloser, int64, error) {
	// Note the use of `Lock` instead of `RLock`. This is intentional.
	//
	// The LRU wrapper mutates internal state on every get call. Using RLock
	// led to panics due to concurrent map and list access.
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Client.Get(key)
}

func (s *synchronized) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Client.Delete(key)
}

func (s *synchronized) Contains(key string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Client.Contains(key)
}

type prefixed struct {
	prefix string
	Client
}

func Prefixed(prefix string, i Client) Client {
	return &prefixed{prefix, i}
}

func (p *prefixed) Put(key string, blob io.Reader, length int64) error {
	return p.Client.Put(path.Join("/", p.prefix, key), blob, length)
}

func (p *prefixed) Get(key string) (io.ReadCloser, int64, error) {
	return p.Client.Get(path.Join("/", p.prefix, key))
}

func (p *prefixed) Delete(key string) error {
	return p.Client.Delete(path.Join("/", p.prefix, key))
}

func (p *prefixed) Contains(key string) (bool, error) {
	return p.Client.Contains(path.Join("/", p.prefix, key))
}

type cached struct {
	authority Client
	caches    []Client
	signal    chan struct{} // for testing only
}

func Cached(authority Client, caches ...Client) Client {
	return &cached{authority, caches, nil}
}

func (c *cached) Put(key string, blob io.Reader, length int64) error {
	blob = c.putCache(key, ioutil.NopCloser(blob), length)

	err := c.authority.Put(key, blob, length)
	if err != nil {
		return err
	}
	return nil
}

func (c *cached) Get(key string) (io.ReadCloser, int64, error) {
	for _, cache := range c.caches {
		rd, length, err := cache.Get(key)
		if err == nil {
			return rd, length, nil
		}
	}
	rd, length, err := c.authority.Get(key)
	if err != nil {
		return nil, 0, err
	}
	rd = c.putCache(key, rd, length)
	return rd, length, nil
}

func (c *cached) Delete(key string) error {
	for _, cache := range c.caches {
		// Deletion from a cache is best-effort, ignore failures
		cache.Delete(key)
	}
	return c.authority.Delete(key)
}

func (c *cached) Contains(key string) (bool, error) {
	return c.authority.Contains(key)
}

func (c *cached) putCache(key string, r io.ReadCloser, length int64) io.ReadCloser {
	return &teeCacher{r: r, c: c, key: key, length: length}
}

type teeCacher struct {
	r io.ReadCloser
	c *cached

	key    string
	length int64
	buf    bytes.Buffer
	once   sync.Once
}

func (t *teeCacher) Read(p []byte) (n int, err error) {
	n, err = t.r.Read(p)
	if n > 0 {
		if n, err := t.buf.Write(p[:n]); err != nil {
			return n, err
		}
	}
	if int64(t.buf.Len()) == t.length {
		t.put()
	}
	return
}

func (t *teeCacher) Close() error {
	return t.r.Close()
}

func (t *teeCacher) put() {
	t.once.Do(func() {
		go func() {
			defer t.signal()
			for _, cache := range t.c.caches {
				err := cache.Put(t.key, bytes.NewReader(t.buf.Bytes()), t.length)
				if err != nil {
					log.Printf("Failed to write to cache key: %s err: %s", t.key, err)
				}
			}
		}()
	})
}

// signal is only set in testing. it notifies a client of when
// the teeCacher has finished putting a value into its cache
func (t *teeCacher) signal() {
	if t.c.signal != nil {
		t.c.signal <- struct{}{}
	}
}
