package remotefetcher

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"

	"gopkg.in/yaml.v3"
)

type HTTPFetcher struct {
	HTTPClient *http.Client
	config     FetcherConfig
}

// NewHTTPFetcher creates a new instance of HTTPFetcher with the provided options.
func NewHTTPFetcher(options ...FetcherOption) *HTTPFetcher {
	cfg := &FetcherConfig{}
	for _, opt := range options {
		opt(cfg)
	}

	// Initialize HTTP client if not provided
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = http.DefaultClient
	}

	return &HTTPFetcher{HTTPClient: cfg.HTTPClient, config: *cfg}
}

// Fetch retrieves the file from the specified source URL
func (h *HTTPFetcher) Fetch(source string) ([]byte, error) {
	resp, err := http.Get(source)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound && h.config.IgnoreFileNotFound {
		return nil, ErrIgnoreFileNotFound
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to fetch remote config: " + resp.Status)
	}

	return io.ReadAll(resp.Body)
}

// Parse decodes the raw data into the provided target structure
func (h *HTTPFetcher) Parse(data []byte, target interface{}) error {
	return yaml.Unmarshal(data, target)
}

func (h *HTTPFetcher) Hash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
