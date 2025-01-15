package configfetcher

import (
	"bytes"
	"context"
	"errors"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"gopkg.in/yaml.v3"
)

type S3Fetcher struct {
	S3Client *s3.Client
}

// NewS3Fetcher creates a new instance of S3Fetcher with the provided options.
func NewS3Fetcher(options ...Option) (*S3Fetcher, error) {
	cfg := &FetcherConfig{}
	for _, opt := range options {
		opt(cfg)
	}

	// Initialize S3 client if not provided
	if cfg.S3Client == nil {
		awsCfg, err := config.LoadDefaultConfig(context.TODO())
		if err != nil {
			return nil, err
		}
		cfg.S3Client = s3.NewFromConfig(awsCfg)
	}

	return &S3Fetcher{S3Client: cfg.S3Client}, nil
}

// Fetch retrieves the configuration from an S3 bucket
// Source should be in the format "bucket-name/object-key"
func (s *S3Fetcher) Fetch(source string) ([]byte, error) {
	bucket, key, err := parseS3Source(source)
	if err != nil {
		return nil, err
	}

	resp, err := s.S3Client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Parse decodes the raw data into the provided target structure
func (s *S3Fetcher) Parse(data []byte, target interface{}) error {
	return yaml.Unmarshal(data, target)
}

// Helper function to parse S3 source into bucket and key
func parseS3Source(source string) (bucket, key string, err error) {
	parts := strings.SplitN(source, "/", 2)
	if len(parts) != 2 {
		return "", "", errors.New("invalid S3 source format, expected bucket-name/object-key")
	}
	return parts[0], parts[1], nil
}
