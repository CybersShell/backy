package remotefetcher

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

type LocalFetcher struct {
	config FetcherConfig
}

// Fetch retrieves the file from the specified local file path
func (l *LocalFetcher) Fetch(source string) ([]byte, error) {
	// Check if the file exists
	if _, err := os.Stat(source); os.IsNotExist(err) {
		if l.config.IgnoreFileNotFound {
			return nil, ErrIgnoreFileNotFound
		}
		return nil, nil
	}
	file, err := os.Open(source)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return io.ReadAll(file)
}

// Parse decodes the raw data into the provided target structure
func (l *LocalFetcher) Parse(data []byte, target interface{}) error {
	return yaml.Unmarshal(data, target)
}

func (l *LocalFetcher) Hash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
