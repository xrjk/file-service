package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds the configuration for the file service
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Storage  StorageConfig  `mapstructure:"storage"`
	Log      LogConfig      `mapstructure:"log"`
}

// ServerConfig holds the HTTP server configuration
type ServerConfig struct {
	Port int `mapstructure:"port"`
}

// StorageConfig holds the storage configuration
type StorageConfig struct {
	Type string `mapstructure:"type"` // minio, oss, obs, azure
	
	// Default bucket name
	Bucket string `mapstructure:"bucket"`
	
	// MinIO configuration
	MinIO MinIOConfig `mapstructure:"minio"`
	
	// Aliyun OSS configuration
	OSS OSSConfig `mapstructure:"oss"`
	
	// Huawei Cloud OBS configuration
	OBS OBSConfig `mapstructure:"obs"`
	
	// Azure Blob configuration
	Azure AzureConfig `mapstructure:"azure"`
}

// MinIOConfig holds MinIO configuration
type MinIOConfig struct {
	Endpoint    string `mapstructure:"endpoint"`
	AccessKey   string `mapstructure:"access_key"`
	SecretKey   string `mapstructure:"secret_key"`
	UseSSL      bool   `mapstructure:"use_ssl"`
}

// OSSConfig holds Aliyun OSS configuration
type OSSConfig struct {
	Endpoint    string `mapstructure:"endpoint"`
	AccessKey   string `mapstructure:"access_key"`
	SecretKey   string `mapstructure:"secret_key"`
	UseSSL      bool   `mapstructure:"use_ssl"`
}

// OBSConfig holds Huawei Cloud OBS configuration
type OBSConfig struct {
	Endpoint    string `mapstructure:"endpoint"`
	AccessKey   string `mapstructure:"access_key"`
	SecretKey   string `mapstructure:"secret_key"`
	UseSSL      bool   `mapstructure:"use_ssl"`
}

// AzureConfig holds Azure Blob configuration
type AzureConfig struct {
	Endpoint        string `mapstructure:"endpoint"`
	AccountName     string `mapstructure:"account_name"`
	AccountKey      string `mapstructure:"account_key"`
	ConnectionString string `mapstructure:"connection_string"`
}

// LogConfig holds log configuration
type LogConfig struct {
	Level string `mapstructure:"level"`
}

// LoadConfig loads configuration from file and environment variables
func LoadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	
	// Set default values
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("storage.type", "minio")
	viper.SetDefault("storage.bucket", "default")
	viper.SetDefault("log.level", "info")
	
	// Enable environment variable support
	viper.SetEnvPrefix("FILESERVICE")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	
	// Read configuration
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found, will use defaults and environment variables
	}
	
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	

	
	return &config, nil
}