package flightcover

import (
	"context"
	"fmt"
	"github.com/sfomuseum/go-sfomuseum-geo/georeference"
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

		if len(from_id) > 0 {
			ref.From = from_id
		}

		if len(to_id) > 0 {
			ref.To = to_id
		}

		if len(sent_id) > 0 {
			ref.Sent = sent_id
		}

		if len(received_id) > 0 {
			ref.Received = received_id
		}

		_, err := georeference.AssignFlightCoverReferences(ctx, opts, ref)

		if err != nil {
			return fmt.Errorf("Failed to assign flight cover references for depiction %d, %w", depiction_id, err)
		}
	}

	return nil
}
