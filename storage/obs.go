package storage

import (
	"context"
	"io"
	"path"
	"strings"
	"time"

	"github.com/huaweicloud/huaweicloud-sdk-go-obs/obs"
)

// OBStorage implements the Storage interface for Huawei Cloud OBS
type OBStorage struct {
	client *obs.ObsClient
}

// NewOBStorage creates a new OBS storage instance
func NewOBStorage(endpoint, accessKey, secretKey string, useSSL bool) (*OBStorage, error) {
	// 根据useSSL参数决定是否使用HTTPS
	if !useSSL {
		endpoint = "http://" + endpoint
	} else {
		endpoint = "https://" + endpoint
	}
	
	client, err := obs.New(accessKey, secretKey, endpoint)
	if err != nil {
		return nil, err
	}

	return &OBStorage{
		client: client,
	}, nil
}

// Upload uploads a file to OBS
func (o *OBStorage) Upload(ctx context.Context, bucketName, objectName string, reader io.Reader, size int64, contentType string) error {
	input := &obs.PutObjectInput{}
	input.Bucket = bucketName
	input.Key = objectName
	input.Body = reader
	
	if contentType != "" {
		input.ContentType = contentType
	}

	_, err := o.client.PutObject(input)
	return err
}

// Download downloads a file from OBS
func (o *OBStorage) Download(ctx context.Context, bucketName, objectName string) (io.ReadCloser, error) {
	input := &obs.GetObjectInput{}
	input.Bucket = bucketName
	input.Key = objectName
	
	output, err := o.client.GetObject(input)
	if err != nil {
		return nil, err
	}
	
	return output.Body, nil
}

// Delete deletes a file from OBS
func (o *OBStorage) Delete(ctx context.Context, bucketName, objectName string) error {
	input := &obs.DeleteObjectInput{}
	input.Bucket = bucketName
	input.Key = objectName
	
	_, err := o.client.DeleteObject(input)
	return err
}

// List lists objects in an OBS bucket
func (o *OBStorage) List(ctx context.Context, bucketName string, prefix string) ([]FileObject, error) {
	input := &obs.ListObjectsInput{}
	input.Bucket = bucketName
	input.Prefix = prefix
	
	output, err := o.client.ListObjects(input)
	if err != nil {
		return nil, err
	}
	
	var objects []FileObject
	for _, object := range output.Contents {
		contentType := string(object.StorageClass) // OBS doesn't directly provide content type
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		
		objects = append(objects, FileObject{
			Name:         object.Key,
			Size:         object.Size,
			ContentType:  contentType,
			LastModified: object.LastModified.Format(time.RFC3339),
			Metadata:     make(map[string]string), // UserMetadata not available in this context
		})
	}
	
	return objects, nil
}

// GetObjectInfo gets metadata of an object from OBS
func (o *OBStorage) GetObjectInfo(ctx context.Context, bucketName, objectName string) (*FileObject, error) {
	input := &obs.GetObjectMetadataInput{}
	input.Bucket = bucketName
	input.Key = objectName
	
	output, err := o.client.GetObjectMetadata(input)
	if err != nil {
		return nil, err
	}
	
	contentType := output.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	
	return &FileObject{
		Name:         objectName,
		Size:         output.ContentLength,
		ContentType:  contentType,
		LastModified: output.LastModified.Format(time.RFC3339),
		Metadata:     make(map[string]string), // Metadata not directly available in this context
	}, nil
}

// ListDirectories lists directories in a bucket with the given prefix
func (o *OBStorage) ListDirectories(ctx context.Context, bucket, prefix string) ([]FileObject, error) {
	input := &obs.ListObjectsInput{}
	input.Bucket = bucket
	input.Prefix = prefix
	input.Delimiter = "/"
	
	result, err := o.client.ListObjects(input)
	if err != nil {
		return nil, err
	}
	
	var dirs []FileObject
	
	// Process common prefixes (directories)
	for _, prefixInfo := range result.CommonPrefixes {
		dirs = append(dirs, FileObject{
			Name:        prefixInfo,
			Size:        0,
			ContentType: "application/directory",
			IsDir:       true,
		})
	}
	
	return dirs, nil
}

// CreateDirectory creates a directory in the storage
func (o *OBStorage) CreateDirectory(ctx context.Context, bucket, objectName string) error {
	input := &obs.PutObjectInput{}
	input.Bucket = bucket
	
	// Ensure the object name ends with "/"
	if !strings.HasSuffix(objectName, "/") {
		objectName += "/"
	}
	
	input.Key = objectName
	input.Body = strings.NewReader("")
	input.ContentType = "application/directory"
	
	_, err := o.client.PutObject(input)
	return err
}

// EnsurePathExists ensures that all directories in the given path exist
func (o *OBStorage) EnsurePathExists(ctx context.Context, bucket, objectPath string) error {
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
	input := &obs.GetObjectMetadataInput{}
	input.Bucket = bucket
	input.Key = dir
	
	_, err := o.client.GetObjectMetadata(input)
	if err == nil {
		// Directory already exists
		return nil
	}
	
	// If the error indicates the object doesn't exist, create the directory
	if obsError, ok := err.(obs.ObsError); ok {
		if obsError.Code == "NoSuchKey" || obsError.Code == "404" { // Also handle 404 Not Found
			return o.CreateDirectory(ctx, bucket, dir)
		}
	}
	
	// For other errors, return the error
	return err
}