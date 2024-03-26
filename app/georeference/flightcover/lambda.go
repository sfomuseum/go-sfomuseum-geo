package flightcover

import (
	"context"
	"fmt"
	
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/sfomuseum/go-sfomuseum-geo/georeference"
)

func runLambda(ctx context.Context, opts *georeference.AssignReferencesOptions) error {

	handler := func(ctx context.Context, flightcover_refs *georeference.FlightCoverReferences) error {

		_, err := georeference.AssignFlightCoverReferences(ctx, opts, flightcover_refs)

		if err != nil {
			return fmt.Errorf("Failed to assign references for depiction %d, %v", -1, err)
		}

		return fmt.Errorf("Not implemented")
	}

	lambda.Start(handler)
	return nil
}
