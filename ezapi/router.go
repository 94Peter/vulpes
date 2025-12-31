package ezapi

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// Router defines the interface for registering routes.
// It supports GET, POST, PUT, and DELETE methods.
type Router interface {
	GET(path string, handler gin.HandlerFunc)
	POST(path string, handler gin.HandlerFunc)
	PUT(path string, handler gin.HandlerFunc)
	DELETE(path string, handler gin.HandlerFunc)
	// register is an internal method to apply the collected routes to a gin.IRouter.
	register(r gin.IRouter)
	ToString() string
}

// newRouterGroup creates a new instance of a routerGroup.
func newRouterGroup() Router {
	return &routerGroup{}
}

// router represents a single API route with its HTTP method, path, and handler.
type router struct {
	method  string
	path    string
	handler gin.HandlerFunc
}

// routerGroup holds a collection of routes that will be registered with the gin engine.
type routerGroup struct {
	routers []*router
}

// GET adds a new GET route to the group.
func (rg *routerGroup) GET(path string, handler gin.HandlerFunc) {
	rg.routers = append(rg.routers, &router{"GET", path, handler})
}

// POST adds a new POST route to the group.
func (rg *routerGroup) POST(path string, handler gin.HandlerFunc) {
	rg.routers = append(rg.routers, &router{"POST", path, handler})
}

// PUT adds a new PUT route to the group.
func (rg *routerGroup) PUT(path string, handler gin.HandlerFunc) {
	rg.routers = append(rg.routers, &router{"PUT", path, handler})
}

// DELETE adds a new DELETE route to the group.
func (rg *routerGroup) DELETE(path string, handler gin.HandlerFunc) {
	rg.routers = append(rg.routers, &router{"DELETE", path, handler})
}

// register iterates through the collected routes and applies them to the provided gin.IRouter.
func (rg *routerGroup) register(r gin.IRouter) {
	for _, router := range rg.routers {
		switch router.method {
		case "GET":
			r.GET(router.path, router.handler)
		case "POST":
			r.POST(router.path, router.handler)
		case "PUT":
			r.PUT(router.path, router.handler)
		case "DELETE":
			r.DELETE(router.path, router.handler)
		}
	}
}

// ToString returns a string representation of the routerGroup, including the number of routes.
func (rg *routerGroup) ToString() string {
	return "routerGroup" + strconv.Itoa(len(rg.routers))
}
