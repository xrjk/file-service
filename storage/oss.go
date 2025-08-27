package storage

import (
	"context"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

// OSSStorage implements the Storage interface for Aliyun OSS
type OSSStorage struct {
	client *oss.Client
}

// NewOSSStorage creates a new OSS storage instance
func NewOSSStorage(endpoint, accessKey, secretKey string, useSSL bool) (*OSSStorage, error) {
	// 根据useSSL参数决定是否使用HTTPS
	options := []oss.ClientOption{}
	if !useSSL {
		options = append(options, oss.HTTPClient(new(http.Client)))
	}
	
	client, err := oss.New(endpoint, accessKey, secretKey, options...)
	if err != nil {
		return nil, err
	}

	return &OSSStorage{
		client: client,
	}, nil
}

// Upload uploads a file to OSS
func (o *OSSStorage) Upload(ctx context.Context, bucketName, objectName string, reader io.Reader, size int64, contentType string) error {
	bucket, err := o.client.Bucket(bucketName)
	if err != nil {
		return err
	}

	// Convert context to options
	var options []oss.Option
	if contentType != "" {
		options = append(options, oss.ContentType(contentType))
	}

	return bucket.PutObject(objectName, reader, options...)
}

// Download downloads a file from OSS
func (o *OSSStorage) Download(ctx context.Context, bucketName, objectName string) (io.ReadCloser, error) {
	bucket, err := o.client.Bucket(bucketName)
	if err != nil {
		return nil, err
	}
	
	return bucket.GetObject(objectName)
}

// Delete deletes a file from OSS
func (o *OSSStorage) Delete(ctx context.Context, bucketName, objectName string) error {
	bucket, err := o.client.Bucket(bucketName)
	if err != nil {
		return err
	}
	
	return bucket.DeleteObject(objectName)
}

// List lists objects in a bucket with the given prefix
func (o *OSSStorage) List(ctx context.Context, bucket string, prefix string) ([]FileObject, error) {
	bucketClient, err := o.client.Bucket(bucket)
	if err != nil {
		return nil, err
	}
	
	// List all objects with the given prefix
	lsRes, err := bucketClient.ListObjects(oss.Prefix(prefix), oss.MaxKeys(1000))
	if err != nil {
		return nil, err
	}
	
	var objects []FileObject
	
	// Add files to the result
	for _, object := range lsRes.Objects {
		objects = append(objects, FileObject{
			Name:         object.Key,
			Size:         object.Size,
			ContentType:  object.Type,
			LastModified: object.LastModified.Format(time.RFC3339),
			Metadata:     make(map[string]string), // 暂时使用空的元数据
		})
	}
	
	return objects, nil
}

// ListObjects lists objects in a bucket with the given prefix
func (o *OSSStorage) ListObjects(ctx context.Context, bucketName, prefix string) ([]FileObject, error) {
	bucket, err := o.client.Bucket(bucketName)
	if err != nil {
		return nil, err
	}
	
	// Use delimiter to separate folders and files
	lsRes, err := bucket.ListObjects(oss.Prefix(prefix), oss.Delimiter("/"))
	if err != nil {
		return nil, err
	}
	
	var objects []FileObject
	
	// Add folders to the result
	for _, prefix := range lsRes.CommonPrefixes {
		objects = append(objects, FileObject{
			Name:        prefix,
			Size:        0,
			ContentType: "application/directory",
			IsDir:       true,
		})
	}
	
	// Add files to the result
	for _, object := range lsRes.Objects {
		objects = append(objects, FileObject{
			Name:         object.Key,
			Size:         object.Size,
			ContentType:  object.Type,
			LastModified: object.LastModified.Format(time.RFC3339),
			Metadata:     make(map[string]string), // 暂时使用空的元数据
			IsDir:        false,
		})
	}
	
	return objects, nil
}

// GetObjectInfo gets object metadata from OSS
func (o *OSSStorage) GetObjectInfo(ctx context.Context, bucketName, objectName string) (*FileObject, error) {
	bucket, err := o.client.Bucket(bucketName)
	if err != nil {
		return nil, err
	}
	
	// Get object properties
	props, err := bucket.GetObjectDetailedMeta(objectName)
	if err != nil {
		return nil, err
	}
	
	contentLength, _ := strconv.ParseInt(props.Get("Content-Length"), 10, 64)
	
	// Convert http.Header to map[string]string
	metadata := make(map[string]string)
	for k, v := range props {
		if len(v) > 0 {
			metadata[k] = v[0]
		}
	}
	
	return &FileObject{
		Name:         objectName,
		Size:         contentLength,
		ContentType:  props.Get("Content-Type"),
		LastModified: props.Get("Last-Modified"),
		Metadata:     metadata,
	}, nil
}

// CreateDirectory creates a directory in the storage
func (o *OSSStorage) CreateDirectory(ctx context.Context, bucket, objectName string) error {
	bucketClient, err := o.client.Bucket(bucket)
	if err != nil {
		return err
	}
	
	// Ensure the object name ends with "/"
	if !strings.HasSuffix(objectName, "/") {
		objectName += "/"
	}
	
	// Create an empty object to represent the directory
	return bucketClient.PutObject(objectName, strings.NewReader(""), oss.ContentType("application/directory"))
}

// ListDirectories lists directories in a bucket with the given prefix
func (o *OSSStorage) ListDirectories(ctx context.Context, bucket, prefix string) ([]FileObject, error) {
	bucketClient, err := o.client.Bucket(bucket)
	if err != nil {
		return nil, err
	}
	
	marker := ""
	var dirs []FileObject
	
	for {
		lsRes, err := bucketClient.ListObjects(oss.Prefix(prefix), oss.Delimiter("/"), oss.Marker(marker))
		if err != nil {
			return nil, err
		}
		
		// Add folders to the result
		for _, prefix := range lsRes.CommonPrefixes {
			dirs = append(dirs, FileObject{
				Name:        prefix,
				Size:        0,
				ContentType: "application/directory",
				IsDir:       true,
			})
		}
		
		if lsRes.IsTruncated {
			marker = lsRes.NextMarker
		} else {
			break
		}
	}
	
	return dirs, nil
}

// EnsurePathExists ensures that all directories in the given path exist
func (o *OSSStorage) EnsurePathExists(ctx context.Context, bucket, objectPath string) error {
	// Extract directory path from the object path
	dir := path.Dir(objectPath)
	
	// If the directory is the root directory, nothing to do
	if dir == "." || dir == "/" {
		return nil
	}
	
	// Ensure the directory path ends with "/"
	if !strings.HasSuffix(dir, "/") {
		dir += "/"
	}
	
	// Check if directory already exists
	bucketClient, err := o.client.Bucket(bucket)
	if err != nil {
		return err
	}
	
	_, err = bucketClient.GetObjectDetailedMeta(dir)
	if err == nil {
		// Directory already exists
		return nil
	}
	
	// If the error indicates the object doesn't exist, create the directory
	// OSS returns a specific error code when object doesn't exist
	if ossErr, ok := err.(oss.ServiceError); ok {
		if ossErr.Code == "NoSuchKey" {
			return o.CreateDirectory(ctx, bucket, dir)
		}
	}
	
	// For other errors, return the error
	return err
}