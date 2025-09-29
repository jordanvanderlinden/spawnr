# Spawnr

A Kubernetes job manager that allows you to create jobs from existing deployments with custom commands.

## Features

- 🚀 **Deployment-based Job Creation**: Create Kubernetes jobs from existing deployment specifications
- 🌐 **Cross-namespace Support**: Work with deployments across different namespaces
- 🖥️ **Web Interface**: Modern, responsive web UI for managing jobs
- 📊 **Real-time Logs**: View job logs in real-time
- 🗑️ **Job Management**: Create, monitor, and delete jobs easily
- 🔒 **Security**: Follows Kubernetes security best practices

## Quick Start

### Local Development

1. **Prerequisites**:
   - Go 1.25+
   - Docker and Docker Compose
   - kubectl configured with access to a Kubernetes cluster

2. **Run locally**:
   ```bash
   # Clone the repository
   git clone <repository-url>
   cd spawnr
   
   # Install dependencies
   go mod download
   
   # Run the application
   go run main.go
   ```

3. **Access the web interface**:
   Open http://localhost:8080 in your browser

### Docker Development

```bash
# Build and run with Docker Compose
docker-compose -f docker-compose.dev.yml up --build

# Or run the production image
docker-compose up --build
```

### Kubernetes Deployment

1. **Using Helm**:
   ```bash
   # Install the Helm chart
   helm install spawnr ./helm/spawnr
   
   # Or with custom values
   helm install spawnr ./helm/spawnr -f custom-values.yaml
   ```

2. **Access the application**:
   ```bash
   # Port forward to access locally
   kubectl port-forward svc/spawnr 8080:80
   ```

## Usage

1. **Select Namespace**: Choose the namespace containing your deployments
2. **Select Deployment**: Pick a deployment to base your job on
3. **Configure Job**: 
   - Enter a unique job name
   - Specify the command to run in the container
4. **Create Job**: Click "Create Job" to launch your job
5. **Monitor**: View logs and job status in real-time
6. **Cleanup**: Delete jobs when no longer needed

## Architecture

### Components

- **Web Interface**: React-based frontend for job management
- **API Server**: Gin-based HTTP server with RESTful endpoints
- **Kubernetes Client**: Go client for cluster interaction
- **Job Controller**: Manages job lifecycle and monitoring

### Security

- **RBAC**: Proper role-based access control for Kubernetes resources
- **Security Context**: Non-root user with read-only filesystem
- **Network Policies**: Isolated network access
- **Image Security**: Multi-stage builds with minimal attack surface

## API Endpoints

- `GET /api/namespaces` - List available namespaces
- `GET /api/deployments?namespace=<ns>` - List deployments in namespace
- `GET /api/deployments/<namespace>/<name>` - Get deployment details
- `POST /api/jobs` - Create a new job
- `GET /api/jobs/<namespace>/<name>` - Get job details
- `DELETE /api/jobs/<namespace>/<name>` - Delete a job
- `GET /api/jobs/<namespace>/<name>/logs` - Get job logs
- `GET /api/jobs/<namespace>/<name>/watch` - Watch job events (SSE)

## Configuration

### Environment Variables

- `PORT`: Server port (default: 8080)
- `GIN_MODE`: Gin mode (debug/release)

### Helm Values

See `helm/spawnr/values.yaml` for all configurable options.

## Development

### Project Structure

```
spawnr/
├── main.go                 # Application entry point
├── internal/
│   ├── handlers/          # HTTP handlers
│   ├── k8s/              # Kubernetes client
│   └── server/           # HTTP server setup
├── web/
│   ├── static/           # Static assets (CSS, JS)
│   └── templates/        # HTML templates
├── helm/spawnr/          # Helm chart
├── Dockerfile            # Production image
├── Dockerfile.dev        # Development image
└── docker-compose.yml    # Local development
```

### Building

```bash
# Build the binary
go build -o spawnr main.go

# Build Docker image
docker build -t spawnr .

# Build with Helm
helm package ./helm/spawnr
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.
