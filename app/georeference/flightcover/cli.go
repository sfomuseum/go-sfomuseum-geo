package flightcover

import (
	"context"
	"fmt"
	"github.com/sfomuseum/go-sfomuseum-geo/georeference"
	"github.com/whosonfirst/go-whosonfirst-id"
)

func runCommandLine(ctx context.Context, opts *georeference.AssignReferencesOptions) error {

	count := len(depictions)

	if count == 0 {
		return fmt.Errorf("Nothing to depict")
	}

	for _, depiction_id := range depictions {

		ref := &georeference.FlightCoverReferences{
			Id: depiction_id,
		}

		if from_id != id.UNKNOWN {
			ref.From = from_id
		}

		if to_id != id.UNKNOWN {
			ref.To = to_id
		}

		if sent_id != id.UNKNOWN {
			ref.Sent = sent_id
		}

		if received_id != id.UNKNOWN {
			ref.Received = received_id
		}

		_, err := georeference.AssignFlightCoverReferences(ctx, opts, ref)

		if err != nil {
			return fmt.Errorf("Failed to assign flight cover references for depiction %d, %w", depiction_id, err)
		}
	}

	return nil
}
