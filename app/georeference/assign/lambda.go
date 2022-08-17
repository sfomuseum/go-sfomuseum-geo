package assign

import (
	"context"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/sfomuseum/go-sfomuseum-geo/georeference"
)

func runLambda(ctx context.Context, opts *georeference.AssignReferencesOptions) error {

	type AssignEvent struct {
		DepictionId int64                     `json:"depiction_id"`
		References  []*georeference.Reference `json:"references"`
	}

	handler := func(ctx context.Context, ev AssignEvent) error {

		_, err := georeference.AssignReferences(ctx, opts, ev.DepictionId, ev.References...)

		if err != nil {
			return fmt.Errorf("Failed to assign references for depiction %d, %v", ev.DepictionId, err)
		}

		return fmt.Errorf("Not implemented")
	}

	lambda.Start(handler)
	return nil
}
