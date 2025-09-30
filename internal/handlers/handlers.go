package handlers

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"spawnr/internal/k8s"

	"github.com/gin-gonic/gin"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Handlers struct {
	k8sClient *k8s.Client
	clientMu  sync.RWMutex
}

func New(k8sClient *k8s.Client) *Handlers {
	return &Handlers{
		k8sClient: k8sClient,
	}
}

type CreateJobRequest struct {
	Namespace  string `json:"namespace" binding:"required"`
	Deployment string `json:"deployment" binding:"required"`
	Command    string `json:"command" binding:"required"`
	JobName    string `json:"jobName" binding:"required"`
}

func (h *Handlers) GetNamespaces(c *gin.Context) {
	h.clientMu.RLock()
	client := h.k8sClient
	h.clientMu.RUnlock()

	fmt.Printf("[GetNamespaces] Handler using client with server: %s\n", client.GetServerURL())

	namespaces, err := client.ListNamespaces()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, namespaces.Items)
}

func (h *Handlers) GetDeployments(c *gin.Context) {
	namespace := c.Query("namespace")
	if namespace == "" {
		namespace = "default"
	}

	h.clientMu.RLock()
	client := h.k8sClient
	h.clientMu.RUnlock()

	fmt.Printf("[GetDeployments] Handler using client with server: %s for namespace: %s\n", client.GetServerURL(), namespace)

	deployments, err := client.ListDeployments(namespace)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, deployments.Items)
}

func (h *Handlers) GetDeployment(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	h.clientMu.RLock()
	client := h.k8sClient
	h.clientMu.RUnlock()

	deployment, err := client.GetDeployment(namespace, name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, deployment)
}

// sanitizeJobName converts a job name to a valid Kubernetes resource name
func sanitizeJobName(name string) string {
	// Convert to lowercase
	name = strings.ToLower(name)

	// Replace spaces and underscores with hyphens
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "_", "-")

	// Remove any characters that aren't alphanumeric or hyphens
	reg := regexp.MustCompile(`[^a-z0-9-]`)
	name = reg.ReplaceAllString(name, "")

	// Remove leading/trailing hyphens
	name = strings.Trim(name, "-")

	// Ensure it starts with an alphanumeric character
	reg = regexp.MustCompile(`^[^a-z0-9]+`)
	name = reg.ReplaceAllString(name, "")

	// Kubernetes names must be max 63 characters
	if len(name) > 63 {
		name = name[:63]
	}

	// Remove trailing hyphens again after truncation
	name = strings.TrimRight(name, "-")

	// If name is empty after sanitization, use a default
	if name == "" {
		name = "job"
	}

	return name
}

func (h *Handlers) CreateJob(c *gin.Context) {
	var req CreateJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Sanitize the job name
	sanitizedName := sanitizeJobName(req.JobName)

	h.clientMu.RLock()
	client := h.k8sClient
	h.clientMu.RUnlock()

	// Get the deployment
	deployment, err := client.GetDeployment(req.Namespace, req.Deployment)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Create job from deployment spec
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      sanitizedName,
			Namespace: req.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "spawnr",
			},
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: deployment.Spec.Template.ObjectMeta,
				Spec:       deployment.Spec.Template.Spec,
			},
		},
	}

	// Ensure pod template has the spawnr label
	if job.Spec.Template.Labels == nil {
		job.Spec.Template.Labels = make(map[string]string)
	}
	job.Spec.Template.Labels["app.kubernetes.io/managed-by"] = "spawnr"

	// Override the command in the first container
	if len(job.Spec.Template.Spec.Containers) > 0 {
		job.Spec.Template.Spec.Containers[0].Command = []string{"/bin/sh", "-c"}
		job.Spec.Template.Spec.Containers[0].Args = []string{req.Command}
	}

	// Set job to not restart
	job.Spec.Template.Spec.RestartPolicy = corev1.RestartPolicyNever

	createdJob, err := client.CreateJob(req.Namespace, job)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, createdJob)
}

func (h *Handlers) GetJob(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	h.clientMu.RLock()
	client := h.k8sClient
	h.clientMu.RUnlock()

	job, err := client.GetJob(namespace, name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, job)
}

func (h *Handlers) DeleteJob(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	h.clientMu.RLock()
	client := h.k8sClient
	h.clientMu.RUnlock()

	err := client.DeleteJob(namespace, name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Job deleted successfully"})
}

func (h *Handlers) GetJobLogs(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	h.clientMu.RLock()
	client := h.k8sClient
	h.clientMu.RUnlock()

	logs, err := client.GetJobLogs(namespace, name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"logs": logs})
}

func (h *Handlers) WatchJob(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	// Set up Server-Sent Events
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	h.clientMu.RLock()
	client := h.k8sClient
	h.clientMu.RUnlock()

	events, err := client.WatchJobEvents(namespace, name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	for event := range events {
		c.SSEvent("message", gin.H{"data": event})
		c.Writer.Flush()
	}
}

func (h *Handlers) GetClusters(c *gin.Context) {
	clusters, err := k8s.ListEKSClusters()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, clusters)
}

func (h *Handlers) GetClusterInfo(c *gin.Context) {
	clusterName := c.Param("name")

	info, err := k8s.GetClusterInfo(clusterName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, info)
}

func (h *Handlers) SwitchCluster(c *gin.Context) {
	var request struct {
		ClusterName string `json:"clusterName"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	fmt.Printf("[SwitchCluster] Switching to cluster: %s\n", request.ClusterName)

	// Create a new client for the selected cluster using the user name
	newClient, err := k8s.NewClientWithCluster(request.ClusterName)
	if err != nil {
		fmt.Printf("[SwitchCluster] ERROR creating client for %s: %v\n", request.ClusterName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	fmt.Printf("[SwitchCluster] Successfully created client for %s, server: %s\n", request.ClusterName, newClient.GetServerURL())

	// Update the handler's client with write lock
	h.clientMu.Lock()
	oldServerURL := h.k8sClient.GetServerURL()
	h.k8sClient = newClient
	h.clientMu.Unlock()

	fmt.Printf("[SwitchCluster] Client updated. Old server: %s, New server: %s\n", oldServerURL, newClient.GetServerURL())

	c.JSON(http.StatusOK, gin.H{"message": "Switched to cluster " + request.ClusterName})
}

// AddCluster adds a new cluster
func (h *Handlers) AddCluster(c *gin.Context) {
	var request struct {
		ClusterName          string `json:"clusterName" binding:"required"`
		FriendlyName         string `json:"friendlyName" binding:"required"`
		RoleArn              string `json:"roleArn" binding:"required"`
		Endpoint             string `json:"endpoint" binding:"required"`
		CertificateAuthority string `json:"certificateAuthority"` // Optional, will be fetched if not provided
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Create the cluster secret (will fetch CA cert if not provided)
	err := k8s.CreateClusterSecret(request.ClusterName, request.FriendlyName, request.RoleArn, request.Endpoint, request.CertificateAuthority)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Cluster added successfully"})
}

// DeleteCluster deletes a cluster
func (h *Handlers) DeleteCluster(c *gin.Context) {
	clusterName := c.Param("name")

	// Delete the cluster secret
	err := k8s.DeleteClusterSecret(clusterName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Cluster deleted successfully"})
}

// GetAllJobs returns all jobs managed by spawnr across all namespaces
func (h *Handlers) GetAllJobs(c *gin.Context) {
	h.clientMu.RLock()
	client := h.k8sClient
	h.clientMu.RUnlock()

	jobs, err := client.ListAllSpawnrJobs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, jobs)
}
