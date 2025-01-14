package configfetcher

import (
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

type LocalFetcher struct{}

// Fetch retrieves the configuration from the specified local file path
func (l *LocalFetcher) Fetch(source string) ([]byte, error) {
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
