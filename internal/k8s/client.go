package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type Client struct {
	clientset *kubernetes.Clientset
	config    *rest.Config
}

type ClusterInfo struct {
	Name         string `json:"name"`
	Region       string `json:"region"`
	Endpoint     string `json:"endpoint"`
	Status       string `json:"status"`
	Profile      string `json:"profile"`
	OriginalName string `json:"originalName"`
}

func NewClient() (*Client, error) {
	return NewClientWithCluster("")
}

func NewClientWithCluster(clusterName string) (*Client, error) {
	return NewClientWithClusterAndProfile(clusterName, "")
}

func NewClientWithClusterAndProfile(clusterName, profile string) (*Client, error) {
	var config *rest.Config
	var err error

	// If a specific cluster is requested, use getKubeconfigForCluster
	if clusterName != "" {
		fmt.Printf("[NewClientWithCluster] Creating client for cluster: %s\n", clusterName)
		config, err = getKubeconfigForCluster(clusterName)
		if err != nil {
			return nil, fmt.Errorf("failed to get kubeconfig for cluster %s: %w", clusterName, err)
		}
	} else {
		// No specific cluster requested, try in-cluster config first
		config, err = rest.InClusterConfig()
		if err != nil {
			// Fall back to default kubeconfig for local development
			kubeconfig := os.Getenv("KUBECONFIG")
			if kubeconfig == "" {
				kubeconfig = ""
			}
			config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
			if err != nil {
				return nil, fmt.Errorf("failed to create Kubernetes config: %w", err)
			}
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes clientset: %w", err)
	}

	return &Client{
		clientset: clientset,
		config:    config,
	}, nil
}

func (c *Client) ListDeployments(namespace string) (*appsv1.DeploymentList, error) {
	// Log the server URL to identify which cluster is being queried
	serverURL := c.config.Host
	fmt.Printf("[ListDeployments] Querying cluster at: %s for namespace: %s\n", serverURL, namespace)

	deployments, err := c.clientset.AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Printf("[ListDeployments] ERROR querying %s (ns=%s): %v\n", serverURL, namespace, err)
		return nil, err
	}

	fmt.Printf("[ListDeployments] Found %d deployments in namespace %s from %s\n", len(deployments.Items), namespace, serverURL)
	return deployments, nil
}

func (c *Client) GetDeployment(namespace, name string) (*appsv1.Deployment, error) {
	return c.clientset.AppsV1().Deployments(namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

func (c *Client) CreateJob(namespace string, job *batchv1.Job) (*batchv1.Job, error) {
	return c.clientset.BatchV1().Jobs(namespace).Create(context.TODO(), job, metav1.CreateOptions{})
}

func (c *Client) GetJob(namespace, name string) (*batchv1.Job, error) {
	return c.clientset.BatchV1().Jobs(namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

func (c *Client) DeleteJob(namespace, name string) error {
	// First, find and delete all pods associated with this job
	labelSelector := fmt.Sprintf("job-name=%s", name)
	pods, err := c.clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		// Log the error but continue with job deletion
		fmt.Printf("Warning: failed to list pods for job %s: %v\n", name, err)
	} else {
		// Delete each pod associated with the job
		deletePolicy := metav1.DeletePropagationForeground
		for _, pod := range pods.Items {
			err := c.clientset.CoreV1().Pods(namespace).Delete(context.TODO(), pod.Name, metav1.DeleteOptions{
				PropagationPolicy: &deletePolicy,
			})
			if err != nil {
				fmt.Printf("Warning: failed to delete pod %s: %v\n", pod.Name, err)
			}
		}
	}

	// Delete the job itself with propagation policy to clean up any remaining resources
	propagationPolicy := metav1.DeletePropagationForeground
	return c.clientset.BatchV1().Jobs(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{
		PropagationPolicy: &propagationPolicy,
	})
}

func (c *Client) GetJobLogs(namespace, jobName string) (string, error) {
	// Get the job to find associated pods
	job, err := c.GetJob(namespace, jobName)
	if err != nil {
		return "", err
	}

	// Find pods associated with this job
	labelSelector := fmt.Sprintf("job-name=%s", job.Name)
	pods, err := c.clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return "", err
	}

	if len(pods.Items) == 0 {
		return "No pods found for this job", nil
	}

	// Get logs from the first pod
	pod := pods.Items[0]
	req := c.clientset.CoreV1().Pods(namespace).GetLogs(pod.Name, &corev1.PodLogOptions{})

	stream, err := req.Stream(context.TODO())
	if err != nil {
		return "", err
	}
	defer func() {
		if closeErr := stream.Close(); closeErr != nil {
			fmt.Printf("Warning: failed to close stream: %v\n", closeErr)
		}
	}()

	logs, err := io.ReadAll(stream)
	if err != nil {
		return "", err
	}

	return string(logs), nil
}

func (c *Client) ListNamespaces() (*corev1.NamespaceList, error) {
	// Log the server URL to identify which cluster is being queried
	serverURL := c.config.Host
	fmt.Printf("[ListNamespaces] Querying cluster at: %s\n", serverURL)

	namespaces, err := c.clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Printf("[ListNamespaces] ERROR querying %s: %v\n", serverURL, err)
		return nil, err
	}

	fmt.Printf("[ListNamespaces] Found %d namespaces from %s\n", len(namespaces.Items), serverURL)
	return namespaces, nil
}

func (c *Client) WatchJobEvents(namespace, jobName string) (<-chan string, error) {
	events := make(chan string, 100)

	go func() {
		defer close(events)

		// Watch for job events
		watcher, err := c.clientset.BatchV1().Jobs(namespace).Watch(context.TODO(), metav1.ListOptions{
			FieldSelector: fields.OneTermEqualSelector("metadata.name", jobName).String(),
		})
		if err != nil {
			events <- fmt.Sprintf("Error watching job: %v", err)
			return
		}
		defer watcher.Stop()

		for event := range watcher.ResultChan() {
			job := event.Object.(*batchv1.Job)
			events <- fmt.Sprintf("Job %s: %s", job.Name, event.Type)

			// Check if job is complete
			if job.Status.Succeeded > 0 {
				events <- "Job completed successfully"
				return
			}
			if job.Status.Failed > 0 {
				events <- "Job failed"
				return
			}
		}
	}()

	return events, nil
}

// ListAllSpawnrJobs lists all jobs across all namespaces managed by spawnr
func (c *Client) ListAllSpawnrJobs() ([]batchv1.Job, error) {
	// Get all namespaces first
	namespaces, err := c.ListNamespaces()
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}

	var allJobs []batchv1.Job

	// Iterate through each namespace and find jobs with the spawnr label
	for _, ns := range namespaces.Items {
		jobs, err := c.clientset.BatchV1().Jobs(ns.Name).List(context.TODO(), metav1.ListOptions{
			LabelSelector: "app.kubernetes.io/managed-by=spawnr",
		})
		if err != nil {
			// Log but continue with other namespaces
			fmt.Printf("Warning: failed to list jobs in namespace %s: %v\n", ns.Name, err)
			continue
		}

		allJobs = append(allJobs, jobs.Items...)
	}

	return allJobs, nil
}

// ListEKSClusters returns a list of available clusters from Kubernetes secrets
func ListEKSClusters() ([]ClusterInfo, error) {
	var clusters []ClusterInfo

	// Always add the default local cluster first
	clusters = append(clusters, ClusterInfo{
		Name:         "Local Cluster",
		Region:       "local",
		Status:       "ACTIVE",
		Profile:      "in-cluster",
		OriginalName: "local", // Use "local" as the identifier for the default cluster
	})

	// Try to get additional clusters from secrets (only if we're in a cluster)
	config, err := rest.InClusterConfig()
	if err != nil {
		// Not running in-cluster, so no additional clusters from secrets
		return clusters, nil
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		// If we can't create a clientset, just return the local cluster
		return clusters, nil
	}

	// Get the current namespace from environment or use spawnr as default
	namespace := os.Getenv("POD_NAMESPACE")
	if namespace == "" {
		namespace = "spawnr"
	}

	// List secrets with label "spawnr.io/cluster=true"
	secrets, err := clientset.CoreV1().Secrets(namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: "spawnr.io/cluster=true",
	})
	if err != nil {
		// If we can't list secrets, just return the local cluster
		return clusters, nil
	}

	// Add clusters from secrets and update any missing CA certificates
	for _, secret := range secrets.Items {
		// Extract cluster information from secret data
		clusterName := string(secret.Data["cluster-name"])
		endpoint := string(secret.Data["endpoint"])
		friendlyName := string(secret.Data["friendly-name"])
		roleArn := string(secret.Data["role-arn"])
		certificateAuthority := string(secret.Data["certificate-authority-data"])

		// If this secret doesn't have a CA certificate, try to fetch and update it
		if certificateAuthority == "" && roleArn != "" && clusterName != "" {
			fmt.Printf("[ListEKSClusters] Secret '%s' missing CA cert, attempting to fetch...\n", secret.Name)
			caCert, err := fetchClusterCertificate(clusterName, roleArn)
			if err != nil {
				fmt.Printf("[ListEKSClusters] WARNING: Failed to fetch CA cert for %s: %v\n", secret.Name, err)
			} else {
				// Update the secret with the CA certificate
				secret.Data["certificate-authority-data"] = []byte(caCert)
				_, err = clientset.CoreV1().Secrets(namespace).Update(context.TODO(), &secret, metav1.UpdateOptions{})
				if err != nil {
					fmt.Printf("[ListEKSClusters] WARNING: Failed to update secret %s with CA cert: %v\n", secret.Name, err)
				} else {
					fmt.Printf("[ListEKSClusters] Successfully updated secret '%s' with CA certificate\n", secret.Name)
				}
			}
		}

		// Extract region from endpoint URL
		// EKS endpoint format: https://<hash>.<region>.eks.amazonaws.com
		region := "unknown"
		if strings.Contains(endpoint, ".eks.") {
			parts := strings.Split(endpoint, ".eks.")
			if len(parts) > 0 {
				// parts[0] should be like "https://<hash>.<region>"
				// Split by dots and get the last element before .eks.
				beforeEks := strings.Split(parts[0], ".")
				if len(beforeEks) >= 2 {
					// The region is the second-to-last element before .eks.
					region = beforeEks[len(beforeEks)-1]
				}
			}
		}

		clusters = append(clusters, ClusterInfo{
			Name:         friendlyName,
			Region:       region,
			Status:       "ACTIVE",
			Profile:      "role-arn",  // Indicate this uses role ARN
			OriginalName: clusterName, // Keep the original cluster name for switching
		})
	}

	return clusters, nil
}

// getKubeconfigForCluster creates a Kubernetes config for a specific cluster using AWS CLI
func getKubeconfigForCluster(clusterName string) (*rest.Config, error) {
	fmt.Printf("[getKubeconfigForCluster] Requested cluster: %s\n", clusterName)

	// Handle the default local cluster
	if clusterName == "local" {
		fmt.Printf("[getKubeconfigForCluster] Using local cluster config\n")
		// Use in-cluster config for the local cluster
		config, err := rest.InClusterConfig()
		if err != nil {
			// For local development, use kubeconfig
			kubeconfig := os.Getenv("KUBECONFIG")
			if kubeconfig == "" {
				kubeconfig = filepath.Join(os.Getenv("HOME"), ".kube", "config")
			}
			config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
			if err != nil {
				return nil, fmt.Errorf("failed to create Kubernetes config: %w", err)
			}
		}
		fmt.Printf("[getKubeconfigForCluster] Local cluster config host: %s\n", config.Host)
		return config, nil
	}

	// Create a Kubernetes client for secret access
	config, err := rest.InClusterConfig()
	if err != nil {
		// For local development, use kubeconfig
		kubeconfig := os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			kubeconfig = filepath.Join(os.Getenv("HOME"), ".kube", "config")
		}
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create Kubernetes config: %w", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes clientset: %w", err)
	}

	// Get the current namespace from environment or use spawnr as default
	namespace := os.Getenv("POD_NAMESPACE")
	if namespace == "" {
		namespace = "spawnr"
	}

	fmt.Printf("[getKubeconfigForCluster] Looking for secret '%s' in namespace '%s'\n", clusterName, namespace)

	// Get the cluster secret
	secret, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), clusterName, metav1.GetOptions{})
	if err != nil {
		fmt.Printf("[getKubeconfigForCluster] ERROR: Failed to get secret: %v\n", err)
		return nil, fmt.Errorf("failed to get cluster secret %s: %w", clusterName, err)
	}

	// Extract cluster information from secret
	actualClusterName := string(secret.Data["cluster-name"])
	roleArn := string(secret.Data["role-arn"])
	endpoint := string(secret.Data["endpoint"])
	certificateAuthority := string(secret.Data["certificate-authority-data"])

	fmt.Printf("[getKubeconfigForCluster] Found secret - actualClusterName: %s, endpoint: %s, roleArn: %s, hasCA: %v\n",
		actualClusterName, endpoint, roleArn, certificateAuthority != "")

	// Use AWS CLI to get the kubeconfig for this cluster
	fmt.Printf("[getKubeconfigForCluster] Running: aws eks get-token --cluster-name %s --role-arn %s\n", actualClusterName, roleArn)
	cmd := exec.Command("aws", "eks", "get-token", "--cluster-name", actualClusterName, "--role-arn", roleArn)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			fmt.Printf("[getKubeconfigForCluster] ERROR: AWS CLI failed: %s\n", string(exitErr.Stderr))
			return nil, fmt.Errorf("failed to get EKS token for cluster %s: %s, %w", actualClusterName, string(exitErr.Stderr), err)
		}
		fmt.Printf("[getKubeconfigForCluster] ERROR: AWS CLI command failed: %v\n", err)
		return nil, fmt.Errorf("failed to get EKS token for cluster %s: %w", actualClusterName, err)
	}

	fmt.Printf("[getKubeconfigForCluster] AWS CLI token retrieved successfully\n")

	// Parse the token response
	var tokenResponse struct {
		Status struct {
			Token string `json:"token"`
		} `json:"status"`
	}
	if err := json.Unmarshal(output, &tokenResponse); err != nil {
		fmt.Printf("[getKubeconfigForCluster] ERROR: Failed to parse token JSON: %v\n", err)
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	// Create cluster config with or without CA certificate
	clusterConfig := &clientcmdapi.Cluster{
		Server: endpoint,
	}

	// Use CA certificate if available, otherwise skip TLS verification
	if certificateAuthority != "" {
		clusterConfig.CertificateAuthorityData = []byte(certificateAuthority)
		fmt.Printf("[getKubeconfigForCluster] Using CA certificate for TLS verification\n")
	} else {
		clusterConfig.InsecureSkipTLSVerify = true
		fmt.Printf("[getKubeconfigForCluster] WARNING: No CA certificate, using insecure TLS\n")
	}

	// Create a temporary kubeconfig with the token
	tempKubeconfig := &clientcmdapi.Config{
		Clusters: map[string]*clientcmdapi.Cluster{
			actualClusterName: clusterConfig,
		},
		AuthInfos: map[string]*clientcmdapi.AuthInfo{
			actualClusterName: {
				Token: tokenResponse.Status.Token,
			},
		},
		Contexts: map[string]*clientcmdapi.Context{
			actualClusterName: {
				Cluster:  actualClusterName,
				AuthInfo: actualClusterName,
			},
		},
		CurrentContext: actualClusterName,
	}

	// Create config from the temporary kubeconfig
	clientConfig := clientcmd.NewDefaultClientConfig(*tempKubeconfig, &clientcmd.ConfigOverrides{})
	finalConfig, err := clientConfig.ClientConfig()
	if err != nil {
		fmt.Printf("[getKubeconfigForCluster] ERROR: Failed to create client config: %v\n", err)
		return nil, err
	}

	fmt.Printf("[getKubeconfigForCluster] Successfully created config for endpoint: %s\n", finalConfig.Host)
	return finalConfig, nil
}

// CreateClusterSecret creates a Kubernetes secret for a cluster
func CreateClusterSecret(clusterName, friendlyName, roleArn, endpoint, certificateAuthority string) error {
	// If CA cert is not provided, fetch it from AWS EKS
	if certificateAuthority == "" {
		fmt.Printf("[CreateClusterSecret] No CA cert provided, fetching from AWS EKS for cluster: %s\n", clusterName)
		caCert, err := fetchClusterCertificate(clusterName, roleArn)
		if err != nil {
			fmt.Printf("[CreateClusterSecret] WARNING: Failed to fetch CA cert: %v, will use insecure\n", err)
			// Continue without CA cert - will use insecure TLS
		} else {
			certificateAuthority = caCert
			fmt.Printf("[CreateClusterSecret] Successfully fetched CA certificate for cluster: %s\n", clusterName)
		}
	}

	// Create a Kubernetes client
	config, err := rest.InClusterConfig()
	if err != nil {
		// For local development, use kubeconfig
		kubeconfig := os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			kubeconfig = filepath.Join(os.Getenv("HOME"), ".kube", "config")
		}
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return fmt.Errorf("failed to create Kubernetes config: %w", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes clientset: %w", err)
	}

	// Get the current namespace from environment or use spawnr as default
	namespace := os.Getenv("POD_NAMESPACE")
	if namespace == "" {
		namespace = "spawnr"
	}

	// Create the secret data
	secretData := map[string][]byte{
		"cluster-name":  []byte(clusterName),
		"friendly-name": []byte(friendlyName),
		"role-arn":      []byte(roleArn),
		"endpoint":      []byte(endpoint),
	}

	// Add CA cert if available
	if certificateAuthority != "" {
		secretData["certificate-authority-data"] = []byte(certificateAuthority)
	}

	// Create the secret
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterName,
			Labels: map[string]string{
				"spawnr.io/cluster": "true",
			},
		},
		Data: secretData,
	}

	// Create the secret
	_, err = clientset.CoreV1().Secrets(namespace).Create(context.TODO(), secret, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create cluster secret: %w", err)
	}

	return nil
}

// fetchClusterCertificate fetches the CA certificate for an EKS cluster using AWS CLI
func fetchClusterCertificate(clusterName, roleArn string) (string, error) {
	// Use AWS CLI to get cluster details
	cmd := exec.Command("aws", "eks", "describe-cluster", "--name", clusterName, "--role-arn", roleArn, "--query", "cluster.certificateAuthority.data", "--output", "text")
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("aws eks describe-cluster failed: %s", string(exitErr.Stderr))
		}
		return "", fmt.Errorf("failed to execute aws eks describe-cluster: %w", err)
	}

	caCert := strings.TrimSpace(string(output))
	if caCert == "" || caCert == "None" {
		return "", fmt.Errorf("no CA certificate returned from AWS")
	}

	return caCert, nil
}

// DeleteClusterSecret deletes a Kubernetes secret for a cluster
func DeleteClusterSecret(clusterName string) error {
	// Create a Kubernetes client
	config, err := rest.InClusterConfig()
	if err != nil {
		// For local development, use kubeconfig
		kubeconfig := os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			kubeconfig = filepath.Join(os.Getenv("HOME"), ".kube", "config")
		}
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return fmt.Errorf("failed to create Kubernetes config: %w", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes clientset: %w", err)
	}

	// Get the current namespace from environment or use spawnr as default
	namespace := os.Getenv("POD_NAMESPACE")
	if namespace == "" {
		namespace = "spawnr"
	}

	// Delete the secret
	err = clientset.CoreV1().Secrets(namespace).Delete(context.TODO(), clusterName, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete cluster secret: %w", err)
	}

	return nil
}

// GetClusterInfo returns detailed information about a specific EKS cluster
func GetClusterInfo(clusterName string) (*ClusterInfo, error) {
	region := os.Getenv("AWS_REGION")
	if region == "" {
		// Try to get the default region from AWS CLI
		cmd := exec.Command("aws", "configure", "get", "region")
		output, err := cmd.Output()
		if err == nil && len(output) > 0 {
			region = strings.TrimSpace(string(output))
		} else {
			region = "us-east-1" // Fallback region
		}
	}

	cmd := exec.Command("aws", "eks", "describe-cluster",
		"--region", region,
		"--name", clusterName)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to describe cluster %s: %s, %w", clusterName, string(output), err)
	}

	// Parse cluster info from JSON output
	// This is simplified - in production you'd use proper JSON parsing
	info := &ClusterInfo{
		Name:   clusterName,
		Region: region,
		Status: "ACTIVE",
	}

	// Extract endpoint from output
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "\"endpoint\"") {
			parts := strings.Split(line, "\"")
			if len(parts) >= 4 {
				info.Endpoint = parts[3]
			}
		}
	}

	return info, nil
}

// GetServerURL returns the Kubernetes API server URL for this client
func (c *Client) GetServerURL() string {
	if c.config == nil {
		return "<no config>"
	}
	return c.config.Host
}
