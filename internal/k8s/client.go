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

	// Try in-cluster config first (when running in Kubernetes)
	config, err = rest.InClusterConfig()
	if err != nil {
		// For local development, use kubeconfig with specific cluster
		if clusterName != "" {
			config, err = getKubeconfigForCluster(clusterName)
			if err != nil {
				return nil, fmt.Errorf("failed to get kubeconfig for cluster %s: %w", clusterName, err)
			}
		} else {
			// Fall back to default kubeconfig
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
	}, nil
}

func (c *Client) ListDeployments(namespace string) (*appsv1.DeploymentList, error) {
	return c.clientset.AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{})
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
	defer stream.Close()

	logs, err := io.ReadAll(stream)
	if err != nil {
		return "", err
	}

	return string(logs), nil
}

func (c *Client) ListNamespaces() (*corev1.NamespaceList, error) {
	return c.clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
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

// getEKSConfig creates a Kubernetes config for an EKS cluster using AWS CLI
func getEKSConfig(clusterName, profile string) (*rest.Config, error) {
	// Get AWS region from environment or use AWS default
	region := os.Getenv("AWS_REGION")
	if region == "" {
		if profile != "" {
			region = getRegionForProfile(profile)
		} else {
			// Try to get the default region from AWS CLI
			cmd := exec.Command("aws", "configure", "get", "region")
			output, err := cmd.Output()
			if err == nil && len(output) > 0 {
				region = strings.TrimSpace(string(output))
			} else {
				region = "us-east-1" // Fallback region
			}
		}
	}

	// Build the aws eks update-kubeconfig command
	args := []string{"eks", "update-kubeconfig", "--region", region, "--name", clusterName, "--kubeconfig", "/tmp/kubeconfig-" + clusterName}
	if profile != "" {
		args = append(args, "--profile", profile)
	}

	cmd := exec.Command("aws", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to update kubeconfig for cluster %s: %s, %w", clusterName, string(output), err)
	}

	// Load the generated kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", "/tmp/kubeconfig-"+clusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig for cluster %s: %w", clusterName, err)
	}

	return config, nil
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

	// Get the current namespace from environment or use default
	namespace := os.Getenv("POD_NAMESPACE")
	if namespace == "" {
		namespace = "default"
	}

	// List secrets with label "spawnr.io/cluster=true"
	secrets, err := clientset.CoreV1().Secrets(namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: "spawnr.io/cluster=true",
	})
	if err != nil {
		// If we can't list secrets, just return the local cluster
		return clusters, nil
	}

	// Add clusters from secrets
	for _, secret := range secrets.Items {
		// Extract cluster information from secret data
		clusterName := string(secret.Data["cluster-name"])
		endpoint := string(secret.Data["endpoint"])
		friendlyName := string(secret.Data["friendly-name"])

		// Extract region from endpoint URL
		region := "unknown"
		if strings.Contains(endpoint, ".eks.") {
			parts := strings.Split(endpoint, ".eks.")
			if len(parts) > 1 {
				regionParts := strings.Split(parts[1], ".")
				if len(regionParts) > 0 {
					region = regionParts[0]
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
	// Handle the default local cluster
	if clusterName == "local" {
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

	// Get the current namespace from environment or use default
	namespace := os.Getenv("POD_NAMESPACE")
	if namespace == "" {
		namespace = "default"
	}

	// Get the cluster secret
	secret, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), clusterName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster secret %s: %w", clusterName, err)
	}

	// Extract cluster information from secret
	roleArn := string(secret.Data["role-arn"])
	endpoint := string(secret.Data["endpoint"])

	// Use AWS CLI to get the kubeconfig for this cluster
	cmd := exec.Command("aws", "eks", "get-token", "--cluster-name", clusterName, "--role-arn", roleArn)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get EKS token: %w", err)
	}

	// Parse the token response
	var tokenResponse struct {
		Status struct {
			Token string `json:"token"`
		} `json:"status"`
	}
	if err := json.Unmarshal(output, &tokenResponse); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	// Create a temporary kubeconfig with the token
	tempKubeconfig := &clientcmdapi.Config{
		Clusters: map[string]*clientcmdapi.Cluster{
			clusterName: {
				Server: endpoint,
			},
		},
		AuthInfos: map[string]*clientcmdapi.AuthInfo{
			clusterName: {
				Token: tokenResponse.Status.Token,
			},
		},
		Contexts: map[string]*clientcmdapi.Context{
			clusterName: {
				Cluster:  clusterName,
				AuthInfo: clusterName,
			},
		},
		CurrentContext: clusterName,
	}

	// Create config from the temporary kubeconfig
	clientConfig := clientcmd.NewDefaultClientConfig(*tempKubeconfig, &clientcmd.ConfigOverrides{})
	return clientConfig.ClientConfig()
}

// CreateClusterSecret creates a Kubernetes secret for a cluster
func CreateClusterSecret(clusterName, friendlyName, roleArn, endpoint string) error {
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

	// Get the current namespace from environment or use default
	namespace := os.Getenv("POD_NAMESPACE")
	if namespace == "" {
		namespace = "default"
	}

	// Create the secret
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterName,
			Labels: map[string]string{
				"spawnr.io/cluster": "true",
			},
		},
		Data: map[string][]byte{
			"cluster-name":  []byte(clusterName),
			"friendly-name": []byte(friendlyName),
			"role-arn":      []byte(roleArn),
			"endpoint":      []byte(endpoint),
		},
	}

	// Create the secret
	_, err = clientset.CoreV1().Secrets(namespace).Create(context.TODO(), secret, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create cluster secret: %w", err)
	}

	return nil
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

	// Get the current namespace from environment or use default
	namespace := os.Getenv("POD_NAMESPACE")
	if namespace == "" {
		namespace = "default"
	}

	// Delete the secret
	err = clientset.CoreV1().Secrets(namespace).Delete(context.TODO(), clusterName, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete cluster secret: %w", err)
	}

	return nil
}

// getAWSProfiles returns a list of available AWS profiles
func getAWSProfiles() ([]string, error) {
	// Get profiles from AWS config
	cmd := exec.Command("aws", "configure", "list-profiles")
	output, err := cmd.Output()
	if err != nil {
		// Fallback: try to read from ~/.aws/config
		return []string{"default"}, nil
	}

	profiles := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(profiles) == 0 {
		return []string{"default"}, nil
	}

	// Filter out problematic profiles that are known to have SSO issues
	var filteredProfiles []string
	skipProfiles := map[string]bool{
		"datalake":      true,
		"datalake_prod": true,
		"datalake_dev":  true,
		"dr":            true,
	}

	for _, profile := range profiles {
		if !skipProfiles[profile] {
			filteredProfiles = append(filteredProfiles, profile)
		}
	}

	return filteredProfiles, nil
}

// listClustersForProfile lists EKS clusters for a specific AWS profile
func listClustersForProfile(profile string) ([]ClusterInfo, error) {
	// Get region for this profile
	region := getRegionForProfile(profile)

	// List EKS clusters for this profile
	cmd := exec.Command("aws", "eks", "list-clusters", "--region", region, "--profile", profile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to list EKS clusters for profile %s: %s, %w", profile, string(output), err)
	}

	// Parse the JSON output to extract cluster names
	var result struct {
		Clusters []string `json:"clusters"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse cluster list JSON: %w", err)
	}

	var clusters []ClusterInfo
	for _, clusterName := range result.Clusters {
		clusters = append(clusters, ClusterInfo{
			Name:    clusterName,
			Region:  region,
			Status:  "ACTIVE", // Assume active, could be enhanced to check actual status
			Profile: profile,  // Add profile information
		})
	}

	return clusters, nil
}

// getRegionForProfile gets the region for a specific AWS profile
func getRegionForProfile(profile string) string {
	cmd := exec.Command("aws", "configure", "get", "region", "--profile", profile)
	output, err := cmd.Output()
	if err != nil || len(output) == 0 {
		return "us-east-1" // Default region
	}
	return strings.TrimSpace(string(output))
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
