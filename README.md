# File Service

A file service that supports multiple cloud storage backends including MinIO, Aliyun OSS, Huawei Cloud OBS, and Azure Blob Storage.

## Features

- Support for multiple storage backends:
  - MinIO
  - Aliyun OSS
  - Huawei Cloud OBS
  - Azure Blob Storage
- RESTful API for file operations
- Configuration via YAML file or environment variables
- Docker support
- Authentication support (API Key based)

## Getting Started

### Prerequisites

- Go 1.21 or higher
- A cloud storage account (MinIO, Aliyun OSS, Huawei Cloud OBS, or Azure Blob Storage)

### Installation

1. Clone the repository:
   ```bash
   git clone <repository-url>
   cd file-service
   ```

2. Install dependencies:
   ```bash
   go mod tidy
   ```

3. Configure the service by editing `config.yaml`:
   ```yaml
   server:
     port: 8080

   auth:
     enabled: true  # Set to true to enable authentication
     api_keys:
       "sk-1234567890abcdef": "Default admin key"
       "sk-0987654321fedcba": "Another user key"

   storage:
     # Storage type: minio, oss, obs, azure
     type: "minio"
     # Default bucket name
     bucket: "test"
     
     minio:
       endpoint: "localhost:9000"
       access_key: "your-access-key"
       secret_key: "your-secret-key"
       use_ssl: false
    
     oss:
       endpoint: "oss-cn-hangzhou.aliyuncs.com"
       access_key: "your-access-key"
       secret_key: "your-secret-key"
       use_ssl: true
    
     obs:
       endpoint: "obs.cn-east-3.myhuaweicloud.com"
       access_key: "your-access-key"
       secret_key: "your-secret-key"
       use_ssl: true
    
     azure:
       endpoint: "https://your-account.blob.core.windows.net"
       account_name: "your-account-name"
       account_key: "your-account-key"
       connection_string: "your-connection-string"  # Optional, if provided will be used for authentication

   log:
     level: "info"
   ```

4. Run the service:
   ```bash
   go run cmd/main/main.go
   ```

## Authentication

The file service supports API Key based authentication. When authentication is enabled, all file operations require a valid API Key.

### Enabling Authentication

To enable authentication, set `auth.enabled` to `true` in the configuration file and define at least one API key in the `auth.api_keys` map.

### Using Authentication

When authentication is enabled, you must provide a valid API Key with each request. You can provide the API Key in one of two ways:

1. Via the `X-API-Key` header:
   ```bash
   curl -H "X-API-Key: sk-1234567890abcdef" \
        -X GET http://localhost:8080/list/mybucket
   ```

2. Via the `api_key` query parameter:
   ```bash
   curl -X GET "http://localhost:8080/list/mybucket?api_key=sk-1234567890abcdef"
   ```

### Disabling Authentication

To disable authentication, set `auth.enabled` to `false` in the configuration file. When authentication is disabled, all requests will be processed without requiring an API Key.

## API Endpoints

### Health Check

- `GET /health` - Health check

### File Operations

- `POST /upload/:bucket/*object` - Upload a file (bucket is optional, will use default if not specified)
- `GET /download/:bucket/*object` - Download a file (bucket is optional, will use default if not specified)
- `GET /download/:bucket/*object?directory=true` - Download all files with the specified prefix as a ZIP archive
- `DELETE /delete/:bucket/*object` - Delete a file (bucket is optional, will use default if not specified)
- `DELETE /delete/:bucket/*prefix` - Delete all files with the specified prefix
- `GET /list/:bucket` - List objects in a bucket (bucket is optional, will use default if not specified)
- `GET /list/:bucket/*prefix` - List objects with the specified prefix in a bucket
- `HEAD /info/:bucket/*object` - Get object information (bucket is optional, will use default if not specified)

### Upload a file

```bash
# With specific bucket and object path
curl -X POST -H "Content-Type: application/octet-stream" --data-binary @file.txt http://localhost:8080/upload/my-bucket/path/to/file.txt

# With default bucket
curl -X POST -H "Content-Type: application/octet-stream" --data-binary @file.txt http://localhost:8080/upload//path/to/file.txt
```

### Download a file

```bash
# With specific bucket
curl -X GET http://localhost:8080/download/my-bucket/file.txt -o downloaded-file.txt

# With default bucket
curl -X GET http://localhost:8080/download//file.txt -o downloaded-file.txt

# Download all files with a specific prefix as a ZIP archive
curl -X GET "http://localhost:8080/download/my-bucket/path/to/files?directory=true" -o files.zip
```

### Delete a file

```bash
# With specific bucket
curl -X DELETE http://localhost:8080/delete/my-bucket/file.txt

# With default bucket
curl -X DELETE http://localhost:8080/delete//file.txt

# Delete all files with a specific prefix
curl -X DELETE http://localhost:8080/delete/my-bucket/path/to/files
```

### List objects

```bash
# With specific bucket
curl -X GET http://localhost:8080/list/my-bucket

# With default bucket
curl -X GET http://localhost:8080/list/

# List objects with a specific prefix
curl -X GET http://localhost:8080/list/my-bucket/path/to/files
```

### Get object info

```bash
# With specific bucket
curl -X HEAD http://localhost:8080/info/my-bucket/file.txt

# With default bucket
curl -X HEAD http://localhost:8080/info//file.txt
```

## Supported Storage Types

### MinIO

Set `storage.type` to `minio` and configure the MinIO section with your MinIO server details.

### Aliyun OSS

Set `storage.type` to `oss` and configure the OSS section with your Aliyun OSS credentials.

### Huawei Cloud OBS

Set `storage.type` to `obs` and configure the OBS section with your Huawei Cloud OBS credentials.

### Azure Blob Storage

Set `storage.type` to `azure` and configure the Azure section with your Azure Blob Storage credentials.

## Building

To build the service:

```bash
go build -o file-service cmd/main/main.go
```

## Docker

To build and run with Docker:

```bash
docker build -t file-service .
docker run -p 8080:8080 file-service
```
