package api

import (
	"context"
	"fmt"
	"net/http"
	"server-master/internal/config"
	"server-master/internal/model"
	"strconv"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

// SubscriptionService defines the interface for subscription management.
type SubscriptionService interface {
	GenerateConfig(ctx context.Context) (*model.ClashConfig, string, error)
	ValidateToken(token string) bool
	GetConfig() config.SubscriptionConfig
}

type SubHandler struct {
	service SubscriptionService
}

func NewSubHandler(s SubscriptionService) *SubHandler {
	return &SubHandler{service: s}
}

// Register registers the subscription routes to the router.
func (h *SubHandler) Register(r *gin.RouterGroup) {
	sub := r.Group("/sub")
	sub.Use(h.AuthMiddleware())
	{
		sub.GET("", h.Handle)
	}
}

// AuthMiddleware validates the subscription token before proceeding.
func (h *SubHandler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Query("token")
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing subscription token"})
			return
		}

		if !h.service.ValidateToken(token) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid subscription token"})
			return
		}
		c.Next()
	}
}

func (h *SubHandler) Handle(c *gin.Context) {
	config, userInfo, err := h.service.GenerateConfig(c.Request.Context())
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate configuration"})
		return
	}

	h.setClashHeaders(c, userInfo)

	c.Status(http.StatusOK)
	c.Header("Content-Type", "application/yaml; charset=utf-8")
	if err := yaml.NewEncoder(c.Writer).Encode(config); err != nil {
		_ = c.Error(err)
	}
}

func (h *SubHandler) setClashHeaders(c *gin.Context, userInfo string) {
	cfg := h.service.GetConfig()
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", cfg.Filename))
	c.Header("profile-update-interval", strconv.Itoa(cfg.UpdateInterval))
	c.Header("profile-web-page-url", cfg.ProfileURL)
	if userInfo != "" {
		c.Header("Subscription-Userinfo", userInfo)
	}
}
