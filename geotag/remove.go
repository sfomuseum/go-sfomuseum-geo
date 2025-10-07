package geotag

import (
	"github.com/whosonfirst/go-reader/v2"
	"github.com/whosonfirst/go-writer/v3"
)

/*

- depiction (image)
  - remove geotag: properties
  - remove (deprecate) geotag-fov alt file; update src:geom_alt
  - reset geometry
    - remove centroid for geotag-fov alt file
    - if no remaining coordinates then reset to ... what?

- subject (object)
  - remove geotag: properties
  - reset geometry, tricky:
    - specifically only remove the coordinates associated with the geotag:depictions property
      - what to do about decimal differences...
    - if no remaining coordinates then reset to ... what?

*/

type RemoveGeotagDepictionOptions struct {
	// A valid whosonfirst/go-reader.Reader instance for reading depiction features.
	DepictionReader reader.Reader
	// A valid whosonfirst/go-writer.Writer instance for writing depiction features.
	DepictionWriter    writer.Writer
	DepictionWriterURI string
	// A valid whosonfirst/go-reader.Reader instance for reading subject features.
	SubjectReader reader.Reader
	// A valid whosonfirst/go-writer.Writer instance for writing subject features.
	SubjectWriter    writer.Writer
	SubjectWriterURI string

	// TBD: are these necessary...

	// A valid whosonfirst/go-reader.Reader instance for reading "parent" features. This includes general Who's On First IDs.
	// This is the equivalent to ../georeference.AssignReferenceOptions.WhosOnFirstReader and should be reconciled one way or the other.
	// ParentReader reader.Reader
	// The name of the person (or process) updating a depiction.
	// Author string
}

// func RemoveGeotagDepiction(ctx context.Context, opts *RemoveGeotagDepictionOptions, update *Depiction) ([]byte, error) {
