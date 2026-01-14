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
}

type RouterGroup interface {
	Router
	Group(name string) Router
	ToString() string
}

// newRouterGroup creates a new instance of a routerGroup.
func NewRouterGroup() RouterGroup {
	return &routerGroup{}
}

// router represents a single API route with its HTTP method, path, and handler.
type router struct {
	method  string
	path    string
	handler gin.HandlerFunc
}

type routerList []router

// register iterates through the collected routes and applies them to the provided gin.IRouter.
func (rg routerList) register(r gin.IRouter) {
	registerRouter(r, rg)
}

// GET adds a new GET route to the group.
func (rl *routerList) GET(path string, handler gin.HandlerFunc) {
	*rl = append(*rl, router{
		method:  "GET",
		path:    path,
		handler: handler,
	})
}

// POST adds a new POST route to the group.
func (rl *routerList) POST(path string, handler gin.HandlerFunc) {
	*rl = append(*rl, router{
		method:  "POST",
		path:    path,
		handler: handler,
	})
}

// PUT adds a new PUT route to the group.
func (rl *routerList) PUT(path string, handler gin.HandlerFunc) {
	*rl = append(*rl, router{
		method:  "PUT",
		path:    path,
		handler: handler,
	})
}

// DELETE adds a new DELETE route to the group.
func (rl *routerList) DELETE(path string, handler gin.HandlerFunc) {
	*rl = append(*rl, router{
		method:  "DELETE",
		path:    path,
		handler: handler,
	})
}

// routerGroup holds a collection of routes that will be registered with the gin engine.
type routerGroup struct {
	routerList
	group map[string]*routerList
}

func (rg *routerGroup) Group(name string) Router {
	if rg.group == nil {
		rg.group = make(map[string]*routerList)
	}
	newRouterList := &routerList{}
	rg.group[name] = newRouterList
	return newRouterList
}

// register iterates through the collected routes and applies them to the provided gin.IRouter.
func (rg *routerGroup) register(r gin.IRouter) {
	if len(rg.routerList) > 0 {
		registerRouter(r, rg.routerList)
	}
	for name, group := range rg.group {
		ginRouterGroup := r.Group(name)
		registerRouter(ginRouterGroup, *group)

	}
}

func registerRouter(r gin.IRouter, routers []router) {
	for _, router := range routers {
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
	return "routerGroup" + strconv.Itoa(len(rg.routerList))
}
