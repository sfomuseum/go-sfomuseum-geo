package geotag

import (
	geojson "github.com/sfomuseum/go-geojson-geotag"
)

// Depiction: object
// Subject: place depicted by object
// Parent: the parent (ID) of the subject

const DEFAULT_DEPICTION_READER_URI string = "github://sfomuseum-data/sfomuseum-data-media-collection?branch=%main"
const DEFAULT_DEPICTION_WRITER_URI string = "githubapi://sfomuseum-data/sfomuseum-data-media-collection?access_token=&branch=main&prefix=data/"

const DEFAULT_SUBJECT_READER_URI string = "github://sfomuseum-data/sfomuseum-data-collection?branch=%main"
const DEFAULT_SUBJECT_WRITER_URI string = "githubapi://sfomuseum-data/sfomuseum-data-collection?access_token=&branch=main&prefix=data/"

// This needs to be updated to pull from the SFOM findingaid because the parent might be the SFO campus
// or some other WOF feature off-campus. Was previously "github://sfomuseum-data/sfomuseum-data-architecture?branch=%main"

// DEFAULT_PARENT_READER_URI is the whosonfirst/go-reader.Reader URI used to retrieve parent records for geotagged features.
const DEFAULT_PARENT_READER_URI string = "https://static.sfomuseum.org/geojson"

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
