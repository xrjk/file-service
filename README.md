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

3. Configure the service by editing `config.yaml` or setting environment variables.

4. Run the service:
   ```bash
   go run cmd/main/main.go
   ```

## Configuration

The service can be configured via a YAML configuration file or environment variables.

### Configuration File

Create a `config.yaml` file in the root directory:

```yaml
server:
  port: 8080

storage:
  # Storage type: minio, oss, obs, azure
  type: "oss"
  # Default bucket name
  bucket: "my-bucket"
  
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

### Environment Variables

All configuration options can also be set via environment variables with the prefix `FILESERVICE_`:

```bash
export FILESERVICE_SERVER_PORT=8080
export FILESERVICE_STORAGE_TYPE=minio
export FILESERVICE_STORAGE_BUCKET=my-bucket
export FILESERVICE_STORAGE_MINIO_ENDPOINT=localhost:9000
export FILESERVICE_STORAGE_MINIO_ACCESS_KEY=minioadmin
export FILESERVICE_STORAGE_MINIO_SECRET_KEY=minioadmin
export FILESERVICE_STORAGE_MINIO_USE_SSL=false
```

## API Endpoints

- `GET /health` - Health check
- `POST /upload/:bucket/:object` - Upload a file (bucket is optional, will use default if not specified)
- `GET /download/:bucket/:object` - Download a file (bucket is optional, will use default if not specified)
- `DELETE /delete/:bucket/:object` - Delete a file (bucket is optional, will use default if not specified)
- `GET /list/:bucket` - List objects in a bucket (bucket is optional, will use default if not specified)
- `HEAD /info/:bucket/:object` - Get object information (bucket is optional, will use default if not specified)

### Upload a file

```bash
# With specific bucket
curl -X POST -H "Content-Type: application/octet-stream" --data-binary @file.txt http://localhost:8080/upload/my-bucket/file.txt

# With default bucket
curl -X POST -H "Content-Type: application/octet-stream" --data-binary @file.txt http://localhost:8080/upload//file.txt
```

### Download a file

```bash
# With specific bucket
curl -X GET http://localhost:8080/download/my-bucket/file.txt -o downloaded-file.txt

# With default bucket
curl -X GET http://localhost:8080/download//file.txt -o downloaded-file.txt
```

### Delete a file

```bash
# With specific bucket
curl -X DELETE http://localhost:8080/delete/my-bucket/file.txt

# With default bucket
curl -X DELETE http://localhost:8080/delete//file.txt
```

### List objects

```bash
# With specific bucket
curl -X GET http://localhost:8080/list/my-bucket

# With default bucket
curl -X GET http://localhost:8080/list/
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