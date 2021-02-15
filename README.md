# stakmachine/blobstore
[![GoDoc](https://godoc.org/stackmachine.com/blobstore?status.svg)](https://godoc.org/stackmachine.com/blobstore) [![Build Status](https://travis-ci.org/stackmachine/blobstore.svg?branch=master)](https://travis-ci.org/stackmachine/blobstore)

## Install

This repository does not include a vendor directory, so you'll need to use
`dep` to manage your dependencies.

```
dep ensure github.com/stackmachine/blobstore
```

## Usage

```go
package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/stackmachine/blobstore"
)

func main() {
	// Create a ctx
	ctx := context.Background()

	// Create a store backed by an S3 bucket
	sess := session.Must(session.NewSession())
	bucket := blobstore.NewS3(s3.New(session), "example-bucket-name")

	// Create another store backed by a local folder
	fs, _ := blobstore.NewFileSystem("cas")

	// Limit the size of this folder to 500MB
	lru := blobstore.LRU(int64(500)*1e+6, fs)

	// Sychnorize access to the LRU store
	cache := blobstore.NewSynchronized(lru)

	// Use the filesystem to cache the S3 bucket
	store := blobstore.Cached(main, cache)

	// Start all keys with a shared prefix
	final := blobstore.Prefixed("prefix", store)

	// Check if a key exists; outputs "false"
	fmt.Println(final.Contains(ctx, "key"))
}
```
