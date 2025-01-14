package configfetcher

import "strings"

type ConfigFetcher interface {
	// Fetch retrieves the configuration from the specified URL or source
	// Returns the raw data as bytes or an error
	Fetch(source string) ([]byte, error)

	// Parse decodes the raw data into a Go structure (e.g., Commands, CommandLists)
	// Takes the raw data as input and populates the target interface
	Parse(data []byte, target interface{}) error
}

func NewConfigFetcher(source string) ConfigFetcher {
	if strings.HasPrefix(source, "http") || strings.HasPrefix(source, "https") {
		return &HTTPFetcher{}
	} else if strings.HasPrefix(source, "s3") {
		return &S3Fetcher{}
	} else {
		return &LocalFetcher{}
	}

}
