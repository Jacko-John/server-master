package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// FileService defines the interface for static rule file serving.
type FileService interface {
	GetFilePath(filename string) (string, error)
}

type FileHandler struct {
	service FileService
}

func NewFileHandler(s FileService) *FileHandler {
	return &FileHandler{service: s}
}

// Register registers the file routes to the router.
func (h *FileHandler) Register(r *gin.RouterGroup) {
	r.GET("/file/:filename", h.Handle)
}

func (h *FileHandler) Handle(c *gin.Context) {
	filename := c.Param("filename")
	if filename == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing filename"})
		return
	}

	path, err := h.service.GetFilePath(filename)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.File(path)
}
