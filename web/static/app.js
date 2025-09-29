class SpawnrApp {
    constructor() {
        this.currentCluster = '';
        this.currentProfile = '';
        this.currentNamespace = '';
        this.currentDeployment = '';
        this.jobs = new Map();
        this.init();
    }

    async init() {
        await this.loadClusters();
        this.setupEventListeners();
        // Load existing jobs on page load
        await this.loadAllJobs();
    }

    setupEventListeners() {
        document.getElementById('clusterSelect').addEventListener('change', (e) => {
            const selectedOption = e.target.options[e.target.selectedIndex];
            this.currentCluster = e.target.value;
            this.currentProfile = selectedOption.dataset.profile || '';
            this.switchCluster();
        });

        document.getElementById('namespaceSelect').addEventListener('change', (e) => {
            this.currentNamespace = e.target.value;
            this.loadDeployments();
        });

        document.getElementById('deploymentSelect').addEventListener('change', (e) => {
            this.currentDeployment = e.target.value;
            this.updateCreateJobButton();
        });

        document.getElementById('createJobBtn').addEventListener('click', () => {
            this.createJob();
        });

        document.getElementById('saveClusterBtn').addEventListener('click', () => {
            this.addCluster();
        });

        document.getElementById('refreshJobsBtn').addEventListener('click', () => {
            this.refreshJobs();
        });

        // Add job name sanitization on blur (when user leaves the field)
        const jobNameInput = document.getElementById('jobName');
        if (jobNameInput) {
            jobNameInput.addEventListener('blur', (e) => {
                if (e.target.value) {
                    e.target.value = this.sanitizeJobName(e.target.value);
                }
            });
        }
    }

    async loadClusters() {
        try {
            const response = await fetch('/api/clusters');
            const clusters = await response.json();
            
            const select = document.getElementById('clusterSelect');
            
            if (clusters.length === 0) {
                select.innerHTML = '<option value="">No EKS clusters found in your AWS account</option>';
                this.showAlert('No EKS clusters found in your AWS account. Please create an EKS cluster first.', 'info');
            } else {
                select.innerHTML = '<option value="">Select a cluster</option>';
                clusters.forEach(cluster => {
                    const option = document.createElement('option');
                    option.value = cluster.originalName || cluster.name;
                    option.dataset.profile = cluster.profile || '';
                    option.textContent = `${cluster.name} (${cluster.region})`;
                    select.appendChild(option);
                });
                
                // Auto-select the first cluster if there's only one
                if (clusters.length === 1) {
                    const firstCluster = clusters[0];
                    select.value = firstCluster.originalName || firstCluster.name;
                    this.currentCluster = firstCluster.originalName || firstCluster.name;
                    this.currentProfile = firstCluster.profile || '';
                    await this.switchCluster(false); // Don't show notification for auto-select
                }
            }
        } catch (error) {
            console.error('Failed to load clusters:', error);
            this.showAlert('Failed to load clusters. Make sure AWS CLI is configured and you have EKS clusters in your account.', 'danger');
        }
    }

    async switchCluster(showNotification = true) {
        if (!this.currentCluster) {
            document.getElementById('namespaceSelect').innerHTML = '<option value="">Select a cluster first</option>';
            document.getElementById('namespaceSelect').disabled = true;
            return;
        }

        try {
            const response = await fetch('/api/clusters/switch', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    clusterName: this.currentCluster
                })
            });

            if (response.ok) {
                await this.loadNamespaces();
                if (showNotification) {
                    this.showAlert(`Switched to cluster: ${this.currentCluster}`, 'success');
                }
            } else {
                const error = await response.json();
                this.showAlert(`Failed to switch cluster: ${error.error}`, 'danger');
            }
        } catch (error) {
            console.error('Failed to switch cluster:', error);
            this.showAlert('Failed to switch cluster', 'danger');
        }
    }

    sanitizeJobName(name) {
        // Convert to lowercase
        name = name.toLowerCase();
        
        // Replace spaces and underscores with hyphens
        name = name.replace(/[\s_]+/g, '-');
        
        // Remove any characters that aren't alphanumeric or hyphens
        name = name.replace(/[^a-z0-9-]/g, '');
        
        // Remove leading/trailing hyphens
        name = name.replace(/^-+|-+$/g, '');
        
        // Ensure it starts with an alphanumeric character
        name = name.replace(/^[^a-z0-9]+/, '');
        
        // Kubernetes names must be max 63 characters
        if (name.length > 63) {
            name = name.substring(0, 63);
        }
        
        // Remove trailing hyphens again after truncation
        name = name.replace(/-+$/, '');
        
        return name;
    }

    async loadAllJobs() {
        try {
            const response = await fetch('/api/jobs');
            if (response.ok) {
                const jobs = await response.json();
                const container = document.getElementById('jobsContainer');
                container.innerHTML = '';
                
                if (jobs.length === 0) {
                    container.innerHTML = '<p class="text-center text-muted">No jobs created yet</p>';
                } else {
                    jobs.forEach(job => this.addJobCard(job));
                }
            }
        } catch (error) {
            console.error('Failed to load jobs:', error);
        }
    }

    async loadNamespaces() {
        try {
            const response = await fetch('/api/namespaces');
            const namespaces = await response.json();
            
            const select = document.getElementById('namespaceSelect');
            select.innerHTML = '<option value="">Select a namespace</option>';
            select.disabled = false;
            
            namespaces.forEach(ns => {
                const option = document.createElement('option');
                option.value = ns.metadata.name;
                option.textContent = ns.metadata.name;
                select.appendChild(option);
            });
        } catch (error) {
            console.error('Failed to load namespaces:', error);
            this.showAlert('Failed to load namespaces', 'danger');
        }
    }

    async loadDeployments() {
        if (!this.currentNamespace) {
            document.getElementById('deploymentSelect').innerHTML = '<option value="">Select a namespace first</option>';
            document.getElementById('deploymentSelect').disabled = true;
            return;
        }

        try {
            const response = await fetch(`/api/deployments?namespace=${this.currentNamespace}`);
            const deployments = await response.json();
            
            const select = document.getElementById('deploymentSelect');
            select.innerHTML = '<option value="">Select a deployment</option>';
            select.disabled = false;
            
            deployments.forEach(deployment => {
                const option = document.createElement('option');
                option.value = deployment.metadata.name;
                option.textContent = deployment.metadata.name;
                select.appendChild(option);
            });
        } catch (error) {
            console.error('Failed to load deployments:', error);
            this.showAlert('Failed to load deployments', 'danger');
        }
    }

    updateCreateJobButton() {
        const jobName = document.getElementById('jobName').value;
        const command = document.getElementById('command').value;
        const createBtn = document.getElementById('createJobBtn');
        
        createBtn.disabled = !(this.currentNamespace && this.currentDeployment && jobName && command);
    }

    async createJob() {
        let jobName = document.getElementById('jobName').value;
        const command = document.getElementById('command').value;

        if (!jobName || !command) {
            this.showAlert('Please fill in all fields', 'warning');
            return;
        }

        // Sanitize the job name before submission
        jobName = this.sanitizeJobName(jobName);
        if (!jobName) {
            this.showAlert('Please provide a valid job name', 'warning');
            return;
        }

        const createBtn = document.getElementById('createJobBtn');
        const originalText = createBtn.innerHTML;
        createBtn.innerHTML = '<span class="spinner-border spinner-border-sm" role="status"></span> Creating...';
        createBtn.disabled = true;

        try {
            const response = await fetch('/api/jobs', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    namespace: this.currentNamespace,
                    deployment: this.currentDeployment,
                    jobName: jobName,
                    command: command
                })
            });

            if (response.ok) {
                const job = await response.json();
                this.showAlert('Job created successfully!', 'success');
                this.addJobCard(job);
                this.clearForm();
            } else {
                const error = await response.json();
                this.showAlert(`Failed to create job: ${error.error}`, 'danger');
            }
        } catch (error) {
            console.error('Failed to create job:', error);
            this.showAlert('Failed to create job', 'danger');
        } finally {
            createBtn.innerHTML = originalText;
            createBtn.disabled = false;
        }
    }

    addJobCard(job) {
        const container = document.getElementById('jobsContainer');
        
        // Remove "no jobs" message if it exists
        const noJobsMsg = container.querySelector('.text-center.text-muted');
        if (noJobsMsg) {
            noJobsMsg.remove();
        }

        const jobCard = document.createElement('div');
        jobCard.className = 'card job-card';
        jobCard.id = `job-${job.metadata.name}`;
        
        const status = this.getJobStatus(job);
        const statusClass = this.getStatusClass(status);
        
        jobCard.innerHTML = `
            <div class="card-body">
                <div class="d-flex justify-content-between align-items-start">
                    <div>
                        <h6 class="card-title">${job.metadata.name}</h6>
                        <p class="card-text">
                            <small class="text-muted">
                                Namespace: ${job.metadata.namespace} | 
                                Created: ${new Date(job.metadata.creationTimestamp).toLocaleString()}
                            </small>
                        </p>
                    </div>
                    <div>
                        <span class="badge ${statusClass}">${status}</span>
                    </div>
                </div>
                <div class="mt-2">
                    <button class="btn btn-sm btn-outline-primary me-2" onclick="app.viewJobLogs('${job.metadata.namespace}', '${job.metadata.name}')">
                        <i class="fas fa-file-alt"></i> View Logs
                    </button>
                    <button class="btn btn-sm btn-outline-danger" onclick="app.deleteJob('${job.metadata.namespace}', '${job.metadata.name}')">
                        <i class="fas fa-trash"></i> Delete
                    </button>
                </div>
            </div>
        `;
        
        container.appendChild(jobCard);
    }

    getJobStatus(job) {
        if (job.status.succeeded > 0) return 'Succeeded';
        if (job.status.failed > 0) return 'Failed';
        if (job.status.active > 0) return 'Running';
        return 'Pending';
    }

    getStatusClass(status) {
        switch (status) {
            case 'Succeeded': return 'bg-success';
            case 'Failed': return 'bg-danger';
            case 'Running': return 'bg-primary';
            default: return 'bg-secondary';
        }
    }

    async viewJobLogs(namespace, name) {
        const modal = new bootstrap.Modal(document.getElementById('logsModal'));
        const logContent = document.getElementById('logContent');
        
        logContent.textContent = 'Loading logs...';
        modal.show();

        try {
            const response = await fetch(`/api/jobs/${namespace}/${name}/logs`);
            if (response.ok) {
                const data = await response.json();
                // Preserve line breaks by using a pre element or setting white-space
                logContent.textContent = data.logs || 'No logs available';
                // Ensure line breaks are preserved
                logContent.style.whiteSpace = 'pre-wrap';
            } else {
                logContent.textContent = 'Failed to load logs';
            }
        } catch (error) {
            console.error('Failed to load logs:', error);
            logContent.textContent = 'Error loading logs';
        }
    }

    async deleteJob(namespace, name) {
        if (!confirm(`Are you sure you want to delete job "${name}"?`)) {
            return;
        }

        try {
            const response = await fetch(`/api/jobs/${namespace}/${name}`, {
                method: 'DELETE'
            });

            if (response.ok) {
                this.showAlert('Job deleted successfully', 'success');
                const jobCard = document.getElementById(`job-${name}`);
                if (jobCard) {
                    jobCard.remove();
                }
            } else {
                const error = await response.json();
                this.showAlert(`Failed to delete job: ${error.error}`, 'danger');
            }
        } catch (error) {
            console.error('Failed to delete job:', error);
            this.showAlert('Failed to delete job', 'danger');
        }
    }

    clearForm() {
        document.getElementById('jobName').value = '';
        document.getElementById('command').value = '';
        this.updateCreateJobButton();
    }

    showAlert(message, type) {
        // Create or get alert container
        let alertContainer = document.getElementById('alertContainer');
        if (!alertContainer) {
            alertContainer = document.createElement('div');
            alertContainer.id = 'alertContainer';
            alertContainer.style.cssText = 'position: fixed; top: 70px; left: 50%; transform: translateX(-50%); z-index: 9999; width: 50%; min-width: 300px; max-width: 600px;';
            document.body.appendChild(alertContainer);
        }

        const alertDiv = document.createElement('div');
        alertDiv.className = `alert alert-${type} alert-dismissible fade show mb-2`;
        alertDiv.innerHTML = `
            ${message}
            <button type="button" class="btn-close" data-bs-dismiss="alert"></button>
        `;
        
        alertContainer.appendChild(alertDiv);
        
        // Auto-dismiss after 3 seconds
        setTimeout(() => {
            if (alertDiv.parentNode) {
                alertDiv.classList.remove('show');
                setTimeout(() => alertDiv.remove(), 150);
            }
        }, 3000);
    }

    async refreshJobs() {
        try {
            await this.loadAllJobs();
            this.showAlert('Jobs refreshed', 'success');
        } catch (error) {
            console.error('Failed to refresh jobs:', error);
            this.showAlert('Failed to refresh jobs', 'danger');
        }
    }

    async addCluster() {
        const clusterName = document.getElementById('clusterName').value;
        const friendlyName = document.getElementById('friendlyName').value;
        const roleArn = document.getElementById('roleArn').value;
        const endpoint = document.getElementById('endpoint').value;

        if (!clusterName || !friendlyName || !roleArn || !endpoint) {
            this.showAlert('Please fill in all fields', 'danger');
            return;
        }

        try {
            const response = await fetch('/api/clusters', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    clusterName: clusterName,
                    friendlyName: friendlyName,
                    roleArn: roleArn,
                    endpoint: endpoint
                })
            });

            if (response.ok) {
                this.showAlert('Cluster added successfully', 'success');
                // Close the modal
                const modal = bootstrap.Modal.getInstance(document.getElementById('addClusterModal'));
                modal.hide();
                // Clear the form
                document.getElementById('addClusterForm').reset();
                // Reload clusters
                await this.loadClusters();
            } else {
                const error = await response.json();
                this.showAlert(`Failed to add cluster: ${error.error}`, 'danger');
            }
        } catch (error) {
            console.error('Failed to add cluster:', error);
            this.showAlert('Failed to add cluster', 'danger');
        }
    }
}

// Initialize the app when the page loads
document.addEventListener('DOMContentLoaded', () => {
    window.app = new SpawnrApp();
    
    // Add event listeners for form validation
    document.getElementById('jobName').addEventListener('input', () => {
        window.app.updateCreateJobButton();
    });
    
    document.getElementById('command').addEventListener('input', () => {
        window.app.updateCreateJobButton();
    });
});
