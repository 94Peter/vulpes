package ezapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type option func(*config)

func WithPort(port int) option {
	return func(c *config) {
		c.Port = uint16(port)
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

func WithMiddleware(middleware ...gin.HandlerFunc) option {
	return func(c *config) {
		c.Middlewares = append(defaultMiddelware, middleware...)
	}
}

func WithStaticFS(path string, fs http.FileSystem) option {
	return func(c *config) {
		c.StaticFS[path] = fs
	}
}
