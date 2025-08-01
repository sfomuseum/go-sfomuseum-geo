package update

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	geojson "github.com/sfomuseum/go-geojson-geotag/v2"
	"github.com/sfomuseum/go-sfomuseum-geo/geotag"
)

func runCommandLine(ctx context.Context, opts *geotag.GeotagDepictionOptions) error {

	var f *geojson.GeotagFeature

	dec := json.NewDecoder(os.Stdin)
	err := dec.Decode(&f)

	if err != nil {
		return fmt.Errorf("Failed to decode geotag feature, %v", err)
	}

	for _, depiction_id := range depictions {

		update := &geotag.Depiction{
			DepictionId: depiction_id,
			Feature:     f,
		}

		_, err := geotag.GeotagDepiction(ctx, opts, update)

		if err != nil {
			return fmt.Errorf("Failed to geotag depiction %d, %v", depiction_id, err)
		}
	}

	return nil
}
