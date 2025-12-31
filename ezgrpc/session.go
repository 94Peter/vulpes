// Package ezgrpc provides a simplified setup for gRPC services with a grpc-gateway.
// It includes utilities for handling cookies, sessions, and standard interceptors.
package ezgrpc

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/94peter/vulpes/codec"
	"github.com/94peter/vulpes/log"

	"github.com/gorilla/sessions"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

// contextKey is a custom type for context keys to avoid collisions.
type contextKey int

const (
	// sessionContextKey is the key for storing the session in the context.
	sessionContextKey contextKey = iota
	// requestContextKey is the key for storing the HTTP request in the context.
	requestContextKey

	// setSessionDataKey is the metadata key for setting session data.
	setSessionDataKey = "set-session-data"
	// deleteSessionKey is the metadata key for deleting a session.
	deleteSessionKey = "delete-session"

	// sessionDataKey is the key for storing session data within the session itself.
	sessionDataKey = "grpc-session-key"
)

var (
	// store is the cookie store for sessions, initialized in InitSessionStore.
	store *sessions.CookieStore

	// SessionCookieForwarder is a grpc-gateway option that modifies the response to handle session data.
	SessionCookieForwarder = runtime.WithForwardResponseOption(gatewayResponseModifier)
	// SessionCookieExtractor is a grpc-gateway option that extracts session data from cookies.
	SessionCookieExtractor = runtime.WithMetadata(extractSessionDataFromCookie)

	// sessionName is the name of the session cookie.
	sessionName = "grpc-gateway-session"
	// sessionSecret is the secret key for encrypting session data. In production, use a strong, randomly generated key.
	sessionSecret = "your-very-secret-key"
)

const maxAgeSecond = 600 // 10minutes

// InitSessionStore initializes the session store with a secret key and configures session options.
func InitSessionStore() {
	store = sessions.NewCookieStore([]byte(sessionSecret))
	// Set session options (e.g., Secure, HttpOnly, SameSite)
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   maxAgeSecond,
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
	}
	router.Use(sessionMiddleware)
}

// gatewayResponseModifier modifies the HTTP response based on session-related metadata.
// It handles setting and deleting session data.
func gatewayResponseModifier(ctx context.Context, response http.ResponseWriter, _ proto.Message) error {
	md, ok := runtime.ServerMetadataFromContext(ctx)
	if !ok {
		return nil
	}
	var sessionData []string

	if sessionData = md.HeaderMD.Get(setSessionDataKey); len(sessionData) > 0 {
		data := sessionData[0]
		log.Debugf("set session data: %s", data)
		err := saveSession(ctx, response, data)
		if err != nil {
			return err
		}
	}

	isDeleteSession := getBoolFromServerMetadata(md, deleteSessionKey, false)
	if isDeleteSession {
		err := deleteSession(ctx, response)
		if err != nil {
			return err
		}
	}
	return nil
}

// getSessionFromCtx retrieves the session from the context.
func getSessionFromCtx(ctx context.Context) (*sessions.Session, bool) {
	s, ok := ctx.Value(sessionContextKey).(*sessions.Session)
	return s, ok
}

// getRequestFromCtx retrieves the HTTP request from the context.
func getRequestFromCtx(ctx context.Context) (*http.Request, bool) {
	r, ok := ctx.Value(requestContextKey).(*http.Request)
	return r, ok
}

// getBoolFromServerMetadata retrieves a boolean value from server metadata.
func getBoolFromServerMetadata(md runtime.ServerMetadata, name string, defaultValue bool) bool {
	values := md.HeaderMD.Get(name)
	if len(values) == 0 {
		return defaultValue
	}
	boolString := values[0]
	return strings.ToLower(boolString) == valueTrue
}

var (
	errSessionNotFoundInContext = fmt.Errorf("%w: session not found in context", ErrSessionNotFound)
	errRequestNotFoundInContext = fmt.Errorf("%w: request not found in context", ErrSessionNotFound)
)

// saveSession saves session data to the session store.
func saveSession(ctx context.Context, response http.ResponseWriter, data string) error {
	session, ok := getSessionFromCtx(ctx)
	if !ok {
		return errSessionNotFoundInContext
	}
	req, ok := getRequestFromCtx(ctx)
	if !ok {
		return errRequestNotFoundInContext
	}
	session.Values[sessionDataKey] = data
	err := session.Save(req, response)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrSessionSaveFailed, err)
	}
	return nil
}

// deleteSession deletes a session by setting its max age to -1.
func deleteSession(ctx context.Context, response http.ResponseWriter) error {
	session, ok := getSessionFromCtx(ctx)
	if !ok {
		return errSessionNotFoundInContext
	}

	req, ok := getRequestFromCtx(ctx)
	if !ok {
		return errRequestNotFoundInContext
	}

	// As documented, to delete a session, set its max age to -1.
	session.Options.MaxAge = -1
	session.Options.Path = "/"
	// "Save" the session with max age = -1 to clear it.
	if err := session.Save(req, response); err != nil {
		return fmt.Errorf("%w: %w", ErrSessionSaveFailed, err)
	}
	return nil
}

// extractSessionDataFromCookie is a grpc-gateway function that reads the HTTP session cookie
// and injects the session data into gRPC metadata for the backend.
func extractSessionDataFromCookie(ctx context.Context, _ *http.Request) metadata.MD {
	md := make(metadata.MD)

	session, ok := getSessionFromCtx(ctx)
	if !ok || session == nil {
		return md
	}

	if val, ok := session.Values[sessionDataKey]; ok {
		if strVal, ok := val.(string); ok {
			md.Set(sessionDataKey, strVal)
			log.Debugf("extract session data: %s", strVal)
		}
	}

	return md
}

// sessionMiddleware is an HTTP middleware that retrieves the session from the request
// and stores it in the context for later use.
func sessionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := store.Get(r, sessionName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		ctx := r.Context()
		if ctx == nil {
			ctx = context.Background()
		}
		ctx = context.WithValue(ctx, sessionContextKey, session)
		ctx = context.WithValue(ctx, requestContextKey, r)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// DeleteSession signals that the session should be deleted by setting a metadata key.
func DeleteSession(ctx context.Context) error {
	return grpc.SetHeader(ctx, metadata.Pairs(deleteSessionKey, valueTrue))
}

// SetSessionData encodes a value and sets it as session data in the gRPC metadata.
func SetSessionData[T any](ctx context.Context, value T) error {
	// Encode the value to a string.
	data, err := codec.Encode(value)
	if err != nil {
		return err
	}
	// Set the session data in the metadata.
	return grpc.SetHeader(ctx, metadata.Pairs(setSessionDataKey, data))
}

// GetSessionData retrieves and decodes session data from the incoming gRPC context.
func GetSessionData[T any](ctx context.Context) (T, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return *new(T), fmt.Errorf("%w: can't get metadata from context", ErrSessionNotFound)
	}
	// Retrieve the session data from the metadata.
	data := md.Get(sessionDataKey)
	if len(data) == 0 {
		return *new(T), fmt.Errorf("%w: can't get session data from metadata", ErrSessionNotFound)
	}
	return codec.Decode[T](data[0])
}
