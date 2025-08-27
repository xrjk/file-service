package storage

import (
	"context"
	"errors"
	"io"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
)

// AzureStorage implements the Storage interface for Azure Blob Storage
type AzureStorage struct {
	client *azblob.Client
}

// NewAzureStorage creates a new Azure Blob storage instance
func NewAzureStorage(accountName, accountKey, serviceURL string) (*AzureStorage, error) {
	// Create a credential object using the account name and key
	credential, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		return nil, err
	}

	// Create a client
	client, err := azblob.NewClientWithSharedKeyCredential(serviceURL, credential, nil)
	if err != nil {
		return nil, err
	}

	return &AzureStorage{
		client: client,
	}, nil
}

// Upload uploads a file to Azure Blob Storage
func (a *AzureStorage) Upload(ctx context.Context, containerName, blobName string, reader io.Reader, size int64, contentType string) error {
	// Upload blob
	options := &azblob.UploadStreamOptions{}
	if contentType != "" {
		options.HTTPHeaders = &blob.HTTPHeaders{
			BlobContentType: &contentType,
		}
	}
	
	_, err := a.client.UploadStream(ctx, containerName, blobName, reader, options)
	return err
}

// Download downloads a file from Azure Blob Storage
func (a *AzureStorage) Download(ctx context.Context, containerName, blobName string) (io.ReadCloser, error) {
	// Download blob
	resp, err := a.client.DownloadStream(ctx, containerName, blobName, nil)
	if err != nil {
		return nil, err
	}
	
	// Return the read closer
	return resp.Body, nil
}

// Delete deletes a file from Azure Blob Storage
func (a *AzureStorage) Delete(ctx context.Context, containerName, blobName string) error {
	// Delete blob
	_, err := a.client.DeleteBlob(ctx, containerName, blobName, nil)
	return err
}

// List lists objects in an Azure Blob Storage container
func (a *AzureStorage) List(ctx context.Context, containerName string, prefix string) ([]FileObject, error) {
	// Create a pager to list blobs
	pager := a.client.NewListBlobsFlatPager(containerName, &azblob.ListBlobsFlatOptions{
		Prefix: &prefix,
	})

	var objects []FileObject
	
	// Iterate through the pages
	for pager.More() {
		// Get the next page
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		
		// Process blobs
		for _, blob := range resp.Segment.BlobItems {
			// Extract content type
			contentType := "application/octet-stream"
			if blob.Properties.ContentType != nil {
				contentType = *blob.Properties.ContentType
			}
			
			// Extract last modified time
			lastModified := time.Now()
			if blob.Properties.LastModified != nil {
				lastModified = *blob.Properties.LastModified
			}
			
			// Extract blob size
			var size int64 = 0
			if blob.Properties.ContentLength != nil {
				size = *blob.Properties.ContentLength
			}
			
			objects = append(objects, FileObject{
				Name:         *blob.Name,
				Size:         size,
				ContentType:  contentType,
				LastModified: lastModified.Format(time.RFC3339),
				Metadata:     make(map[string]string), // Metadata not directly available in this context
			})
		}
	}
	
	return objects, nil
}

// GetObjectInfo gets metadata of a blob from Azure Blob Storage
func (a *AzureStorage) GetObjectInfo(ctx context.Context, containerName, blobName string) (*FileObject, error) {
	// Get blob properties
	blobClient := a.client.ServiceClient().NewContainerClient(containerName).NewBlobClient(blobName)
	resp, err := blobClient.GetProperties(ctx, nil)
	if err != nil {
		return nil, err
	}
	
	// Extract content type
	contentType := "application/octet-stream"
	if resp.ContentType != nil {
		contentType = *resp.ContentType
	}
	
	// Extract last modified time
	lastModified := time.Now()
	if resp.LastModified != nil {
		lastModified = *resp.LastModified
	}
	
	// Extract blob size
	var size int64 = 0
	if resp.ContentLength != nil {
		size = *resp.ContentLength
	}
	
	return &FileObject{
		Name:         blobName,
		Size:         size,
		ContentType:  contentType,
		LastModified: lastModified.Format(time.RFC3339),
		Metadata:     make(map[string]string), // Metadata not directly available in this context
	}, nil
}

// ListDirectories lists directories in a bucket with the given prefix
func (a *AzureStorage) ListDirectories(ctx context.Context, bucket, prefix string) ([]FileObject, error) {
	// In Azure Blob Storage, directories are simulated using prefixes
	// We'll list blobs and extract directory-like prefixes
	
	pager := a.client.NewListBlobsFlatPager(bucket, &azblob.ListBlobsFlatOptions{
		Prefix: &prefix,
	})
	
	dirMap := make(map[string]bool) // To avoid duplicates
	var dirs []FileObject
	
	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		
		for _, blob := range resp.Segment.BlobItems {
			// Extract directory path from blob name
			if blob.Name != nil {
				parts := strings.Split(*blob.Name, "/")
				if len(parts) > 1 {
					// Add all parent directories
					for i := 1; i < len(parts); i++ {
						dirPath := strings.Join(parts[:i], "/") + "/"
						if !dirMap[dirPath] {
							dirMap[dirPath] = true
							dirs = append(dirs, FileObject{
								Name:        dirPath,
								Size:        0,
								ContentType: "application/directory",
								IsDir:       true,
							})
						}
					}
				}
			}
		}
	}
	
	return dirs, nil
}

// CreateDirectory creates a directory in the storage
func (a *AzureStorage) CreateDirectory(ctx context.Context, bucket, objectName string) error {
	// Ensure the object name ends with "/"
	if !strings.HasSuffix(objectName, "/") {
		objectName += "/"
	}
	
	// Create an empty blob to represent the directory
	contentType := "application/directory"
	_, err := a.client.UploadBuffer(ctx, bucket, objectName, []byte{}, &azblob.UploadBufferOptions{
		HTTPHeaders: &blob.HTTPHeaders{
			BlobContentType: &contentType,
		},
	})
	return err
}

// EnsurePathExists ensures that all directories in the given path exist
func (a *AzureStorage) EnsurePathExists(ctx context.Context, bucket, objectPath string) error {
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
	_, err := a.client.DownloadStream(ctx, bucket, dir, nil)
	if err == nil {
		// Directory already exists
		return nil
	}
	
	// If the error indicates the blob doesn't exist, create the directory
	var respErr *azcore.ResponseError
	if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
		return a.CreateDirectory(ctx, bucket, dir)
	}
	
	// For other errors, return the error
	return err
}