package blobstore

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"
)

func ExampleMap_Usage() {
	ctx := context.TODO()
	store := NewMap()
	_ = store.Put(ctx, "foo", strings.NewReader("bar"), int64(len("bar")))
	rc, _, _ := store.Get(ctx, "foo")
	b, _ := ioutil.ReadAll(rc)
	fmt.Println(string(b))
	// "bar"
}
