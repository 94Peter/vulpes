package ezapi

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/arwoosa/vulpes/log"
	"github.com/gorilla/csrf"

	"github.com/gin-gonic/gin"
)

type config struct {
	Port uint16
	CSRF struct {
		Enable       bool
		Secret       string
		FieldName    string
		ExcludePaths []string
	}
	StaticFS    map[string]http.FileSystem
	Middlewares []gin.HandlerFunc
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

	defaultConfig = config{
		Port: 8080,
		CSRF: struct {
			Enable       bool
			Secret       string
			FieldName    string
			ExcludePaths []string
		}{},
		Middlewares: defaultMiddelware,
		StaticFS:    make(map[string]http.FileSystem),
	}
)

// RegisterGinApi allows for the registration of API routes using a function.
// This function can be called from anywhere to add routes to the central routerGroup.
func RegisterGinApi(f func(router Router)) {
	f(routers)
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

// server creates and configures an *http.Server with the gin engine as its handler.
func server(cfg *config) *http.Server {
	portStr := fmt.Sprintf(":%d", cfg.Port)
	log.Info("api service listen on port " + portStr)
	if cfg.CSRF.Enable {
		if cfg.CSRF.Secret == "" {
			panic("csrf secret is required")
		}
		if cfg.CSRF.FieldName == "" {
			panic("csrf field name is required")
		}
		isProd := os.Getenv("GIN_MODE") == "release"
		protect := csrf.Protect(
			[]byte(cfg.CSRF.Secret),
			csrf.Secure(isProd),
			csrf.HttpOnly(true),
			csrf.MaxAge(43200), // 12 hours
			csrf.FieldName(cfg.CSRF.FieldName),
		)
		cfg.Middlewares = append(cfg.Middlewares, csrfProtectionWithExclusion(protect, cfg.CSRF.ExcludePaths))
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
