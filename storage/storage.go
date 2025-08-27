package storage

import (
	"context"
	"io"
)

// FileObject represents a file object in the storage system
type FileObject struct {
	Name         string
	Size         int64
	ContentType  string
	LastModified string
	Metadata     map[string]string
	IsDir        bool // 标识是否为目录
}

// Storage interface defines the methods that all storage providers must implement
type Storage interface {
	// Upload uploads a file to the storage
	Upload(ctx context.Context, bucket, objectName string, reader io.Reader, size int64, contentType string) error
	
	// Download downloads a file from the storage
	Download(ctx context.Context, bucket, objectName string) (io.ReadCloser, error)
	
	// Delete deletes a file from the storage
	Delete(ctx context.Context, bucket, objectName string) error
	
	// List lists objects in a bucket
	List(ctx context.Context, bucket string, prefix string) ([]FileObject, error)
	
	// GetObjectInfo gets metadata of an object
	GetObjectInfo(ctx context.Context, bucket, objectName string) (*FileObject, error)
	
	// CreateDirectory creates a directory in the storage
	CreateDirectory(ctx context.Context, bucket, objectName string) error
	
	// ListDirectories lists directories in a bucket with the given prefix
	ListDirectories(ctx context.Context, bucket, prefix string) ([]FileObject, error)
	
	// EnsurePathExists ensures that all directories in the given path exist
	EnsurePathExists(ctx context.Context, bucket, objectPath string) error
}