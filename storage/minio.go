package storage

import (
	"context"
	"io"

	"path"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinIOStorage implements the Storage interface for MinIO
type MinIOStorage struct {
	client *minio.Client
}

// NewMinIOStorage creates a new MinIO storage instance
func NewMinIOStorage(endpoint, accessKeyID, secretAccessKey string, useSSL bool) (*MinIOStorage, error) {
	// Initialize minio client object.
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, err
	}

	return &MinIOStorage{
		client: client,
	}, nil
}

// Upload uploads a file to MinIO
func (m *MinIOStorage) Upload(ctx context.Context, bucket, objectName string, reader io.Reader, size int64, contentType string) error {
	opts := minio.PutObjectOptions{
		ContentType: contentType,
	}
	_, err := m.client.PutObject(ctx, bucket, objectName, reader, size, opts)
	return err
}

// Download downloads a file from MinIO
func (m *MinIOStorage) Download(ctx context.Context, bucket, objectName string) (io.ReadCloser, error) {
	opts := minio.GetObjectOptions{}
	return m.client.GetObject(ctx, bucket, objectName, opts)
}

// Delete deletes a file from MinIO
func (m *MinIOStorage) Delete(ctx context.Context, bucket, objectName string) error {
	opts := minio.RemoveObjectOptions{}
	return m.client.RemoveObject(ctx, bucket, objectName, opts)
}

// List lists objects in a MinIO bucket
func (m *MinIOStorage) List(ctx context.Context, bucket string, prefix string) ([]FileObject, error) {
	opts := minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	}
	
	objectCh := m.client.ListObjects(ctx, bucket, opts)
	var objects []FileObject
	
	for object := range objectCh {
		if object.Err != nil {
			return nil, object.Err
		}
		
		objects = append(objects, FileObject{
			Name:         object.Key,
			Size:         object.Size,
			ContentType:  object.ContentType,
			LastModified: object.LastModified.Format(time.RFC3339),
			Metadata:     convertMetadata(object.UserMetadata),
		})
	}
	
	return objects, nil
}

// GetObjectInfo gets metadata of an object from MinIO
func (m *MinIOStorage) GetObjectInfo(ctx context.Context, bucket, objectName string) (*FileObject, error) {
	info, err := m.client.StatObject(ctx, bucket, objectName, minio.StatObjectOptions{})
	if err != nil {
		return nil, err
	}
	
	return &FileObject{
		Name:         info.Key,
		Size:         info.Size,
		ContentType:  info.ContentType,
		LastModified: info.LastModified.Format(time.RFC3339),
		Metadata:     convertMetadata(info.UserMetadata),
	}, nil
}

// ListDirectories lists directories in a bucket with the given prefix
func (m *MinIOStorage) ListDirectories(ctx context.Context, bucket, prefix string) ([]FileObject, error) {
	// In MinIO, directories are simulated by objects with a trailing slash
	opts := minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: false,
	}
	
	var dirs []FileObject
	
	// List all objects with the given prefix
	objectCh := m.client.ListObjects(ctx, bucket, opts)
	for object := range objectCh {
		if object.Err != nil {
			return nil, object.Err
		}
		
		// Check if the object represents a directory (ends with "/")
		if strings.HasSuffix(object.Key, "/") {
			dirs = append(dirs, FileObject{
				Name:         object.Key,
				Size:         object.Size,
				ContentType:  object.ContentType,
				LastModified: object.LastModified.Format(time.RFC3339),
				Metadata:     convertMetadata(object.UserMetadata),
				IsDir:        true,
			})
		}
	}
	
	return dirs, nil
}

// CreateDirectory creates a directory in the storage
func (m *MinIOStorage) CreateDirectory(ctx context.Context, bucket, objectName string) error {
	// Ensure the object name ends with "/"
	if !strings.HasSuffix(objectName, "/") {
		objectName += "/"
	}
	
	// Create an empty object to represent the directory
	opts := minio.PutObjectOptions{ContentType: "application/directory"}
	_, err := m.client.PutObject(ctx, bucket, objectName, strings.NewReader(""), 0, opts)
	return err
}

// EnsurePathExists ensures that all directories in the given path exist
func (m *MinIOStorage) EnsurePathExists(ctx context.Context, bucket, objectPath string) error {
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
	_, err := m.client.StatObject(ctx, bucket, dir, minio.StatObjectOptions{})
	if err == nil {
		// Directory already exists
		return nil
	}
	
	// If the error is not "object not found", return the error
	if minio.ToErrorResponse(err).Code != "NoSuchKey" {
		return err
	}
	
	// Directory doesn't exist, create it
	return m.CreateDirectory(ctx, bucket, dir)
}

// convertMetadata converts minio metadata to map[string]string
func convertMetadata(metadata map[string]string) map[string]string {
	result := make(map[string]string)
	for k, v := range metadata {
		result[k] = v
	}
	return result
}