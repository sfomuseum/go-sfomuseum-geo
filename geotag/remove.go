package geotag

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/sfomuseum/go-sfomuseum-geo"
	"github.com/paulmach/orb/geojson"	
	"github.com/tidwall/gjson"
	"github.com/whosonfirst/go-reader/v2"
	"github.com/whosonfirst/go-whosonfirst-export/v3"
	wof_reader "github.com/whosonfirst/go-whosonfirst-reader/v2"
	"github.com/whosonfirst/go-whosonfirst-uri"
	wof_writer "github.com/whosonfirst/go-whosonfirst-writer/v3"
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
	// A valid whosonfirst/go-writer.Writer URI for writing depiction features.
	DepictionWriterURI string
	// A valid whosonfirst/go-reader.Reader instance for reading subject features.
	SubjectReader reader.Reader
	// A valid whosonfirst/go-writer.Writer URI for writing subject features.
	SubjectWriterURI string
	// The name of the person (or process) updating a depiction.
	Author string
	// A default or "fallback" geometry to use for depictions and subjects if no other
	// geometry can be derived
	DefaultGeometry *geojson.Geometry

	// TBD: are these necessary...

	// A valid whosonfirst/go-reader.Reader instance for reading "parent" features. This includes general Who's On First IDs.
	// This is the equivalent to ../georeference.AssignReferenceOptions.WhosOnFirstReader and should be reconciled one way or the other.
	// ParentReader reader.Reader
	// A valid whosonfirst/go-writer.Writer instance for writing depiction features.
	// DepictionWriter    writer.Writer
	// A valid whosonfirst/go-writer.Writer instance for writing subject features.
	// SubjectWriter    writer.Writer

}

func RemoveGeotagDepiction(ctx context.Context, opts *RemoveGeotagDepictionOptions, update *Depiction) ([]byte, error) {

	depiction_id := update.DepictionId

	logger := slog.Default()
	logger = logger.With("action", "remove geotag")
	logger = logger.With("depiction id", depiction_id)

	writer_opts := &CreateGeotagWritersOptions{
		DepictionId:        depiction_id,
		Author:             opts.Author,
		SubjectWriterURI:   opts.SubjectWriterURI,
		DepictionWriterURI: opts.DepictionWriterURI,
	}

	writers, err := CreateGeotagWriters(ctx, writer_opts)

	if err != nil {
		logger.Error("Failed to create writers", "error", err)
		return nil, fmt.Errorf("Failed to create geotag writers, %w", err)
	}

	// Load depiction

	depiction_body, err := wof_reader.LoadBytes(ctx, opts.DepictionReader, depiction_id)

	if err != nil {
		logger.Error("Failed to load depiction", "error", err)
		return nil, fmt.Errorf("Failed to load depiction record, %w", err)
	}

	// Derive subject (for depiction)

	subject_prop := fmt.Sprintf("properties.%s", geo.RESERVED_GEOTAG_SUBJECT)

	subject_rsp := gjson.GetBytes(depiction_body, subject_prop)

	if !subject_rsp.Exists() {
		return nil, fmt.Errorf("Depiction is missing %s property", subject_prop)
	}

	subject_id := subject_rsp.Int()
	logger = logger.With("subject id", subject_id)

	// Update depiction

	depiction_update := make(map[string]any)
	depiction_remove := make([]string, 0)

	depiction_props := gjson.GetBytes(depiction_body, "properties")

	for k, _ := range depiction_props.Map() {

		if strings.HasPrefix(k, "geotag:") {
			path := fmt.Sprintf("properties.%s", k)
			depiction_remove = append(depiction_remove, path)
		}
	}

	alt_rsp := gjson.GetBytes(depiction_body, "properties.geom_alt")

	if !alt_rsp.Exists() {
		logger.Warn("Depiction is missing geom_alt property")
	} else {

		// Derive new geom_alt array

		fov_label := "geotag-fov"
		alt_geoms := make([]string, 0)

		for _, r := range alt_rsp.Array() {
			geom := r.String()

			if geom != fov_label {
				alt_geoms = append(alt_geoms, geom)
			}
		}

		depiction_update["properties.geom_alt"] = alt_geoms

		// Deprecate alt geom
		// START OF put me in a function somewhere...

		alt_args, err := uri.NewAlternateURIArgsFromAltLabel(fov_label)

		if err != nil {
			return nil, fmt.Errorf("Failed to derive URI args from geotag-fov alt label, %w", err)
		}

		alt_uri, err := uri.Id2RelPath(depiction_id, alt_args)

		if err != nil {
			return nil, fmt.Errorf("Failed to derive alt URI for depiction, %w", err)
		}

		// START OF whosonfirst/go-whosonfirst-reader doesn't know how to work with alt files

		alt_r, err := opts.DepictionReader.Read(ctx, alt_uri)

		if err != nil {
			return nil, fmt.Errorf("Failed to open alt URI for reading, %w", err)
		}

		defer alt_r.Close()

		alt_body, err := io.ReadAll(alt_r)

		if err != nil {
			return nil, fmt.Errorf("Failed to load alt file, %w", err)
		}

		// END OF whosonfirst/go-whosonfirst-reader doesn't know how to work with alt files

		now := time.Now()

		alt_updates := map[string]any{
			"properties.edtf:deprecated": now.Format("2006-01-02"),
		}

		new_alt_body, err := export.AssignProperties(ctx, alt_body, alt_updates)

		if err != nil {
			return nil, fmt.Errorf("Failed to assign properties to alt depiction, %w", err)
		}

		_, new_alt_body, err = export.Export(ctx, new_alt_body)

		if err != nil {
			return nil, fmt.Errorf("Failed to export alt depiction, %w", err)
		}

		_, err = wof_writer.WriteBytes(ctx, writers.DepictionWriter, new_alt_body)

		if err != nil {
			return nil, fmt.Errorf("Failed to write changes for alt depiction, %w", err)
		}

		// END OF put me in a function somewhere...
	}

	// Update depiction geometry

	// Rebuild geometry from any existing georef: properties
	// Otherwise use opts.DefaultGeomtry

	depiction_update["geometry"] = opts.DefaultGeometry

	// Apply depiction changes

	depiction_body, err = export.RemoveProperties(ctx, depiction_body, depiction_remove)

	if err != nil {
		return nil, fmt.Errorf("Failed to remove properties from depiction, %w", err)
	}

	depiction_body, err = export.AssignProperties(ctx, depiction_body, depiction_update)

	if err != nil {
		return nil, fmt.Errorf("Failed to update properties for depiction, %w", err)
	}

	_, depiction_body, err = export.Export(ctx, depiction_body)

	if err != nil {
		return nil, fmt.Errorf("Failed to export depiction, %w", err)
	}

	// Load subject

	subject_body, err := wof_reader.LoadBytes(ctx, opts.SubjectReader, subject_id)

	if err != nil {
		return nil, fmt.Errorf("Failed to load subject record, %w", err)
	}

	// Update subject

	subject_update := make(map[string]any)
	subject_remove := make([]string, 0)

	subject_props := gjson.GetBytes(subject_body, "properties")

	for k, _ := range subject_props.Map() {

		if strings.HasPrefix(k, "geotag:") {
			path := fmt.Sprintf("properties.%s", k)
			subject_remove = append(subject_remove, path)
		}
	}

	// Update subject geometry

	// Rebuild geometry from any existing georef: properties
	// Otherwise use opts.DefaultGeomtry

	subject_update["geometry"] = opts.DefaultGeometry

	// Apply changes for subject

	subject_body, err = export.RemoveProperties(ctx, subject_body, subject_remove)

	if err != nil {
		return nil, fmt.Errorf("Failed to remove properties from subject, %w", err)
	}

	subject_body, err = export.AssignProperties(ctx, subject_body, subject_update)

	if err != nil {
		return nil, fmt.Errorf("Failed to update properties for subject, %w", err)
	}

	_, subject_body, err = export.Export(ctx, subject_body)

	if err != nil {
		return nil, fmt.Errorf("Failed to export subject, %w", err)
	}

	// Write changes (for depiction and subject)

	_, err = wof_writer.WriteBytes(ctx, writers.DepictionMultiWriter, depiction_body)

	if err != nil {
		return nil, fmt.Errorf("Failed to write changes for depiction, %w", err)
	}

	_, err = wof_writer.WriteBytes(ctx, writers.SubjectMultiWriter, subject_body)

	if err != nil {
		return nil, fmt.Errorf("Failed to write changes for subject, %w", err)
	}

	// Return GeoJSON FeatureCollection with updated features (depiction, subject)

	fc, err := writers.AsFeatureCollection()

	if err != nil {
		return nil, fmt.Errorf("Failed to derive feature collection, %w", err)
	}

	fc_body, err := fc.MarshalJSON()

	if err != nil {
		return nil, fmt.Errorf("Failed to marshal feature collection, %w", err)
	}

	return fc_body, nil
}
