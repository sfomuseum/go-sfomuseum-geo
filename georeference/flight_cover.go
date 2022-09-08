package georeference

import (
	"context"
	"fmt"
)

const SUFFIX_FLIGHTCOVER_SRC_GEOM string = "flightcover"

const PROPERTY_FLIGHTCOVER_FROM string = "millsfield:flightcover_address_from"
const ALTLABEL_FLIGHTCOVER_FROM string = "flightcover-address-from"

const PROPERTY_FLIGHTCOVER_TO string = "millsfield:flightcover_address_to"
const ALTLABEL_FLIGHTCOVER_TO string = "flightcover-address-to"

const PROPERTY_FLIGHTCOVER_SENT string = "millsfield:flightcover_address_sent"
const ALTLABEL_FLIGHTCOVER_SENT string = "flightcover-address-sent"

const PROPERTY_FLIGHTCOVER_RECEIVED string = "millsfield:flightcover_address_received"
const ALTLABEL_FLIGHTCOVER_RECEIVED string = "flightcover-address-received"

type FlightCoverReferences struct {
	// Id is the Who's On First (sfomuseum-data) ID of the record to which georeferences are being applied.
	Id int64 `json:"id"`
	// From is a list of zero or more Who's On First IDs representing the places where a flight cover letter was sent from.
	From []int64 `json:"from"`
	// To is a list of zero or more Who's On First IDs representing the places where a flight cover letter was sent to.
	To []int64 `json:"to"`
	// Sent is a list of zero or more Who's On First IDs representing the places where a flight cover letter was postmarked as having been sent from.
	Sent []int64 `json:"sent"`
	// Received is a list of zero or more Who's On First IDs representing the places where a flight cover letter was postmarked as having been received at.
	Received []int64 `json:"received"`
}

// References translates 'flightcover_refs' into a list of `Reference` instances.
func (flightcover_refs *FlightCoverReferences) References() []*Reference {

	// Note that we are passing in all the flight covers because a zero-length
	// entry signals that it should be removed.

	refs := []*Reference{
		&Reference{
			Ids:      flightcover_refs.From,
			Property: PROPERTY_FLIGHTCOVER_FROM,
			AltLabel: ALTLABEL_FLIGHTCOVER_FROM,
		},
		&Reference{
			Ids:      flightcover_refs.To,
			Property: PROPERTY_FLIGHTCOVER_TO,
			AltLabel: ALTLABEL_FLIGHTCOVER_TO,
		},
		&Reference{
			Ids:      flightcover_refs.Sent,
			Property: PROPERTY_FLIGHTCOVER_SENT,
			AltLabel: ALTLABEL_FLIGHTCOVER_SENT,
		},
		&Reference{
			Ids:      flightcover_refs.Received,
			Property: PROPERTY_FLIGHTCOVER_RECEIVED,
			AltLabel: ALTLABEL_FLIGHTCOVER_RECEIVED,
		},
	}

	return refs
}

func AssignFlightCoverReferences(ctx context.Context, assign_opts *AssignReferencesOptions, flightcover_refs *FlightCoverReferences) ([]byte, error) {

	wof_id := flightcover_refs.Id

	refs := flightcover_refs.References()

	if len(refs) == 0 {
		return nil, fmt.Errorf("Nothing to update")
	}

	assign_opts.SourceGeomSuffix = SUFFIX_FLIGHTCOVER_SRC_GEOM

	return AssignReferences(ctx, assign_opts, wof_id, refs...)
}
