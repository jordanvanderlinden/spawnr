package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handlers) Index(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{
		"title": "Spawnr - Kubernetes Job Manager",
	})
}
