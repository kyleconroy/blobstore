package blobstore

import (
	"bytes"
	"context"
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
)

type s3Store struct {
	svc        *s3.S3
	bucketName string
}

func NewS3(svc *s3.S3, bucketName string) Client {
	return &s3Store{svc, bucketName}
}

func (s *s3Store) Put(ctx context.Context, key string, blob io.Reader, length int64) error {
	// aws SDK can't stream, buffer in memory
	var buf bytes.Buffer
	_, err := io.CopyN(&buf, blob, length)
	if err != nil {
		return err
	}
	_, err = s.svc.PutObjectWithContext(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucketName),
		Key:           aws.String(key),
		Body:          bytes.NewReader(buf.Bytes()),
		ContentType:   aws.String("application/octet-stream"),
		ContentLength: aws.Int64(length),
	})
	return err
}

func (s *s3Store) Get(ctx context.Context, key string) (io.ReadCloser, int64, error) {
	resp, err := s.svc.GetObjectWithContext(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, 0, err
	}
	return resp.Body, *resp.ContentLength, nil
}

func (s *s3Store) Delete(ctx context.Context, key string) error {
	_, err := s.svc.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	})
	return err
}

func (s *s3Store) Contains(ctx context.Context, key string) (bool, error) {
	_, err := s.svc.HeadObjectWithContext(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		if rf, ok := err.(awserr.RequestFailure); ok {
			// 404 error code requires ListBucket permission, otherwise you get 403:
			// https://docs.aws.amazon.com/AmazonS3/latest/API/RESTObjectHEAD.html#RESTObjectHEAD_Description
			if rf.StatusCode() == 404 {
				return false, nil
			}
		}
		return false, err
	}
	return true, nil
}
