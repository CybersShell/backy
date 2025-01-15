package configfetcher

import (
	"errors"
	"io"
	"net/http"

	"gopkg.in/yaml.v3"
)

type HTTPFetcher struct {
	HTTPClient *http.Client
}

// NewHTTPFetcher creates a new instance of HTTPFetcher with the provided options.
func NewHTTPFetcher(options ...Option) *HTTPFetcher {
	cfg := &FetcherConfig{}
	for _, opt := range options {
		opt(cfg)
	}

	// Initialize HTTP client if not provided
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = http.DefaultClient
	}

	return &HTTPFetcher{HTTPClient: cfg.HTTPClient}
}

// Fetch retrieves the configuration from the specified URL
func (h *HTTPFetcher) Fetch(source string) ([]byte, error) {
	resp, err := http.Get(source)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to fetch remote config: " + resp.Status)
	}

	return io.ReadAll(resp.Body)
}

// Parse decodes the raw data into the provided target structure
func (h *HTTPFetcher) Parse(data []byte, target interface{}) error {
	return yaml.Unmarshal(data, target)
}
