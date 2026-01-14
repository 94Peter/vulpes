package ezapi

import (
	"context"
	"errors"
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

type ctxKey string

const (
	CtxKeyCSRFToken = ctxKey("gorilla.csrf.Token")

	defaultPort              = 8080
	defaultReadHeaderTimeout = 3 * time.Second
	defaultWaitDuration      = 5 * time.Second
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
	Routers RouterGroup
}

func (cfg *config) initSession() error {
	if cfg.Session.Enable {
		if cfg.Session.Store == "" {
			return errors.New("session store is required")
		}
		if cfg.Session.CookieName == "" {
			return errors.New("session name is required")
		}
		if cfg.Session.MaxAge == 0 {
			return errors.New("session max age is required")
		}
		if len(cfg.Session.KeyPairs) == 0 {
			return errors.New("session key pairs is required")
		}
		myStore = store.NewStore(cfg.Session.Store, cfg.Session.MaxAge, cfg.Session.KeyPairs...)
		if myStore == nil {
			return errors.New("session store is not supported: " + cfg.Session.Store)
		}

		for _, injector := range sessionInjectors {
			injector.InjectSessionStore(myStore, cfg.Session.CookieName)
		}
		cfg.prependMiddlewares(sessions.Sessions(cfg.Session.CookieName, myStore))
	}
	return nil
}

func (cfg *config) initCSRF() error {
	if cfg.CSRF.Enable {
		if !cfg.Session.Enable {
			return errors.New("csrf requires session")
		}
		if cfg.CSRF.Secret == "" {
			return errors.New("csrf secret is required")
		}
		if cfg.CSRF.FieldName == "" {
			return errors.New("csrf field name is required")
		}
		cfg.appendMiddlewares(
			csrfProtectionWithExclusion(
				csrf.Middleware(
					csrf.Options{
						Secret: cfg.CSRF.Secret,
						ErrorFunc: func(c *gin.Context) {
							c.String(http.StatusBadRequest, "CSRF token mismatch")
							log.Warn("csrf token mismatch")
							c.Abort()
						},
					},
				),
				cfg.CSRF.ExcludePaths,
			),
			func(c *gin.Context) {
				if !slices.Contains(cfg.CSRF.ExcludePaths, c.Request.URL.Path) {
					c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), CtxKeyCSRFToken, csrf.GetToken(c)))
				}
				c.Next()
			},
		)
	}
	return nil
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
	routers = NewRouterGroup()
	// defaultMiddelware is the set of default middleware used by the gin engine.
	defaultMiddelware = []gin.HandlerFunc{
		gin.Recovery(),
		gin.Logger(),
	}
	myStore store.Store

	defaultConfig = config{
		Port: defaultPort,
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
func RegisterGinApi(f func(router RouterGroup)) {
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
		// 評估留一種方法即可
		routers.register(engine)
		// 擴充從Cfx可以註冊Router
		if cfg.Routers != nil {
			cfg.Routers.register(engine)
		}
	})
}

// server creates and configures an *http.Server with the gin engine as its handler.
func server(cfg *config) *http.Server {
	portStr := fmt.Sprintf(":%d", cfg.Port)
	log.Info("api service listen on port " + portStr)
	var err error
	if err = cfg.initSession(); err != nil {
		panic(err)
	}

	if err = cfg.initCSRF(); err != nil {
		panic(err)
	}

	if cfg.Tracer.Enable {
		cfg.prependMiddlewares(otelgin.Middleware("API Server"))
	}
	initEngin(cfg)

	return &http.Server{
		Addr:              portStr,
		ReadHeaderTimeout: defaultReadHeaderTimeout,
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
				time.Sleep(defaultWaitDuration)
			} else if err == http.ErrServerClosed {
				return
			}
		}
	}(ser)
	<-ctx.Done()
	ctx, cancel := context.WithTimeout(context.Background(), defaultWaitDuration)
	defer cancel()
	if err := ser.Shutdown(ctx); err != nil {
		log.Error("Server forced to shutdown failed: " + err.Error())
	}
	apiWait.Wait()
	return nil
}
