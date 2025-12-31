package main

import (
	"context"
	"fmt"
	"time"

	"github.com/94peter/vulpes/log"

	pb "github.com/ory/keto/proto/ory/keto/relation_tuples/v1alpha2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// GrpcKeto is an example function demonstrating how to interact with the Keto Read API.
// It is not intended for production use but serves as a useful reference.
func main() {
	// Create a gRPC connection to the Keto Read API.
	conn, err := grpc.NewClient("keto.dev.orb.local:4466", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Error("failed to connect to keto", log.Err(err))
		panic(err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Error("failed to close keto connection", log.Err(err))
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create a new ReadServiceClient.
	readClt := pb.NewReadServiceClient(conn)
	resp, err := readClt.ListRelationTuples(ctx, &pb.ListRelationTuplesRequest{
		Query: &pb.ListRelationTuplesRequest_Query{
			Namespace: "Image",
			Relation:  "viewer",
			Subject: &pb.Subject{
				Ref: &pb.Subject_Set{
					Set: &pb.SubjectSet{
						Namespace: "User",
						Object:    "user:kkkkkk",
					},
				},
			},
		},
	})
	if err != nil {
		log.Error("failed to list relation tuples", log.Err(err))
		panic(err)
	}

	fmt.Println("🔍 Query Result:")
	for _, t := range resp.RelationTuples {
		if t.Subject.GetId() != "" {
			fmt.Printf(" - %s is a %s of %s#%s\n",
				t.Subject.GetId(), t.Relation, t.Namespace, t.Object)
		} else if t.Subject.GetSet() != nil {
			fmt.Printf(" - members of %s:%s are %s of %s#%s\n",
				t.Subject.GetSet().Namespace, t.Subject.GetSet().Object, t.Relation, t.Namespace, t.Object)
		}
	}
}
