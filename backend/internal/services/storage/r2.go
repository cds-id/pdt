package storage

import (
	"bytes"
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type R2Client struct {
	client       *s3.Client
	bucket       string
	publicDomain string
}

func NewR2Client(accountID, accessKeyID, secretAccessKey, bucketName, publicDomain string) *R2Client {
	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID)

	client := s3.New(s3.Options{
		Region:       "auto",
		BaseEndpoint: &endpoint,
		Credentials:  credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, ""),
	})

	return &R2Client{
		client:       client,
		bucket:       bucketName,
		publicDomain: publicDomain,
	}
}

// Upload uploads content to R2 and returns the public URL.
func (r *R2Client) Upload(ctx context.Context, key string, content []byte, contentType string) (string, error) {
	_, err := r.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      &r.bucket,
		Key:         &key,
		Body:        bytes.NewReader(content),
		ContentType: &contentType,
		CacheControl: aws.String("public, max-age=86400"),
	})
	if err != nil {
		return "", fmt.Errorf("r2 upload failed: %w", err)
	}

	url := fmt.Sprintf("https://%s/%s", r.publicDomain, key)
	return url, nil
}
