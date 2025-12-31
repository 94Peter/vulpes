package cache

import (
	"context"
	"sync"
	"time"

	"github.com/94peter/vulpes/constant"

	"github.com/go-redis/redis/v8"
)

const (
	defaultPoolSize     = 10
	defaultMinIdleConns = 3
	defaultDialTime     = constant.DefaultTimeout
	defaultReadTimeout  = 3 * time.Second
	defaultWriteTimeout = defaultReadTimeout
	defaultIdleTimeout  = constant.DefaultIdleTimeout
	defaultPoolTimeout  = constant.DefaultIdleTimeout
)

var (
	conn *redis.Client
	once sync.Once

	defaultOptions = &redis.Options{
		PoolSize:     defaultPoolSize,
		MinIdleConns: defaultMinIdleConns,
		DialTimeout:  defaultDialTime,
		ReadTimeout:  defaultReadTimeout,
		WriteTimeout: defaultWriteTimeout,
		PoolTimeout:  defaultPoolTimeout,
		IdleTimeout:  defaultIdleTimeout,
	}
)

type initConnOpt func(*redis.Options)

func WithAddr(addr string) initConnOpt {
	return func(o *redis.Options) {
		o.Addr = addr
	}
}

func WithDb(db int) initConnOpt {
	return func(o *redis.Options) {
		o.DB = db
	}
}

func WithPassword(password string) initConnOpt {
	return func(o *redis.Options) {
		o.Password = password
	}
}

func WithUsername(username string) initConnOpt {
	return func(o *redis.Options) {
		o.Username = username
	}
}

func InitConnection(opts ...initConnOpt) error {
	if conn != nil {
		return nil
	}
	once.Do(func() {
		for _, opt := range opts {
			opt(defaultOptions)
		}
		conn = redis.NewClient(defaultOptions)
	})
	ctx, cancel := context.WithTimeout(context.Background(), constant.DefaultTimeout)
	defer cancel()
	if conn.Ping(ctx).Val() != "PONG" {
		return ErrCacheNotConnected
	}
	return nil
}

func Close() error {
	if conn != nil {
		return conn.Close()
	}
	return nil
}
