package api

import (
	"server-master/internal/service"

	"github.com/gin-gonic/gin"
)

// Router defines the interface for modules that can register their own routes.
type Router interface {
	Register(r *gin.RouterGroup)
}

func NewRouter(routers ...Router) *gin.Engine {
	r := gin.New()
	
	// Use standard middlewares
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// Create a root group to pass to routers
	root := r.Group("/")
	for _, router := range routers {
		router.Register(root)
	}

	return r
}

// NewDefaultRouter creates a router with all standard handlers initialized.
func NewDefaultRouter(svcs *service.Container) *gin.Engine {
	return NewRouter(
		NewSubHandler(svcs.Subscription),
		NewFileHandler(svcs.File),
	)
}
