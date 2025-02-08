package remotefetcher

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/mitchellh/go-homedir"
	"gopkg.in/yaml.v3"
)

type S3Fetcher struct {
	S3Client *minio.Client
	config   FetcherConfig
}

// NewS3Fetcher creates a new instance of S3Fetcher with the provided options.
func NewS3Fetcher(endpoint string, options ...FetcherOption) (*S3Fetcher, error) {
	cfg := &FetcherConfig{}
	var s3Client *minio.Client
	var err error
	for _, opt := range options {
		opt(cfg)
	}

	/*
		options for S3 urls:
			1. s3://bucket.region.endpoint.tld/path/to/object
			2. alias with path and rest is looked up in file - add FetcherOptions


		options for S3 credentials:
			1. from file ($HOME/.aws/credentials)
			2. env vars (AWS_SECRET_KEY, etc.)
	*/

	s3Endpoint := os.Getenv("S3_ENDPOINT")
	creds, err := getS3Credentials("default", s3Endpoint, cfg.HTTPClient)
	if err != nil {
		println(err.Error())
		return nil, err
	}
	// Initialize S3 client if not provided
	if cfg.S3Client == nil {
		s3Client, err = minio.New(s3Endpoint, &minio.Options{
			Creds:  creds,
			Secure: true,
		})

		if err != nil {
			return nil, err
		}

	}

	return &S3Fetcher{S3Client: s3Client, config: *cfg}, nil
}

// Fetch retrieves the configuration from an S3 bucket
// Source should be in the format "bucket-name/object-key"
func (s *S3Fetcher) Fetch(source string) ([]byte, error) {
	bucket, object, err := parseS3Source(source)
	if err != nil {
		return nil, err
	}

	doesObjectExist, objErr := objectExists(bucket, object, s.S3Client)
	if !doesObjectExist {
		if objErr != nil {
			return nil, err
		}

		if s.config.IgnoreFileNotFound {
			return nil, ErrIgnoreFileNotFound
		}
	}

	fileObject, err := s.S3Client.GetObject(context.TODO(), bucket, object, minio.GetObjectOptions{})
	if err != nil {
		println(err.Error())

		return nil, err
	}
	defer fileObject.Close()
	fileObjectStats, statErr := fileObject.Stat()
	if statErr != nil {
		return nil, statErr
	}
	buffer := make([]byte, fileObjectStats.Size)

	// Read the object into the buffer
	_, err = io.ReadFull(fileObject, buffer)
	if err != nil {
		return nil, err
	}

	return buffer, nil
}

// Parse decodes the raw data into the provided target structure
func (s *S3Fetcher) Parse(data []byte, target interface{}) error {
	return yaml.Unmarshal(data, target)
}

// Helper function to parse S3 source into bucket and key
func parseS3Source(source string) (bucket, key string, err error) {
	parts := strings.SplitN(source, "/", 2)
	if len(parts) != 2 {
		return "", "", errors.New("invalid S3 source format, expected bucket-name/object-key")
	}
	u, _ := url.Parse(source)
	u.Path = strings.TrimPrefix(u.Path, "/")
	return u.Host, u.Path, nil
}

func (s *S3Fetcher) Hash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func getS3Credentials(profile, host string, httpClient *http.Client) (*credentials.Credentials, error) {
	// println(s3utils.GetRegionFromURL(*u))
	homeDir, hdirErr := homedir.Dir()
	if hdirErr != nil {
		return nil, hdirErr
	}
	s3Creds := credentials.NewFileAWSCredentials(path.Join(homeDir, ".aws", "credentials"), "default")
	credVals, credErr := s3Creds.GetWithContext(&credentials.CredContext{Endpoint: host, Client: httpClient})
	if credErr != nil {
		return nil, credErr
	}
	creds := credentials.NewStaticV4(credVals.AccessKeyID, credVals.SecretAccessKey, "")
	return creds, nil
}

var (
	doesNotExist = "The specified key does not exist."
)

// objectExists checks for name in bucket using client.
// It returns false and nil if the key does not exist
func objectExists(bucket, name string, client *minio.Client) (bool, error) {
	_, err := client.StatObject(context.TODO(), bucket, name, minio.StatObjectOptions{})
	if err != nil {
		switch err.Error() {
		case doesNotExist:
			return false, nil
		default:
			return false, errors.Join(err, errors.New("error stating object"))
		}
	}
	return true, nil
}
