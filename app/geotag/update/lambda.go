package update

import (
	"context"
	"fmt"
	
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/sfomuseum/go-sfomuseum-geo/geotag"
)

func runLambda(ctx context.Context, opts *geotag.UpdateDepictionOptions) error {

	handler := func(ctx context.Context, update *geotag.Depiction) error {

		_, err := geotag.UpdateDepiction(ctx, opts, update)

		if err != nil {
			return fmt.Errorf("Failed to update depiction %d, %v", update.DepictionId, err)
		}

		return nil
	}

	lambda.Start(handler)
	return nil
}
