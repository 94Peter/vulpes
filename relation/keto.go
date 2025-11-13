// Package relation provides a client for Ory Keto, an open-source authorization server.
// It simplifies the process of creating and managing relation tuples for access control.
package relation

import (
	"context"
	"fmt"
	"sync"

	"github.com/94peter/vulpes/log"

	pb "github.com/ory/keto/proto/ory/keto/relation_tuples/v1alpha2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	once sync.Once

	writeconn *grpc.ClientConn
	readconn  *grpc.ClientConn

	defaultConfig = config{}
)

type Option func(*config)

func WithWriteAddr(addr string) Option {
	return func(c *config) {
		c.writeAddr = addr
	}
}

func WithReadAddr(addr string) Option {
	return func(c *config) {
		c.readAddr = addr
	}
}

type config struct {
	writeAddr string
	readAddr  string
}

// "keto.dev.orb.local:4467"
// "keto.dev.orb.local:4466"
func Initialize(opts ...Option) {
	once.Do(func() {
		var err error
		for _, opt := range opts {
			opt(&defaultConfig)
		}
		if defaultConfig.writeAddr == "" && defaultConfig.readAddr == "" {
			panic("keto service address is not set")
		}
		if defaultConfig.writeAddr != "" {
			writeconn, err = grpc.NewClient(defaultConfig.writeAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				panic(fmt.Errorf("failed to connect to keto write service: %w", err))
			}
		}
		if defaultConfig.readAddr != "" {
			readconn, err = grpc.NewClient(defaultConfig.readAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				panic(fmt.Errorf("failed to connect to keto read service: %w", err))
			}
		}
	})
}

func Close() {
	if writeconn != nil {
		if err := writeconn.Close(); err != nil {
			log.Error("failed to close keto write connection", log.Err(err))
		}
	}
	if readconn != nil {
		if err := readconn.Close(); err != nil {
			log.Error("failed to close keto read connection", log.Err(err))
		}
	}
}

func WriteTuple(ctx context.Context, tuples tupleBuilder) error {
	if writeconn == nil {
		return ErrWriteConnectNotInitialed
	}

	writeClient := pb.NewWriteServiceClient(writeconn)

	_, err := writeClient.TransactRelationTuples(ctx, &pb.TransactRelationTuplesRequest{
		RelationTupleDeltas: tuples,
	})
	if err != nil {
		return fmt.Errorf("%w: %w", ErrWriteFailed, err)
	}
	return nil
}
