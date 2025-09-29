package server

import (
	"spawnr/internal/handlers"

	"github.com/gin-gonic/gin"
)

type Server struct {
	handlers *handlers.Handlers
}

func New(h *handlers.Handlers) *Server {
	return &Server{
		handlers: h,
	}
}

func (s *Server) Run(addr string) error {
	r := gin.Default()

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Serve static files
	r.Static("/static", "./web/static")
	r.LoadHTMLGlob("web/templates/*")

	// Web routes
	r.GET("/", s.handlers.Index)

	// Cluster management
	r.GET("/api/clusters", s.handlers.GetClusters)
	r.POST("/api/clusters/switch", s.handlers.SwitchCluster)
	r.POST("/api/clusters", s.handlers.AddCluster)
	r.GET("/api/clusters/:name", s.handlers.GetClusterInfo)
	r.DELETE("/api/clusters/:name", s.handlers.DeleteCluster)

	// Kubernetes resources
	r.GET("/api/namespaces", s.handlers.GetNamespaces)
	r.GET("/api/deployments", s.handlers.GetDeployments)
	r.GET("/api/deployments/:namespace/:name", s.handlers.GetDeployment)
	r.GET("/api/jobs", s.handlers.GetAllJobs)
	r.POST("/api/jobs", s.handlers.CreateJob)
	r.GET("/api/jobs/:namespace/:name", s.handlers.GetJob)
	r.DELETE("/api/jobs/:namespace/:name", s.handlers.DeleteJob)
	r.GET("/api/jobs/:namespace/:name/logs", s.handlers.GetJobLogs)
	r.GET("/api/jobs/:namespace/:name/watch", s.handlers.WatchJob)

	return r.Run(addr)
}
