package ezapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type option func(*config)

func WithRouterGroup(routerGroup RouterGroup) option {
	return func(c *config) {
		c.Routers = routerGroup
	}
}

func WithPort(port uint16) option {
	return func(c *config) {
		c.Port = port
	}
}

func WithCsrf(enable bool, secret string, fieldname string, excludePaths ...string) option {
	return func(c *config) {
		c.CSRF.Enable = enable
		c.CSRF.Secret = secret
		c.CSRF.FieldName = fieldname
		c.CSRF.ExcludePaths = excludePaths
	}
}

func WithSession(enable bool, store string, cookieName string, maxAge int, keyPairs ...string) option {
	return func(c *config) {
		c.Session.Enable = enable
		c.Session.Store = store
		c.Session.CookieName = cookieName
		c.Session.MaxAge = maxAge
		c.Session.KeyPairs = make([][]byte, len(keyPairs))
		for i, keyPair := range keyPairs {
			c.Session.KeyPairs[i] = []byte(keyPair)
		}
	}
}

func WithMiddleware(middleware ...gin.HandlerFunc) option {
	return func(c *config) {
		c.Middlewares = defaultMiddelware
		c.Middlewares = append(c.Middlewares, middleware...)
	}
}

func WithStaticFS(path string, fs http.FileSystem) option {
	return func(c *config) {
		c.StaticFS[path] = fs
	}
}

func WithTracerEnable(enable bool) option {
	return func(c *config) {
		c.Tracer.Enable = enable
	}
}

func WithLoggerEnable(enable bool) option {
	return func(c *config) {
		c.Logger.Enable = enable
	}
}

func WithMode(mode string) option {
	return func(c *config) {
		c.Mode = mode
	}
}
