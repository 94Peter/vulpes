package ezapi

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	csrf "github.com/utrack/gin-csrf"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	"github.com/94peter/vulpes/ezapi/session/store"
	"github.com/94peter/vulpes/log"
)

type config struct {
	Port    uint16
	Session struct {
		Enable     bool
		Store      string
		CookieName string
		MaxAge     int
		KeyPairs   [][]byte
	}
	CSRF struct {
		Enable       bool
		Secret       string
		FieldName    string
		ExcludePaths []string
	}
	StaticFS    map[string]http.FileSystem
	Middlewares []gin.HandlerFunc
	Tracer      struct {
		Enable bool
	}
}

func (c *config) appendMiddlewares(mids ...gin.HandlerFunc) {
	c.Middlewares = append(c.Middlewares, mids...)
}

func (c *config) prependMiddlewares(mids ...gin.HandlerFunc) {
	c.Middlewares = append(mids, c.Middlewares...)
}

var (
	// engine is the singleton gin.Engine instance.
	engine *gin.Engine
	// routers holds all the registered routes before the engine is initialized.
	routers = newRouterGroup()
	// defaultMiddelware is the set of default middleware used by the gin engine.
	defaultMiddelware = []gin.HandlerFunc{
		gin.Recovery(),
		gin.Logger(),
	}
	myStore store.Store

	defaultConfig = config{
		Port: 8080,
		Session: struct {
			Enable     bool
			Store      string
			CookieName string
			MaxAge     int
			KeyPairs   [][]byte
		}{},
		CSRF: struct {
			Enable       bool
			Secret       string
			FieldName    string
			ExcludePaths []string
		}{},
		Middlewares: defaultMiddelware,
		StaticFS:    make(map[string]http.FileSystem),
		Tracer: struct{ Enable bool }{
			Enable: false,
		},
	}
	sessionInjectors []SessionStoreInjector
)

// RegisterGinApi allows for the registration of API routes using a function.
// This function can be called from anywhere to add routes to the central routerGroup.
func RegisterGinApi(f func(router Router)) {
	f(routers)
}

type SessionStoreInjector interface {
	InjectSessionStore(sessionStore store.Store, cookieName string)
}

func RegisterSessionInjector(injector SessionStoreInjector) {
	sessionInjectors = append(sessionInjectors, injector)
}

// initEngin initializes the gin engine as a singleton.
// It sets up the default middleware and registers all the routes that have been collected.
func initEngin(cfg *config) {
	once.Do(func() {
		engine = gin.New()
		for k, fs := range cfg.StaticFS {
			engine.StaticFS(k, fs)
		}
		engine.Use(cfg.Middlewares...)
		routers.register(engine)
	})
}

type contextKey string

const _CtxKeyCSRF = contextKey("gorilla.csrf.Token")

// server creates and configures an *http.Server with the gin engine as its handler.
func server(cfg *config) *http.Server {
	portStr := fmt.Sprintf(":%d", cfg.Port)
	log.Info("api service listen on port " + portStr)
	if cfg.Session.Enable {
		if cfg.Session.Store == "" {
			panic("session store is required")
		}
		if cfg.Session.CookieName == "" {
			panic("session name is required")
		}
		if cfg.Session.MaxAge == 0 {
			panic("session max age is required")
		}
		if len(cfg.Session.KeyPairs) == 0 {
			panic("session key pairs is required")
		}
		myStore = store.NewStore(cfg.Session.Store, cfg.Session.MaxAge, cfg.Session.KeyPairs...)
		if myStore == nil {
			panic("session store is not supported: " + cfg.Session.Store)
		}

		for _, injector := range sessionInjectors {
			injector.InjectSessionStore(myStore, cfg.Session.CookieName)
		}
		cfg.prependMiddlewares(sessions.Sessions(cfg.Session.CookieName, myStore))
	}

	if cfg.CSRF.Enable {
		if !cfg.Session.Enable {
			panic("csrf requires session")
		}
		if cfg.CSRF.Secret == "" {
			panic("csrf secret is required")
		}
		if cfg.CSRF.FieldName == "" {
			panic("csrf field name is required")
		}
		cfg.appendMiddlewares(
			csrfProtectionWithExclusion(
				csrf.Middleware(
					csrf.Options{
						Secret: cfg.CSRF.Secret,
						ErrorFunc: func(c *gin.Context) {
							c.String(400, "CSRF token mismatch")
							log.Warn("csrf token mismatch")
							c.Abort()
						},
					},
				),
				cfg.CSRF.ExcludePaths,
			),
			func(c *gin.Context) {
				if !slices.Contains(cfg.CSRF.ExcludePaths, c.Request.URL.Path) {
					c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), _CtxKeyCSRF, csrf.GetToken(c)))
				}
				c.Next()
			},
		)
	}
	if cfg.Tracer.Enable {
		cfg.prependMiddlewares(otelgin.Middleware("API Server"))
	}
	initEngin(cfg)

	return &http.Server{
		Addr:              portStr,
		ReadHeaderTimeout: 3 * time.Second,
		Handler:           engine,
	}
}

// once ensures that the engine is initialized only once.
var once sync.Once

// GetHttpHandler returns the singleton gin.Engine instance as an http.Handler.
// This allows the gin engine to be used with an existing http.Server.
func GetHttpHandler(opts ...option) http.Handler {
	for _, opt := range opts {
		opt(&defaultConfig)
	}
	initEngin(&defaultConfig)
	return engine
}

// RunGin starts the gin server on the specified port and handles graceful shutdown.
// It blocks until the provided context is canceled.
func RunGin(ctx context.Context, opts ...option) error {
	for _, opt := range opts {
		opt(&defaultConfig)
	}
	ser := server(&defaultConfig)
	var apiWait sync.WaitGroup
	apiWait.Add(1)
	go func(srv *http.Server) {
		defer apiWait.Done()
		for {
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Info("api service listen failed: " + err.Error())
				time.Sleep(5 * time.Second)
			} else if err == http.ErrServerClosed {
				return
			}
		}
	}(ser)
	<-ctx.Done()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := ser.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown failed: " + err.Error())
	}
	apiWait.Wait()
	return nil
}
