# Spawnr

A modern Kubernetes job manager with multi-cluster support that allows you to create, monitor, and manage jobs from existing deployments across multiple EKS clusters.

## Features

### Core Functionality
- 🚀 **Deployment-based Job Creation**: Create Kubernetes jobs from existing deployment specifications
- 🌐 **Multi-Cluster Management**: Connect and manage multiple EKS clusters from a single interface
- 🔄 **Cross-namespace Support**: Work with deployments across different namespaces
- 🖥️ **Modern Web Interface**: Responsive web UI with dark mode support
- 📊 **Real-time Logs**: View job logs with proper formatting
- 🔍 **Job Persistence**: Track jobs created by Spawnr using Kubernetes labels
- 🗑️ **Complete Job Management**: Create, monitor, refresh status, and delete jobs with automatic pod cleanup

### Multi-Cluster Features
- 🔌 **Cluster Connectivity Testing**: Verify cluster connections before use
- 🔐 **AWS IAM Integration**: Uses IAM Roles for Service Accounts (IRSA) for secure EKS access
- 📋 **Cluster Cards**: Visual representation of configured clusters with status indicators
- 🔄 **Easy Cluster Switching**: Switch between local and remote EKS clusters seamlessly
- 🔒 **Certificate Management**: Automatic fetching and storage of cluster CA certificates

### User Experience
- 🌓 **Dark Mode**: Toggle between light and dark themes with persistent preferences
- 📑 **Tabbed Interface**: Separate tabs for Jobs and Cluster management
- ✅ **Input Validation**: Automatic sanitization of job names to meet Kubernetes requirements
- 🔔 **Smart Notifications**: Non-intrusive alerts that don't disrupt the layout

## Quick Start

### Install with Helm (Recommended)

The fastest way to get started:

```bash
# Add the Helm repository
helm repo add spawnr https://jordanvanderlinden.github.io/spawnr/
helm repo update

# Install Spawnr
helm install spawnr spawnr/spawnr --namespace spawnr --create-namespace

# Access the UI
kubectl port-forward -n spawnr svc/spawnr 8080:80
```

Then open http://localhost:8080 in your browser.

For production deployments with IRSA, see [Kubernetes Deployment](#kubernetes-deployment) below.

### Prerequisites for Development
- Go 1.25+
- Docker (for building images)
- kubectl configured with access to a Kubernetes cluster
- Helm 3+ (for Kubernetes deployment)
- AWS CLI (for EKS cluster management)

### Local Development

1. **Clone the repository**:
   ```bash
   git clone https://github.com/jordanvanderlinden/spawnr.git
   cd spawnr
   ```

2. **Install dependencies**:
   ```bash
   go mod download
   ```

3. **Run the application**:
   ```bash
   go run main.go
   ```

4. **Access the web interface**:
   Open http://localhost:8080 in your browser

### Kubernetes Deployment

1. **Add the Helm repository**:
   ```bash
   helm repo add spawnr https://jordanvanderlinden.github.io/spawnr/
   helm repo update
   ```

2. **Install with default values**:
   ```bash
   # Create namespace
   kubectl create namespace spawnr
   
   # Install the Helm chart
   helm install spawnr spawnr/spawnr --namespace spawnr
   ```

3. **Install with custom values**:
   ```bash
   # Create a values.yaml file with your configuration
   helm install spawnr spawnr/spawnr --namespace spawnr -f custom-values.yaml
   ```

4. **Configure IAM Role for Service Account (IRSA)**:
   - Create an IAM role with EKS access permissions
   - Set the role ARN in your values file or use `--set`:
     ```bash
     helm install spawnr spawnr/spawnr \
       --namespace spawnr \
       --set serviceAccount.annotations."eks\.amazonaws\.io/role-arn"="arn:aws:iam::ACCOUNT_ID:role/ROLE_NAME"
     ```
   
   Or in `values.yaml`:
   ```yaml
   serviceAccount:
     annotations:
       eks.amazonaws.com/role-arn: arn:aws:iam::ACCOUNT_ID:role/ROLE_NAME
   ```

5. **Access the application**:
   ```bash
   # Port forward to access locally
   kubectl port-forward -n spawnr svc/spawnr 8080:80
   ```
   
   Then open http://localhost:8080 in your browser

### Helm Configuration Options

For a complete list of configuration options, see `helm/spawnr/values.yaml` or run:
```bash
helm show values spawnr/spawnr
```

### Building Docker Images

```bash
# Build for production (multi-arch)
docker build --platform linux/amd64 -t spawnr:latest .

# Push to your registry
docker tag spawnr:latest YOUR_REGISTRY/spawnr:latest
docker push YOUR_REGISTRY/spawnr:latest
```

## Usage

### Jobs Tab (Default View)

1. **Select Cluster**: Choose the cluster you want to work with (Local or remote EKS clusters)
2. **Select Namespace**: Pick the namespace containing your deployments
3. **Select Deployment**: Choose a deployment to base your job on
4. **Configure Job**: 
   - Enter a job name (will be auto-sanitized if needed)
   - Specify the command to run in the container
5. **Create Job**: Click "Create Job" to launch your job
6. **Monitor Jobs**: 
   - View all jobs created by Spawnr across namespaces
   - Click "View Logs" to see job output
   - Use "Refresh" to update job statuses
   - Delete jobs when no longer needed (automatically cleans up pods)

### Clusters Tab

1. **View Clusters**: See all configured clusters with their status
2. **Test Connectivity**: Click "Test Connection" to verify cluster access
3. **Add New Cluster**:
   - Click "Add New Cluster"
   - Fill in the required information:
     - **Cluster Name**: The actual EKS cluster name
     - **Friendly Name**: Display name for the cluster
     - **Role ARN**: AWS IAM role ARN for cluster access
     - **Endpoint URL**: EKS cluster endpoint URL
     - **Certificate Authority** (Optional): Base64-encoded CA cert (auto-fetched if not provided)
4. **Delete Cluster**: Remove clusters you no longer need (Local cluster cannot be deleted)

### Theme Toggle

Click the moon/sun icon in the navigation bar to toggle between light and dark modes. Your preference is automatically saved.

## Architecture

### Components

- **Web Interface**: Modern HTML/CSS/JavaScript frontend with Bootstrap 5.3
- **API Server**: Gin-based HTTP server with RESTful endpoints
- **Kubernetes Client**: Go client-go library for cluster interaction
- **Multi-Cluster Manager**: Handles switching between different Kubernetes contexts
- **Job Controller**: Manages job lifecycle, monitoring, and cleanup

### Security

- **RBAC**: Proper role-based access control for Kubernetes resources
- **IRSA**: IAM Roles for Service Accounts for secure AWS access
- **Security Context**: Non-root user with appropriate permissions
- **TLS**: Secure communication with Kubernetes API servers
- **Image Security**: Multi-stage builds with minimal attack surface

### Job Tracking

Jobs created by Spawnr are labeled with:
```yaml
labels:
  app.kubernetes.io/managed-by: spawnr
```

This allows Spawnr to track and display jobs across all namespaces, providing persistence between browser sessions.

## API Endpoints

### Cluster Management
- `GET /api/clusters` - List all configured clusters
- `POST /api/clusters` - Add a new cluster
- `POST /api/clusters/switch` - Switch to a different cluster
- `GET /api/clusters/:name` - Get cluster information
- `DELETE /api/clusters/:name` - Remove a cluster

### Namespace & Deployment Management
- `GET /api/namespaces` - List namespaces in the current cluster
- `GET /api/deployments` - List deployments in the current namespace
- `GET /api/deployments/:namespace/:name` - Get deployment details

### Job Management
- `GET /api/jobs` - List all jobs managed by Spawnr (across all namespaces)
- `POST /api/jobs` - Create a new job
- `GET /api/jobs/:namespace/:name` - Get job details
- `DELETE /api/jobs/:namespace/:name` - Delete a job (and its pods)
- `GET /api/jobs/:namespace/:name/logs` - Get job logs
- `GET /api/jobs/:namespace/:name/watch` - Watch job events (SSE)

## Configuration

### Environment Variables

- `PORT`: Server port (default: 8080)
- `GIN_MODE`: Gin mode (debug/release)
- `POD_NAMESPACE`: Current namespace (injected by Kubernetes)
- `AWS_SDK_LOAD_CONFIG`: Enable AWS SDK config loading (set to "true")
- `AWS_EC2_METADATA_DISABLED`: Control EC2 metadata access (set to "false")
- `HOME`: Home directory for AWS CLI cache (set to "/tmp" in container)

### Helm Values

Key configuration options in `helm/spawnr/values.yaml`:

```yaml
image:
  repository: spawnr
  pullPolicy: Always
  tag: "latest"

serviceAccount:
  create: true
  annotations:
    eks.amazonaws.com/role-arn: ""  # Set your IAM role ARN here

rbac:
  create: true
  rules:
    - apiGroups: [""]
      resources: ["namespaces", "pods", "secrets"]
      verbs: ["get", "list", "watch", "create", "delete"]
    - apiGroups: ["apps"]
      resources: ["deployments"]
      verbs: ["get", "list"]
    - apiGroups: ["batch"]
      resources: ["jobs"]
      verbs: ["get", "list", "create", "delete", "watch"]

resources:
  limits:
    cpu: 500m
    memory: 512Mi
  requests:
    cpu: 100m
    memory: 128Mi
```

### Cluster Secret Format

Remote EKS clusters are stored as Kubernetes secrets with the label `spawnr.io/cluster: "true"`:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: eks-cluster-name
  namespace: spawnr
  labels:
    spawnr.io/cluster: "true"
type: Opaque
data:
  cluster-name: <base64-encoded-cluster-name>
  friendly-name: <base64-encoded-display-name>
  endpoint: <base64-encoded-endpoint-url>
  role-arn: <base64-encoded-iam-role-arn>
  certificate-authority-data: <base64-encoded-ca-cert>
```

## Development

### Project Structure

```
spawnr/
├── main.go                      # Application entry point
├── go.mod                       # Go module definition
├── internal/
│   ├── handlers/
│   │   └── handlers.go          # HTTP request handlers
│   ├── k8s/
│   │   └── client.go            # Kubernetes client and multi-cluster logic
│   └── server/
│       └── server.go            # HTTP server setup and routing
├── web/
│   ├── static/
│   │   └── app.js               # Frontend JavaScript application
│   └── templates/
│       └── index.html           # Main HTML template
├── helm/spawnr/                 # Helm chart
│   ├── Chart.yaml
│   ├── values.yaml
│   └── templates/
│       ├── deployment.yaml
│       ├── service.yaml
│       ├── rbac.yaml
│       ├── serviceaccount.yaml
│       └── hpa.yaml
├── Dockerfile                   # Production multi-stage build
└── README.md
```

### Building

```bash
# Build the binary
go build -o spawnr main.go

# Build Docker image for production
docker build --platform linux/amd64 -t spawnr:latest .

# Build and push to ECR (example)
docker build --platform linux/amd64 -t 123456789.dkr.ecr.us-east-1.amazonaws.com/spawnr:latest .
docker push 123456789.dkr.ecr.us-east-1.amazonaws.com/spawnr:latest
```

### Testing Locally

```bash
# Run unit tests
go test ./...

# Run with verbose output
go test -v ./...

# Test with coverage
go test -cover ./...
```

## RBAC Permissions

Spawnr requires the following Kubernetes permissions:

- **Namespaces**: `get`, `list`, `watch` - To discover available namespaces
- **Deployments**: `get`, `list` - To read deployment specifications
- **Jobs**: `get`, `list`, `create`, `delete`, `watch` - To manage job lifecycle
- **Pods**: `get`, `list`, `delete` - To view logs and cleanup orphaned pods
- **Secrets**: `get`, `list`, `watch`, `create`, `delete` - To store cluster configurations

These are defined in the Helm chart's `rbac.yaml` template.

## Troubleshooting

### Cluster Connection Issues

1. **Verify IRSA Configuration**: Ensure your service account has the correct IAM role annotation
2. **Check AWS Permissions**: The IAM role must have permissions to call EKS APIs
3. **Test Connectivity**: Use the "Test Connection" button in the Clusters tab
4. **Check Logs**: View pod logs with `kubectl logs -n spawnr deployment/spawnr`

### Job Creation Failures

1. **Check RBAC**: Ensure the service account has permissions to create jobs
2. **Verify Deployment**: Make sure the source deployment exists in the selected namespace
3. **Review Logs**: Check the job logs for container-specific errors

### Certificate Issues

If you see TLS certificate errors:
1. Provide the CA certificate when adding the cluster
2. Or let Spawnr auto-fetch it (requires AWS CLI access)
3. Check that the certificate-authority-data is stored in the cluster secret

## CI/CD Pipeline

This project uses GitHub Actions for automated releases and deployments:

### Automated Release Process

When commits are pushed to `master`, the following happens automatically:

1. **Semantic Release**: Analyzes commit messages to determine version bump
   - `feat:` commits trigger minor version bump (e.g., 1.0.0 → 1.1.0)
   - `fix:` commits trigger patch version bump (e.g., 1.0.0 → 1.0.1)
   - `BREAKING CHANGE:` or `feat!:` triggers major version bump (e.g., 1.0.0 → 2.0.0)
   
2. **Docker Image Build & Push**: Multi-architecture images built and pushed to GitHub Container Registry
   - Tagged with version (e.g., `ghcr.io/jordanvanderlinden/spawnr:1.2.3`)
   - Tagged with `latest`
   
3. **Helm Chart Release**: Chart packaged and published to GitHub Pages
   - Chart version updated to match release version
   - Available via Helm repository

### Commit Message Convention

This project follows [Conventional Commits](https://www.conventionalcommits.org/):

```bash
# Feature (minor version bump)
feat(clusters): add connectivity testing

# Bug fix (patch version bump)
fix(jobs): resolve pod cleanup issue

# Breaking change (major version bump)
feat!: redesign API endpoints

# Documentation, no release
docs: update installation guide
```

For more details, see [CONTRIBUTING.md](.github/CONTRIBUTING.md).

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feat/amazing-feature`)
3. Make your changes using [conventional commits](https://www.conventionalcommits.org/)
4. Add tests if applicable
5. Commit your changes (`git commit -m 'feat: add some amazing feature'`)
6. Push to the branch (`git push origin feat/amazing-feature`)
7. Open a Pull Request

See [CONTRIBUTING.md](.github/CONTRIBUTING.md) for detailed guidelines.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Built with [Gin Web Framework](https://gin-gonic.com/)
- Kubernetes client-go library
- Bootstrap 5.3 for the UI
- Font Awesome for icons