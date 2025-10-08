package remove

import (
	"context"
	"fmt"

	"github.com/sfomuseum/go-sfomuseum-geo/geotag"
)

func runCommandLine(ctx context.Context, opts *geotag.RemoveGeotagDepictionOptions) error {

	for _, depiction_id := range depictions {

		update := &geotag.Depiction{
			DepictionId: depiction_id,
		}

		_, err := geotag.RemoveGeotagDepiction(ctx, opts, update)

		if err != nil {
			return fmt.Errorf("Failed to remove geotag depiction %d, %v", depiction_id, err)
		}
	}

	return nil
}
