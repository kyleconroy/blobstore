package blobstore

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"testing"
)

func TestFS(t *testing.T) {
	var blob = []byte{0x1, 0x2, 0x3, 0x4}
	blobLength := int64(len(blob))

	fs, err := NewFileSystem("TestFS")
	if err != nil {
		t.Fatalf("Failed to create FS blob store: %v", err)
	}
	defer os.RemoveAll("TestFS")

	if err := fs.Put("/foo", bytes.NewReader(blob), blobLength); err != nil {
		t.Fatalf("Failed to put blob: %v", err)
	}

	rd, length, err := fs.Get("/foo")
	if err != nil {
		t.Fatalf("Failed to get blob: %v", err)
	}
	defer rd.Close()

	got, err := ioutil.ReadAll(rd)
	if err != nil {
		t.Fatalf("Failed to read returned blob: %v", err)
	}

	Equal(t, blob, got)
	Equal(t, length, blobLength)
}

type controlReader struct {
	io.Reader
	c chan struct{}
}

func (r *controlReader) Read(p []byte) (int, error) {
	_, ok := <-r.c
	if ok {
		// read one byte at a time
		return r.Reader.Read(p[:1])
	} else {
		// after channel close, read as fast as possible
		return r.Reader.Read(p)
	}
}

func (r *controlReader) Advance() {
	r.c <- struct{}{}
}

func (r *controlReader) Uncork() {
	close(r.c)
}

func TestNoPartial(t *testing.T) {
	var blob = []byte{0x1, 0x2, 0x3, 0x4}
	blobLength := int64(len(blob))

	fs, err := NewFileSystem("TestNoPartial")
	if err != nil {
		t.Fatalf("Failed to create FS blob store: %v", err)
	}
	defer os.RemoveAll("TestNoPartial")

	control := controlReader{bytes.NewReader(blob), make(chan struct{})}
	done := make(chan struct{})
	go func() {
		if err := fs.Put("/foo", &control, blobLength); err != nil {
			t.Errorf("Failed to put blob: %v", err)
		}
		close(done)
	}()

	// allow two one byte reads
	control.Advance()
	control.Advance()

	rd, _, err := fs.Get("/foo")
	if err == nil {
		rd.Close()
		t.Fatalf("Got partial blob while Put operation was still in progress!")
	}

	// let the Put finish
	control.Uncork()
	<-done

	rd, length, err := fs.Get("/foo")
	if err != nil {
		t.Fatalf("Failed to get blob: %v", err)
	}
	Equal(t, length, blobLength)
	defer rd.Close()
}
