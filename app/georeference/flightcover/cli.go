package flightcover

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/sfomuseum/go-sfomuseum-geo/georeference"
	"os"
)

func runCommandLine(ctx context.Context, opts *georeference.AssignReferencesOptions) error {

	var flightcover_refs *georeference.FlightCoverReferences

	dec := json.NewDecoder(os.Stdin)
	err := dec.Decode(&flightcover_refs)

	if err != nil {
		return fmt.Errorf("Failed to decode references, %v", err)
	}

	_, err = georeference.AssignFlightCoverReferences(ctx, opts, flightcover_refs)

	if err != nil {
		return fmt.Errorf("Failed to assign flight cover references, %w", err)
	}

	return nil
}
