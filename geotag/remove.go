package geotag

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/paulmach/orb/geojson"
	"github.com/sfomuseum/go-sfomuseum-geo"
	"github.com/sfomuseum/go-sfomuseum-geo/github"
	geo_writers "github.com/sfomuseum/go-sfomuseum-geo/writers"
	"github.com/tidwall/gjson"
	"github.com/whosonfirst/go-reader/v2"
	"github.com/whosonfirst/go-whosonfirst-export/v3"
	wof_reader "github.com/whosonfirst/go-whosonfirst-reader/v2"
	"github.com/whosonfirst/go-whosonfirst-uri"
	wof_writer "github.com/whosonfirst/go-whosonfirst-writer/v3"
)

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
	// A default or "fallback" geometry to use for depictions and subjects if no other geometry can be derived
	DefaultGeometry *geojson.Geometry
	// A valid whosonfirst/go-reader.Reader instance for reading "parent" features. This includes general Who's On First IDs.
	WhosOnFirstReader reader.Reader
}

func RemoveGeotagDepiction(ctx context.Context, opts *RemoveGeotagDepictionOptions, update *Depiction) ([]byte, error) {

	depiction_id := update.DepictionId

	logger := slog.Default()
	logger = logger.With("action", "remove geotag")
	logger = logger.With("depiction id", depiction_id)

	logger.Debug("Set up writers")

	github_opts := &github.UpdateWriterURIOptions{
		Author:        opts.Author,
		WhosOnFirstId: depiction_id,
		Action:        github.GeotagAction,
	}

	writers_opts := &geo_writers.CreateWritersOptions{
		SubjectWriterURI:    opts.SubjectWriterURI,
		DepictionWriterURI:  opts.DepictionWriterURI,
		GithubWriterOptions: github_opts,
	}

	// See notes in writers/writers.go for why this returns both "Writer" and "MultiWriter" instances (for now)
	writers, err := geo_writers.CreateWriters(ctx, writers_opts)

	if err != nil {
		logger.Error("Failed to create writers", "error", err)
		return nil, fmt.Errorf("Failed to create geotag writers, %w", err)
	}

	// Load depiction

	logger.Debug("Load depiction")

	depiction_body, err := wof_reader.LoadBytes(ctx, opts.DepictionReader, depiction_id)

	if err != nil {
		logger.Error("Failed to load depiction", "error", err)
		return nil, fmt.Errorf("Failed to load depiction record, %w", err)
	}

	// Derive subject (for depiction)

	logger.Debug("Derive subject")

	subject_prop := fmt.Sprintf("properties.%s", geo.RESERVED_GEOTAG_SUBJECT)

	subject_rsp := gjson.GetBytes(depiction_body, subject_prop)

	if !subject_rsp.Exists() {
		return nil, fmt.Errorf("Depiction is missing %s property", subject_prop)
	}

	subject_id := subject_rsp.Int()
	logger = logger.With("subject id", subject_id)

	// Update depiction

	logger.Debug("Update depiction")

	depiction_update := make(map[string]any)
	depiction_remove := make([]string, 0)

	depiction_props := gjson.GetBytes(depiction_body, "properties")

	for k, _ := range depiction_props.Map() {

		if strings.HasPrefix(k, "geotag:") {
			path := fmt.Sprintf("properties.%s", k)
			logger.Debug("Remove depiction property", "path", path)
			depiction_remove = append(depiction_remove, path)
		}
	}

	alt_rsp := gjson.GetBytes(depiction_body, "properties.src:geom_alt")

	if !alt_rsp.Exists() {
		logger.Warn("Depiction is missing geom_alt property")
	} else {

		logger.Debug("Update alt geom(s)")

		// Derive new geom_alt array

		fov_label := "geotag-fov"
		alt_geoms := make([]string, 0)

		for _, r := range alt_rsp.Array() {
			label := r.String()

			if label != fov_label {
				logger.Debug("Append alt geom", "label", label)
				alt_geoms = append(alt_geoms, label)
			}
		}

		depiction_update["properties.src:geom_alt"] = alt_geoms

		// Deprecate alt geom
		// START OF put me in a function somewhere...

		logger.Debug("Deprecated alt geom", "label", fov_label)

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

	logger.Debug("Update depiction geometry")

	depiction_geom_opts := &DeriveGeometryForDepictionOptions{
		WhosOnFirstReader: opts.WhosOnFirstReader,
	}

	depiction_geom, err := DeriveGeometryForDepiction(ctx, depiction_geom_opts, depiction_body)

	if err != nil {
		return nil, fmt.Errorf("Failed to derive geometry for depiction, %w", err)
	}

	if depiction_geom == nil {

		if opts.DefaultGeometry == nil {
			return nil, fmt.Errorf("Default geometry is not defined")
		}

		depiction_geom = opts.DefaultGeometry
	}

	depiction_update["geometry"] = depiction_geom

	// Apply depiction changes

	logger.Debug("Apply changes for depiction")

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

	logger.Debug("Load subject")

	subject_body, err := wof_reader.LoadBytes(ctx, opts.SubjectReader, subject_id)

	if err != nil {
		return nil, fmt.Errorf("Failed to load subject record, %w", err)
	}

	// Update subject

	logger.Debug("Update subject")

	subject_update := make(map[string]any)
	subject_remove := make([]string, 0)

	subject_props := gjson.GetBytes(subject_body, "properties")

	for k, _ := range subject_props.Map() {

		if strings.HasPrefix(k, "geotag:") {
			path := fmt.Sprintf("properties.%s", k)
			logger.Debug("Remove subject property", "path", path)
			subject_remove = append(subject_remove, path)
		}
	}

	// Update subject geometry

	logger.Debug("Update subject geometry")

	subject_geom_opts := &DeriveGeometryForSubjectOptions{
		WhosOnFirstReader: opts.WhosOnFirstReader,
		DepictionReader:   opts.DepictionReader,
	}

	subject_geom, err := DeriveGeometryForSubject(ctx, subject_geom_opts, subject_body)

	if err != nil {
		return nil, fmt.Errorf("Failed to derive geometry for subject, %w", err)
	}

	if subject_geom == nil {

		if opts.DefaultGeometry == nil {
			return nil, fmt.Errorf("Default geometry is not defined")
		}

		subject_geom = opts.DefaultGeometry
	}

	subject_update["geometry"] = subject_geom

	// Apply changes for subject

	logger.Debug("Apply changes for subject")

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

	logger.Debug("Write changes for depiction")

	_, err = wof_writer.WriteBytes(ctx, writers.DepictionMultiWriter, depiction_body)

	if err != nil {
		return nil, fmt.Errorf("Failed to write changes for depiction, %w", err)
	}

	logger.Debug("Write changes for subject")

	_, err = wof_writer.WriteBytes(ctx, writers.SubjectMultiWriter, subject_body)

	if err != nil {
		return nil, fmt.Errorf("Failed to write changes for subject, %w", err)
	}

	// Close the depiction and subject writers - this is a no-op for many writer but
	// required for things like the githubapi-tree:// and githubapi-pr:// writers.

	err = writers.DepictionMultiWriter.Close(ctx)

	if err != nil {
		logger.Error("Failed to close depiction writer", "error", err)
		return nil, fmt.Errorf("Failed to close depiction writer, %w", err)
	}

	err = writers.SubjectMultiWriter.Close(ctx)

	if err != nil {
		logger.Error("Failed to close subject writer", "error", err)
		return nil, fmt.Errorf("Failed to close subject writer, %w", err)
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
