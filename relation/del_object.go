package relation

import (
	"context"
	"fmt"

	pb "github.com/ory/keto/proto/ory/keto/relation_tuples/v1alpha2"
)

func DeleteObjectId(ctx context.Context, namespace, objectId string) error {
	if writeconn == nil {
		return ErrWriteConnectNotInitialed
	}
	writeClient := pb.NewWriteServiceClient(writeconn)
	_, err := writeClient.DeleteRelationTuples(ctx, &pb.DeleteRelationTuplesRequest{
		RelationQuery: &pb.RelationQuery{
			Namespace: &namespace,
			Object:    &objectId,
		},
	})
	if err != nil {
		return fmt.Errorf("%w: %w", ErrReadFailed, err)
	}
	return nil
}
