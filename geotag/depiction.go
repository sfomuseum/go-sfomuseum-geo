package geotag

import (
	"github.com/sfomuseum/go-geojson-geotag/v2"
)

// type Depiction is a struct definining properties for updating geotagging information in an depiction and its parent subject.
type Depiction struct {
	// The unique numeric identifier of the depiction being geotagged
	DepictionId int64 `json:"depiction_id"`
	// The GeoJSON Feature containing geotagging information
	Feature *geotag.GeotagFeature `json:"feature"`
}

