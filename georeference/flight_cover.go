package georeference

import (
	"context"
	"fmt"
)

// sfomuseum:mail_from = ID
// sfomuseum:mail_to = ID
// sfomuseum:postmark_sent = ID
// sfomuseum:postmark_received = ID

type FlightCoverReferences struct {
	Id       int64 `json:"id"`
	From     int64 `json:"from"`
	To       int64 `json:"to"`
	Sent     int64 `json:"sent"`
	Received int64 `json:"received"`
}

func (flightcover_refs *FlightCoverReferences) References() []*Reference {

	refs := make([]*Reference, 0)

	if flightcover_refs.From != 0 {

		r := &Reference{
			Id:       flightcover_refs.From,
			Property: "sfomuseum:flightcover_address_from",
			AltLabel: "flightcover-address-from",
		}

		refs = append(refs, r)
	}

	if flightcover_refs.To != 0 {

		r := &Reference{
			Id:       flightcover_refs.To,
			Property: "sfomuseum:flightcover_address_to",
			AltLabel: "flightcover-address-to",
		}

		refs = append(refs, r)
	}

	if flightcover_refs.Sent != 0 {

		r := &Reference{
			Id:       flightcover_refs.Sent,
			Property: "sfomuseum:flightcover_postmark_sent",
			AltLabel: "flightcover-postmark-sent",
		}

		refs = append(refs, r)
	}

	if flightcover_refs.Received != 0 {

		r := &Reference{
			Id:       flightcover_refs.Received,
			Property: "sfomuseum:flightcover_postmark_received",
			AltLabel: "flightcover-postmark-received",
		}

		refs = append(refs, r)
	}

	return refs
}

func AssignFlightCoverReferences(ctx context.Context, assign_opts *AssignReferencesOptions, flightcover_refs *FlightCoverReferences) ([]byte, error) {

	wof_id := flightcover_refs.Id

	refs := flightcover_refs.References()

	if len(refs) == 0 {
		return nil, fmt.Errorf("Nothing to update")
	}

	return AssignReferences(ctx, assign_opts, wof_id, refs...)
}
