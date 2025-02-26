package remotefetcher

import (
	"errors"
	"strings"
)

type RemoteFetcher interface {
	// Fetch retrieves the configuration from the specified URL or source
	// Returns the raw data as bytes or an error
	Fetch(source string) ([]byte, error)

	// Parse decodes the raw data into a Go structure (e.g., Commands, CommandLists)
	// Takes the raw data as input and populates the target interface
	Parse(data []byte, target interface{}) error

	// Hash returns the hash of the configuration data
	Hash(data []byte) string
}

// ErrIgnoreFileNotFound is returned when the file is not found and should be ignored
var ErrIgnoreFileNotFound = errors.New("remotefetcher: file not found")

func NewRemoteFetcher(source string, cache *Cache, options ...FetcherOption) (RemoteFetcher, error) {
	var fetcher RemoteFetcher

	config := FetcherConfig{}
	for _, option := range options {
		option(&config)
	}

	// If FileType is empty (i.e. WithFileType was not called), yaml is the default file type
	if strings.TrimSpace(config.FileType) == "" {
		config.FileType = "yaml"
	}
	if strings.HasPrefix(source, "http") || strings.HasPrefix(source, "https") {
		fetcher = NewHTTPFetcher(options...)
	} else if strings.HasPrefix(source, "s3") {
		var err error
		fetcher, err = NewS3Fetcher(source, options...)
		if err != nil {
			return nil, err
		}
	} else {
		fetcher = &LocalFetcher{}

		return fetcher, nil
	}

	//TODO: should local files be cached?

	data, err := fetcher.Fetch(source)
	if err != nil {
		if config.IgnoreFileNotFound && isFileNotFoundError(err) {
			return nil, ErrIgnoreFileNotFound
		}
		return nil, err
	}

	URLHash := HashURL(source)
	if cachedData, cacheMeta, exists := cache.Get(URLHash); exists {
		println(cachedData)
		return &CachedFetcher{data: cachedData, path: cacheMeta.Path, dataType: cacheMeta.Type}, nil
	}

	hash := fetcher.Hash(data)
	cacheData, err := cache.Set(source, hash, data, config.FileType)
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
