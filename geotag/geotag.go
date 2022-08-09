package geotag

import (
	geojson "github.com/sfomuseum/go-geojson-geotag"
)

// Depiction: object
// Subject: place depicted by object
// Parent: the parent (ID) of the subject

// GEOTAG_NS is the prefix (namespace) to assign to geotagging specific properties.
const GEOTAG_NS string = "geotag"

// GEOTAG_SRC is the (whosonfirst/whosonfirst-sources) source label identifying the source of the geotagging geometry.
const GEOTAG_SRC string = "geotag"

// GEOTAG_LABEL is the Who's On First alternate geometry label for the "field of view" alternate geometry record
const GEOTAG_LABEL string = "geotag-fov" // field of view

// type Depiction is a struct definining properties for updating geotagging information in an depiction and its parent subject.
type Depiction struct {
	// The unique numeric identifier of the depiction being geotagged
	DepictionId int64 `json:"depiction_id"`
	// The unique numeric identifier of the Who's On First feature that parents the subject being geotagged
	ParentId int64 `json:"parent_id,omitempty"`
	// The GeoJSON Feature containing geotagging information
	Feature *geojson.GeotagFeature `json:"feature"`
}
