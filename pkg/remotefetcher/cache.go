package remotefetcher

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

type CacheData struct {
	Hash string `yaml:"hash"`
	Path string `yaml:"path"`
	Type string `yaml:"type"`
	URL  string `yaml:"url"`
}

type Cache struct {
	mu    sync.Mutex
	store map[string]CacheData
	file  string
	dir   string
}

func NewCache(file, dir string) (*Cache, error) {
	cache := &Cache{
		store: make(map[string]CacheData),
		file:  file,
		dir:   dir,
	}
	err := cache.loadFromFile()
	if err != nil {
		return nil, err
	}
	return cache, nil
}

func (c *Cache) loadFromFile() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, err := os.Stat(c.file); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(c.file)
	if err != nil {
		return err
	}

	var cacheData []CacheData
	err = yaml.Unmarshal(data, &cacheData)
	if err != nil {
		return err
	}

	for _, item := range cacheData {
		c.store[item.Hash] = item
	}

	return nil
}

func (c *Cache) saveToFile() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var cacheData []CacheData
	for _, data := range c.store {
		cacheData = append(cacheData, data)
	}
	cacheData = unique(cacheData)
	data, err := yaml.Marshal(cacheData)
	if err != nil {
		return err
	}

	return os.WriteFile(c.file, data, 0644)
}

func (c *Cache) Get(hash string) ([]byte, CacheData, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	cacheData, exists := c.store[hash]
	if !exists {
		return nil, CacheData{}, false
	}

	data, err := os.ReadFile(cacheData.Path)
	if err != nil {
		return nil, CacheData{}, false
	}

	return data, cacheData, true
}

func (c *Cache) AddDataToStore(hash string, cacheData CacheData) error {
	c.store[hash] = cacheData
	return c.saveToFile()
}

// Set stores data on disk and in the cache file and returns the cache data
// The filepath of the data is the file name + a SHA256 hash of the URL
func (c *Cache) Set(source, hash string, data []byte, dataType string) (CacheData, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	sourceHash := HashURL(source)

	fileName := filepath.Base(source)

	path := filepath.Join(c.dir, fmt.Sprintf("%s-%s", fileName, sourceHash))

	if _, err := os.Stat(path); os.IsNotExist(err) {
		_ = os.MkdirAll(c.dir, 0700)
	}

	err := os.WriteFile(path, data, 0644)
	if err != nil {
		return CacheData{}, err
	}

	cacheData := CacheData{
		Hash: hash,
		Path: path,
		Type: dataType,
		URL:  sourceHash,
	}

	c.store[sourceHash] = cacheData

	// Unlock before calling saveToFile to avoid double-locking
	c.mu.Unlock()
	err = c.saveToFile()
	c.mu.Lock()
	if err != nil {
		return CacheData{}, err
	}

	// fmt.Printf("Cache data: %v", cacheData)
	return cacheData, nil
}

type CachedFetcher struct {
	data     []byte
	path     string
	dataType string
}

func (cf *CachedFetcher) Fetch(source string) ([]byte, error) {
	return cf.data, nil
}

func (cf *CachedFetcher) Parse(data []byte, target interface{}) error {
	if cf.dataType == "yaml" {
		return yaml.Unmarshal(data, target)
	}
	return errors.New("parse not supported on cached fetcher for scripts")
}

func (cf *CachedFetcher) Hash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// Function to read and parse the metadata file
func LoadMetadataFromFile(filePath string) ([]*CacheData, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// Create the file if it does not exist
		_ = os.MkdirAll(path.Dir(filePath), 0700)
		emptyData := []byte("[]")
		err := os.WriteFile(filePath, emptyData, 0644)
		if err != nil {
			return nil, err
		}
		return nil, nil
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var cacheData []*CacheData
	err = yaml.Unmarshal(data, &cacheData)

	if err != nil {
		return nil, err
	}

	return cacheData, nil
}

func HashURL(url string) string {
	hash := sha256.Sum256([]byte(url))
	return hex.EncodeToString(hash[:])
}

func unique(cache []CacheData) []CacheData {
	var unique []CacheData
	type key struct{ value1, value2, value3, value4 string }
	m := make(map[key]int)
	for _, v := range cache {
		k := key{v.URL, v.Hash, v.Path, v.Type}
		if i, ok := m[k]; ok {
			// Overwrite previous value per requirement in
			// question to keep last matching value.
			unique[i] = v
		} else {
			// Unique key found. Record position and collect
			// in result.
			m[k] = len(unique)
			unique = append(unique, v)
		}
	}
	return unique
}
