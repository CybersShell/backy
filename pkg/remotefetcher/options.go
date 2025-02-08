package remotefetcher

import (
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Option is a function that configures a fetcher.
type FetcherOption func(*FetcherConfig)

// FetcherConfig holds the configuration for a fetcher.
type FetcherConfig struct {
	S3Client           *s3.Client
	HTTPClient         *http.Client
	FileType           string
	IgnoreFileNotFound bool
}

// WithS3Client sets the S3 client for the fetcher.
func WithS3Client(client *s3.Client) FetcherOption {
	return func(cfg *FetcherConfig) {
		cfg.S3Client = client
	}
}

// WithHTTPClient sets the HTTP client for the fetcher.
func WithHTTPClient(client *http.Client) FetcherOption {
	return func(cfg *FetcherConfig) {
		cfg.HTTPClient = client
	}
}

func IgnoreFileNotFound() FetcherOption {
	return func(cfg *FetcherConfig) {
		cfg.IgnoreFileNotFound = true
	}
}

// WithFileType ensures the default FileType will be yaml
func WithFileType(fileType string) FetcherOption {
	return func(cfg *FetcherConfig) {
		cfg.FileType = fileType
		if strings.TrimSpace(fileType) == "" {
			cfg.FileType = "yaml"
		}
	}
}
