# ezapi Package

The `ezapi` package provides a simplified and streamlined way to create, manage, and run a web server using the [Gin](https://github.com/gin-gonic/gin) framework. It abstracts away the boilerplate code for server setup, route registration, and graceful shutdown.

## Features

- **Decentralized Route Registration**: Register API routes from different parts of your application using `init()` functions.
- **Graceful Shutdown**: The server handles `context` cancellation to shut down gracefully.
- **Singleton Engine**: Uses a singleton pattern for the Gin engine, ensuring consistency.
- **Default Middleware**: Comes with `gin.Recovery()` and `gin.Logger()` pre-configured.
- **Easy Integration**: Can be run as a standalone service or its `http.Handler` can be integrated into another Go HTTP server.

## Usage

The recommended pattern is to define routes in their own packages and use `init()` functions to register them. The `main` package then imports these packages for their side effects (i.e., to trigger the `init()` functions).

### 1. Define Routes in a Separate Package

Create a package (e.g., `api`) to define your handlers and register your routes. This keeps your code organized.

**File: `api/routes.go`**
```go
package api

import (
	"net/http"

	"github.com/94peter/vulpes/ezapi"
	"github.com/gin-gonic/gin"
)

// Use the init() function to register routes.
// This function is automatically called when this package is imported.
func init() {
	ezapi.RegisterGinApi(func(router ezapi.Router) {
		router.GET("/ping", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "pong"})
		})

		router.POST("/users", func(c *gin.Context) {
			// Logic to create a user
			c.JSON(http.StatusCreated, gin.H{"status": "user created"})
		})
	})
}
```

### 2. Run the Server from `main`

In your `main` package, import the `api` package using a blank identifier (`_`). This tells Go to execute the `api` package's `init()` function without needing to reference any of its exported variables or functions.

**File: `main.go`**
```go
package main

import (
	"context"

	"github.com/94peter/vulpes/ezapi"

	// IMPORTANT: Import your routes package with a blank identifier
	// to trigger its init() function for route registration.
	_ "your/project/path/api"
)

func main() {
	// Create a context that can be canceled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Run the server on port 8080.
	// Routes have already been registered via the init() function in the api package.
	if err := ezapi.RunGin(ctx, 8080); err != nil {
		// Handle error
	}
}
```

### 3. Integrating with an Existing Server

If you need to integrate the Gin engine into an existing `http.Server`, you can get the handler. The routes registered in your `api` package's `init()` will be included.

```go
package main

import (
    "net/http"
    "github.com/94peter/vulpes/ezapi"
    _ "your/project/path/api" // Don't forget to import for side effects
)

func main() {
    // Get the configured Gin engine as an http.Handler
    handler := ezapi.GetHttpHandler()

    // Use it with your own http.Server instance
    myServer := &http.Server{
        Addr:    ":8080",
        Handler: handler,
    }
    // ... run your server
}
```
