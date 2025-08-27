package api

import (
	// "context"
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"

	"github.com/example/file-service/config"
	"github.com/example/file-service/storage"
)

// Server represents the HTTP server
type Server struct {
	engine  *gin.Engine
	storage storage.Storage
	config  *config.Config
}

// AuthMiddleware is the authentication middleware
func (s *Server) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 如果未启用鉴权，则直接通过
		if !s.config.Auth.Enabled {
			c.Next()
			return
		}

		// 获取API Key
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			// 如果header中没有，尝试从查询参数获取
			apiKey = c.Query("api_key")
		}

		// 检查API Key是否有效
		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "API key is required"})
			c.Abort()
			return
		}

		// 检查API Key是否在配置中
		if _, exists := s.config.Auth.APIKeys[apiKey]; !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key"})
			c.Abort()
			return
		}

		// 鉴权通过
		c.Next()
	}
}

// NewServer creates a new HTTP server
func NewServer(cfg *config.Config) (*Server, error) {
	// Set gin to release mode in production
	if viper.GetString("log.level") != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create gin engine
	engine := gin.New()
	engine.Use(gin.Logger())
	engine.Use(gin.Recovery())

	// Create storage based on config
	store, err := createStorage(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage: %w", err)
	}

	server := &Server{
		engine:  engine,
		storage: store,
		config:  cfg,
	}

	// Register routes
	server.registerRoutes()

	return server, nil
}

// createStorage creates a storage instance based on configuration
func createStorage(cfg *config.Config) (storage.Storage, error) {
	switch cfg.Storage.Type {
	case "minio":
		return storage.NewMinIOStorage(
			cfg.Storage.MinIO.Endpoint,
			cfg.Storage.MinIO.AccessKey,
			cfg.Storage.MinIO.SecretKey,
			cfg.Storage.MinIO.UseSSL,
		)
	case "oss":
		return storage.NewOSSStorage(
			cfg.Storage.OSS.Endpoint,
			cfg.Storage.OSS.AccessKey,
			cfg.Storage.OSS.SecretKey,
			cfg.Storage.OSS.UseSSL,
		)
	case "obs":
		return storage.NewOBStorage(
			cfg.Storage.OBS.Endpoint,
			cfg.Storage.OBS.AccessKey,
			cfg.Storage.OBS.SecretKey,
			cfg.Storage.OBS.UseSSL,
		)
	case "azure":
		// 如果提供了连接字符串，优先使用连接字符串
		if cfg.Storage.Azure.ConnectionString != "" {
			// 这里需要修改Azure存储实现以支持连接字符串
			// 暂时还是使用账户名和密钥的方式
		}
		// 构造完整的endpoint URL
		endpoint := cfg.Storage.Azure.Endpoint
		if endpoint == "" && cfg.Storage.Azure.AccountName != "" {
			endpoint = fmt.Sprintf("https://%s.blob.core.windows.net", cfg.Storage.Azure.AccountName)
		}
		return storage.NewAzureStorage(
			cfg.Storage.Azure.AccountName,
			cfg.Storage.Azure.AccountKey,
			endpoint,
		)
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", cfg.Storage.Type)
	}
}

// registerRoutes registers HTTP routes
func (s *Server) registerRoutes() {
	// Health check endpoint - 不需要鉴权
	s.engine.GET("/health", s.healthCheck)

	// 应用鉴权中间件到所有需要保护的路由
	authorized := s.engine.Group("/")
	authorized.Use(s.AuthMiddleware())

	{
		// File operations
		authorized.POST("/upload/:bucket/*object", s.uploadFile)
		authorized.GET("/download/:bucket/*object", s.downloadFile)
		authorized.DELETE("/delete/:bucket/*object", s.deleteFile)
		authorized.GET("/list/:bucket", s.listObjects)
		authorized.GET("/list/", s.listObjects) // 添加对/list/路径的支持
		authorized.HEAD("/info/:bucket/*object", s.getObjectInfo)
	}
}

// healthCheck handles health check requests
func (s *Server) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"storage": s.config.Storage.Type,
	})
}

// uploadFile handles file upload requests
func (s *Server) uploadFile(c *gin.Context) {
	// Use default bucket if not specified
	bucket := c.Param("bucket")
	object := c.Param("object")
	
	// Remove leading slash from object name (Gin adds it for wildcard parameters)
	if strings.HasPrefix(object, "/") {
		object = object[1:]
	}
	
	// Debug logging
	fmt.Printf("Upload request - Bucket: %s, Object: %s\n", bucket, object)
	
	// Ensure path exists
	if err := s.storage.EnsurePathExists(c.Request.Context(), bucket, object); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to ensure path exists: %v", err)})
		return
	}
	
	// Get content type
	contentType := c.GetHeader("Content-Type")
	// 当Content-Type不为空时使用它，否则使用默认值
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	
	// Get content length
	contentLengthStr := c.GetHeader("Content-Length")
	var contentLength int64
	if contentLengthStr != "" {
		var err error
		contentLength, err = strconv.ParseInt(contentLengthStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Content-Length header"})
			return
		}
	}
	
	// Upload file
	err := s.storage.Upload(c.Request.Context(), bucket, object, c.Request.Body, contentLength, contentType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to upload file: %v", err)})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "File uploaded successfully",
		"bucket":  bucket,
		"object":  object,
	})
}

// downloadFile handles file download requests
// If the 'directory' query parameter is set to 'true', it downloads all files with the given prefix as a ZIP archive
func (s *Server) downloadFile(c *gin.Context) {
	// Use default bucket if not specified
	bucket := c.Param("bucket")
	if bucket == "" {
		bucket = s.config.Storage.Bucket
	}
	object := c.Param("object")
	
	// Remove leading slash from object name (Gin adds it for wildcard parameters)
	if strings.HasPrefix(object, "/") {
		object = object[1:]
	}
	
	// Check if directory download is requested
	isDirectory := c.Query("directory") == "true"
	
	if isDirectory {
		// Ensure object (prefix) ends with "/" to denote a directory
		prefix := object
		if prefix != "" && !strings.HasSuffix(prefix, "/") {
			prefix += "/"
		}
		
		// List objects with the given prefix
		objects, err := s.storage.List(c.Request.Context(), bucket, prefix)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to list objects: %v", err)})
			return
		}
		
		// Set response headers for ZIP file download
		c.Header("Content-Type", "application/zip")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.zip\"", path.Base(strings.TrimSuffix(prefix, "/"))))
		
		// Create a zip writer
		zipWriter := zip.NewWriter(c.Writer)
		defer zipWriter.Close()
		
		// Download and add each object to the ZIP archive
		for _, obj := range objects {
			// Skip directories
			if obj.IsDir || strings.HasSuffix(obj.Name, "/") {
				continue
			}
			
			// Download object
			reader, err := s.storage.Download(c.Request.Context(), bucket, obj.Name)
			if err != nil {
				// Log error and continue with other files
				continue
			}
			
			// Create file header in ZIP
			zipFileWriter, err := zipWriter.Create(obj.Name[len(prefix):]) // Remove prefix from file name in ZIP
			if err != nil {
				reader.Close()
				continue
			}
			
			// Copy file content to ZIP
			_, err = io.Copy(zipFileWriter, reader)
			reader.Close()
			if err != nil {
				continue
			}
		}
		return
	}
	
	// Download single file
	reader, err := s.storage.Download(c.Request.Context(), bucket, object)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to download file: %v", err)})
		return
	}
	defer reader.Close()
	
	// Get file info
	info, err := s.storage.GetObjectInfo(c.Request.Context(), bucket, object)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get file info: %v", err)})
		return
	}
	
	// Set content type header
	c.Header("Content-Type", info.ContentType)
	
	// Stream file to client
	_, err = io.Copy(c.Writer, reader)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to stream file: %v", err)})
		return
	}
}

// deleteObjects handles bulk object deletion requests by prefix
func (s *Server) deleteObjects(c *gin.Context) {
	// Use default bucket if not specified
	bucket := c.Param("bucket")
	if bucket == "" {
		bucket = s.config.Storage.Bucket
	}
	
	// Get prefix from path parameter
	prefix := c.Param("prefix")
	// Remove leading slash from prefix (Gin adds it for wildcard parameters)
	if strings.HasPrefix(prefix, "/") {
		prefix = prefix[1:]
	}
	
	// List objects with the given prefix
	objects, err := s.storage.List(c.Request.Context(), bucket, prefix)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to list objects: %v", err)})
		return
	}
	
	// Delete each object
	var deleted []string
	var errors []string
	
	for _, obj := range objects {
		err := s.storage.Delete(c.Request.Context(), bucket, obj.Name)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Failed to delete %s: %v", obj.Name, err))
		} else {
			deleted = append(deleted, obj.Name)
		}
	}
	
	c.JSON(http.StatusOK, gin.H{
		"bucket":  bucket,
		"prefix":  prefix,
		"deleted": deleted,
		"errors":  errors,
	})
}

// deleteFile handles file deletion requests
func (s *Server) deleteFile(c *gin.Context) {
	// Use default bucket if not specified
	bucket := c.Param("bucket")
	if bucket == "" {
		bucket = s.config.Storage.Bucket
	}
	object := c.Param("object")
	
	// Remove leading slash from object name (Gin adds it for wildcard parameters)
	if strings.HasPrefix(object, "/") {
		object = object[1:]
	}
	
	// Delete file
	err := s.storage.Delete(c.Request.Context(), bucket, object)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to delete file: %v", err)})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "File deleted successfully",
		"bucket":  bucket,
		"object":  object,
	})
}

// listObjects handles object listing requests
func (s *Server) listObjects(c *gin.Context) {
	// Use default bucket if not specified
	bucket := c.Param("bucket")
	if bucket == "" {
		bucket = s.config.Storage.Bucket
	}
	
	// Get prefix from query parameter or path parameter
	prefix := c.Query("prefix")
	if prefix == "" {
		prefix = c.Param("prefix")
		// Remove leading slash from prefix (Gin adds it for wildcard parameters)
		if strings.HasPrefix(prefix, "/") {
			prefix = prefix[1:]
		}
	}
	
	// List objects
	objects, err := s.storage.List(c.Request.Context(), bucket, prefix)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to list objects: %v", err)})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"bucket":  bucket,
		"prefix":  prefix,
		"objects": objects,
	})
}

// getObjectInfo handles object info requests
func (s *Server) getObjectInfo(c *gin.Context) {
	// Use default bucket if not specified
	bucket := c.Param("bucket")
	if bucket == "" {
		bucket = s.config.Storage.Bucket
	}
	object := c.Param("object")
	
	// Remove leading slash from object name (Gin adds it for wildcard parameters)
	if strings.HasPrefix(object, "/") {
		object = object[1:]
	}
	
	// Get object info
	info, err := s.storage.GetObjectInfo(c.Request.Context(), bucket, object)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get object info: %v", err)})
		return
	}
	
	// Set headers
	c.Header("Content-Type", info.ContentType)
	c.Header("Content-Length", strconv.FormatInt(info.Size, 10))
	c.Header("Last-Modified", info.LastModified)
	
	// Return metadata in response headers or body
	for key, value := range info.Metadata {
		c.Header("X-Meta-"+key, value)
	}
	
	c.Status(http.StatusOK)
}

// Start starts the HTTP server
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.config.Server.Port)
	return s.engine.Run(addr)
}
