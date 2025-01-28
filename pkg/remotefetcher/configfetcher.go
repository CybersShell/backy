package remotefetcher

import (
	"errors"
	"strings"
)

type ConfigFetcher interface {
	// Fetch retrieves the configuration from the specified URL or source
	// Returns the raw data as bytes or an error
	Fetch(source string) ([]byte, error)

	// Parse decodes the raw data into a Go structure (e.g., Commands, CommandLists)
	// Takes the raw data as input and populates the target interface
	Parse(data []byte, target interface{}) error

	// Hash returns the hash of the configuration data
	Hash(data []byte) string
}

// ErrFileNotFound is returned when the file is not found and should be ignored
var ErrFileNotFound = errors.New("remotefetcher: file not found")

func NewConfigFetcher(source string, cache *Cache, options ...Option) (ConfigFetcher, error) {
	var fetcher ConfigFetcher
	var dataType string

	config := FetcherConfig{}
	for _, option := range options {
		option(&config)
	}
	if strings.HasPrefix(source, "http") || strings.HasPrefix(source, "https") {
		fetcher = NewHTTPFetcher(options...)
		dataType = "yaml"
	} else if strings.HasPrefix(source, "s3") {
		var err error
		fetcher, err = NewS3Fetcher(options...)
		if err != nil {
			return nil, err
		}
		dataType = "yaml"
	} else {
		fetcher = &LocalFetcher{}
		dataType = "yaml"

		return fetcher, nil
	}

	//TODO: should local files be cached?

	data, err := fetcher.Fetch(source)
	if err != nil {
		if config.IgnoreFileNotFound && isFileNotFoundError(err) {
			return nil, ErrFileNotFound
		}
		return nil, err
	}

	hash := fetcher.Hash(data)
	if cachedData, cacheMeta, exists := cache.Get(hash); exists {
		return &CachedFetcher{data: cachedData, path: cacheMeta.Path, dataType: cacheMeta.Type}, nil
	}

	cacheData, err := cache.Set(source, hash, data, dataType)
	if err != nil {
		return nil, err
	}
	return &CachedFetcher{data: data, path: cacheData.Path, dataType: cacheData.Type}, nil
}

func isFileNotFoundError(err error) bool {
	// Implement logic to check if the error is a "file not found" error
	// This can be based on the error type or message
	return strings.Contains(err.Error(), "file not found")
}
