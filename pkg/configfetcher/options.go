package configfetcher

import (
	"net/http"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Option is a function that configures a fetcher.
type Option func(*FetcherConfig)

// FetcherConfig holds the configuration for a fetcher.
type FetcherConfig struct {
	S3Client   *s3.Client
	HTTPClient *http.Client
}

// WithS3Client sets the S3 client for the fetcher.
func WithS3Client(client *s3.Client) Option {
	return func(cfg *FetcherConfig) {
		cfg.S3Client = client
	}
}

// WithHTTPClient sets the HTTP client for the fetcher.
func WithHTTPClient(client *http.Client) Option {
	return func(cfg *FetcherConfig) {
		cfg.HTTPClient = client
	}
}
