package blobstore

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"sync"
	"testing"
	"time"
)

func TestSync(t *testing.T) {
	m := NewMap()
	c := NewSynchronized(m)
	var wg sync.WaitGroup
	wg.Add(2)
	client := func(ns string) {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			key := fmt.Sprintf("/%s/%d", ns, i)
			val := fmt.Sprintf("val%d", i)
			if err := c.Put(key, bytes.NewReader([]byte(val)), int64(len(val))); err != nil {
				t.Fatal(err)
			}
			exists, err := c.Contains(key)
			switch {
			case err != nil:
				t.Fatal(err)
			case !exists:
				t.Fatal("key should exist")
			}
			rd, _, err := c.Get(key)
			if err != nil {
				t.Fatal(err)
			}
			getVal, err := ioutil.ReadAll(rd)
			switch {
			case err != nil:
				t.Fatal(err)
			case val != string(getVal):
				t.Fatal("got wrong value")
			}
			rd.Close()
			if err := c.Delete(key); err != nil {
				t.Fatal(err)
			}
		}
	}
	go client("foo")
	go client("bar")
	wg.Wait()
	if len(m.Values) != 0 {
		t.Fatal("expected test to finish with zero values")
	}
}

func TestLRU(t *testing.T) {
	auth := NewMap()
	lru := LRU(3, auth)
	b := []byte{0x1, 0x2}

	err := lru.Put("/foo", bytes.NewReader(b), 2)
	Equal(t, err, nil)

	err = lru.Put("/bar", bytes.NewReader(b), 2)
	Equal(t, err, nil)

	Equal(t, len(auth.Values["/foo"]), 0)
	Equal(t, len(auth.Values["/bar"]), 2)
}

func TestPutCache(t *testing.T) {
	auth := NewMap()
	cache := NewMap()
	b := []byte{0x1, 0x2}

	store, signal := newCached(auth, cache)
	err := store.Put("/foo", bytes.NewReader(b), 2)
	Equal(t, err, nil)

	wait(signal)
	Equal(t, cache.Values["/foo"], b)
}

func TestGetCache(t *testing.T) {
	t.Parallel()
	auth := NewMap()
	cache := NewMap()
	b := []byte{0x1, 0x2}
	auth.Values["/foo"] = b

	store, signal := newCached(auth, cache)
	rd, length, err := store.Get("/foo")
	Equal(t, err, nil)
	Equal(t, length, int64(2))

	val, _ := ioutil.ReadAll(rd)
	Equal(t, val, b)

	wait(signal)
	Equal(t, cache.Values["/foo"], b)
}

func TestPrematurePutClose(t *testing.T) {
	t.Parallel()
	auth := NewMap()
	cache := NewMap()
	b := []byte{0x1, 0x2}

	store, signal := newCached(auth, cache)
	err := store.Put("/foo", bytes.NewReader(b), 3)
	NotEqual(t, err, nil)

	wait(signal)
	_, ok := cache.Values["/foo"]
	Equal(t, ok, false)
}

func TestPrematureGetClose(t *testing.T) {
	t.Parallel()
	auth := NewMap()
	cache := NewMap()
	b := []byte{0x1, 0x2}
	auth.Values["/foo"] = b

	store, signal := newCached(auth, cache)
	rd, length, err := store.Get("/foo")
	Equal(t, err, nil)
	Equal(t, length, int64(2))

	rd.Close()
	wait(signal)
	_, ok := cache.Values["/foo"]
	Equal(t, ok, false)
}

func newCached(auth Client, caches ...Client) (Client, chan struct{}) {
	c := Cached(auth, caches...).(*cached)
	c.signal = make(chan struct{}, 1)
	return c, c.signal
}

func wait(signal chan struct{}) {
	select {
	case <-signal:
	case <-time.After(time.Second):
	}
}
