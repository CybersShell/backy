package remotefetcher

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

type CacheData struct {
	Hash string `yaml:"hash"`
	Path string `yaml:"path"`
	Type string `yaml:"type"`
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
	// println("Saving cache to file:", c.file)
	c.mu.Lock()
	defer c.mu.Unlock()

	var cacheData []CacheData
	for _, data := range c.store {
		cacheData = append(cacheData, data)
	}

	data, err := yaml.Marshal(cacheData)
	if err != nil {
		return err
	}

	return os.WriteFile(c.file, data, 0644)
}

func (c *Cache) Get(hash string) ([]byte, CacheData, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	println("Getting cache data for hash:", hash)
	cacheData, exists := c.store[hash]
	if !exists {
		println("Cache data does not exist for hash:", hash)
		return nil, CacheData{}, false
	}

	data, err := os.ReadFile(cacheData.Path)
	if err != nil {
		return nil, CacheData{}, false
	}

	return data, cacheData, true
}

func (c *Cache) AddDataToStore(hash string, cacheData CacheData) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store[hash] = cacheData
}

func (c *Cache) Set(source, hash string, data []byte, dataType string) (CacheData, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	fileName := filepath.Base(source)

	path := filepath.Join(c.dir, fmt.Sprintf("%s-%s", fileName, hash))

	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.MkdirAll(c.dir, 0700)
	}

	err := os.WriteFile(path, data, 0644)
	if err != nil {
		return CacheData{}, err
	}

	cacheData := CacheData{
		Hash: hash,
		Path: path,
		Type: dataType,
	}

	c.store[hash] = cacheData

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

// Function to read and parse the hashMetadataSample.yml file
func LoadMetadataFromFile(filePath string) ([]*CacheData, error) {
	// fmt.Println("Loading metadata from file:", filePath)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// Create the file if it does not exist
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
