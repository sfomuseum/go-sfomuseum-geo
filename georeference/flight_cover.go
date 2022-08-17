package georeference

import (
	"context"
	"fmt"
)

const PROPERTY_FLIGHTCOVER_FROM string = "millsfield:flightcover_address_from"
const ALTLABEL_FLIGHTCOVER_FROM string = "flightcover-address-from"

const PROPERTY_FLIGHTCOVER_TO string = "millsfield:flightcover_address_to"
const ALTLABEL_FLIGHTCOVER_TO string = "flightcover-address-to"

const PROPERTY_FLIGHTCOVER_SENT string = "millsfield:flightcover_address_sent"
const ALTLABEL_FLIGHTCOVER_SENT string = "flightcover-address-sent"

const PROPERTY_FLIGHTCOVER_RECEIVED string = "millsfield:flightcover_address_received"
const ALTLABEL_FLIGHTCOVER_RECEIVED string = "flightcover-address-received"

type FlightCoverReferences struct {
	Id       int64   `json:"id"`
	From     []int64 `json:"from"`
	To       []int64 `json:"to"`
	Sent     []int64 `json:"sent"`
	Received []int64 `json:"received"`
}

func (flightcover_refs *FlightCoverReferences) References() []*Reference {

	refs := make([]*Reference, 0)

	if len(flightcover_refs.From) > 0 {

		r := &Reference{
			Ids:      flightcover_refs.From,
			Property: PROPERTY_FLIGHTCOVER_FROM,
			AltLabel: ALTLABEL_FLIGHTCOVER_FROM,
		}

		refs = append(refs, r)
	}

	if len(flightcover_refs.To) > 0 {

		r := &Reference{
			Ids:      flightcover_refs.To,
			Property: PROPERTY_FLIGHTCOVER_TO,
			AltLabel: ALTLABEL_FLIGHTCOVER_TO,
		}

		refs = append(refs, r)
	}

	if len(flightcover_refs.Sent) > 0 {

		r := &Reference{
			Ids:      flightcover_refs.Sent,
			Property: PROPERTY_FLIGHTCOVER_SENT,
			AltLabel: ALTLABEL_FLIGHTCOVER_SENT,
		}

		refs = append(refs, r)
	}

	if len(flightcover_refs.Received) > 0 {

		r := &Reference{
			Ids:      flightcover_refs.Received,
			Property: PROPERTY_FLIGHTCOVER_RECEIVED,
			AltLabel: ALTLABEL_FLIGHTCOVER_RECEIVED,
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
