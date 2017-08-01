package blobstore

import (
	"fmt"
	"io/ioutil"
	"strings"
)

func ExampleMap_Usage() {
	store := NewMap()
	_ = store.Put("foo", strings.NewReader("bar"), len("bar"))
	rc, _, _ = store.Get("foo")
	b, _ := ioutil.ReadAll(rc)
	fmt.Println(string(b))
	// "bar"
}
