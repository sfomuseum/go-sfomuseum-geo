package geotag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
	"github.com/sfomuseum/go-sfomuseum-geo"
	"github.com/sfomuseum/go-sfomuseum-geo/alt"
	"github.com/sfomuseum/go-sfomuseum-geo/geometry"
	"github.com/sfomuseum/go-sfomuseum-geo/github"
	geo_writers "github.com/sfomuseum/go-sfomuseum-geo/writers"
	"github.com/tidwall/gjson"
	"github.com/whosonfirst/go-ioutil"
	"github.com/whosonfirst/go-reader/v2"
	"github.com/whosonfirst/go-whosonfirst-export/v3"
	"github.com/whosonfirst/go-whosonfirst-feature/properties"
	wof "github.com/whosonfirst/go-whosonfirst-id"
	wof_reader "github.com/whosonfirst/go-whosonfirst-reader/v2"
	"github.com/whosonfirst/go-whosonfirst-uri"
	wof_writer "github.com/whosonfirst/go-whosonfirst-writer/v3"
)

// AddGeotagDepictionOptions defines a struct for reading/writing options when updating geotagging information in depictions.
type AddGeotagDepictionOptions struct {
	// A valid whosonfirst/go-reader.Reader instance for reading depiction features.
	DepictionReader reader.Reader
	// A valid whosonfirst/go-writer.Writer URI for writing depiction features.
	DepictionWriterURI string
	// A valid whosonfirst/go-reader.Reader instance for reading subject features.
	SubjectReader reader.Reader
	// A valid whosonfirst/go-writer.Writer URI for writing subject features.
	SubjectWriterURI string
	// A valid whosonfirst/go-reader.Reader instance for reading "parent" features. This includes general Who's On First IDs.
	// This is the equivalent to ../georeference.AssignReferenceOptions.WhosOnFirstReader and should be reconciled one way or the other.
	WhosOnFirstReader reader.Reader
	// The name of the person (or process) updating a depiction.
	Author string
}

// AddGeotagDepiction will update the geometries and relevant properties for SFOM/WOF records 'depiction_id' and 'subject_id' using
// data defined in 'geotag_f' and 'parent_id'.
//
// 'geotag_f' is a GeoJSON Feature produced by the https://github.com/sfomuseum/Leaflet.GeotagPhoto package. There is also
// a https://github.com/sfomuseum/go-geojson-geotag Go package but we are not using it at this time.
//
// Here's how things work:
//
// First we retrieve the subject record associated with 'depiction_id' and update its geometry; This is assumed to be the
// value of the `wof:parent_id` property in the WOF/GeoJSON record for 'depiction_id'. The rules for updating subject geometries are:
//
//   - If the geometry is a `Point` we assume that the subject (and its depictions) have not been geotagged and assign the focal
//     point (centroid) of the 'geotag_f' feature as the first element of a new `MultiPoint` geometry.
//   - If the geometry is a `MultiPoint` we assume that the subject and at least one of its depictions have been geotagged. The
//     will assign the focalpoint (centroid) of the 'geotag_f' feature to the existing `MultiPoint` geometry assuming it is
//     not already present.
//   - Other geometry types will trigger an error, at this time.
//
// If 'parent_id' is not `-1` the code retrieve the record associated with that ID and updates the `wof:parent_id` and `wof:hierarchy`
// properties (in the subject record) with the `wof:id` and `wof:hierarchy` properties, respectively, in the parent record.
//
// After exporting and writing the subject record the depiction record associated with 'depiction_id' is retrieved.
//   - Its geometry is assigned the focal point (centroid) of the 'geotag_f' feature.
//   - Its `wof:hierarchy` property is updated with the corresponding value in the subject record.
//   - Other relevant properties are updated notably the `src:geom_alt` property which references an alternate geometry for the depiction
//     to be created or updated.
//
// After exporting and writing the depiction record a new alternate geometry (`geotag-fov`) is created for the depiction.
// - Its geometry is assigned the field of view (line string) of the 'geotag_f' feature.
//
// Finally the alternate geometry is exported and written (to `opts.DepictionWriter`).
func AddGeotagDepiction(ctx context.Context, opts *AddGeotagDepictionOptions, update *Depiction) ([]byte, error) {

	depiction_id := update.DepictionId
	geotag_f := update.Feature

	geotag_props := geotag_f.Properties
	camera_parent_id := geotag_props.Camera.ParentId
	target_parent_id := geotag_props.Target.ParentId

	logger := slog.Default()
	logger = logger.With("action", "add geotag")
	logger = logger.With("depiction id", depiction_id)
	logger = logger.With("camera", camera_parent_id)
	logger = logger.With("target", target_parent_id)

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

	logger.Debug("Load depiction")

	depiction_body, err := wof_reader.LoadBytes(ctx, opts.DepictionReader, depiction_id)

	if err != nil {
		return nil, fmt.Errorf("Failed to load depiction record %d, %w", depiction_id, err)
	}

	parent_rsp := gjson.GetBytes(depiction_body, "properties.wof:parent_id")

	if !parent_rsp.Exists() {
		return nil, fmt.Errorf("Failed to determine wof:parent_id for depiction")
	}

	subject_id := parent_rsp.Int()

	// subject_body is the feature that parents the depiction (for example an object that has one or more depictions)

	subject_body, err := wof_reader.LoadBytes(ctx, opts.SubjectReader, subject_id)

	if err != nil {
		return nil, fmt.Errorf("Failed to load subject record %d, %w", subject_id, err)
	}

	// *_parent_f are the Who's On First place features that parent/contains a geotagging geometry
	// camera is point at which a depiction was created; target is what the depiction is pointing at

	var camera_parent_f []byte
	var target_parent_f []byte

	if camera_parent_id != -1 {

		f, err := wof_reader.LoadBytes(ctx, opts.WhosOnFirstReader, camera_parent_id)

		if err != nil {
			return nil, fmt.Errorf("Failed to load parent record for camera %d, %w", camera_parent_id, err)
		}

		camera_parent_f = f
	}

	if target_parent_id != -1 {

		f, err := wof_reader.LoadBytes(ctx, opts.WhosOnFirstReader, target_parent_id)

		if err != nil {
			return nil, fmt.Errorf("Failed to load parent record for target %d, %w", target_parent_id, err)
		}

		target_parent_f = f
	}

	// Update the subject

	// FIX ME: account for multitple camera/target properties (as in multiple images (depictions)
	// with different geotagging properties

	subject_updates := map[string]interface{}{
		"properties.src:geom": "sfomuseum#geotagged",
	}

	depiction_wof_belongsto := []int64{
		camera_parent_id,
		target_parent_id,
	}

	subject_wof_belongsto := []int64{
		camera_parent_id,
		target_parent_id,
	}

	subject_wof_camera := []int64{
		camera_parent_id,
	}

	subject_wof_target := []int64{
		target_parent_id,
	}

	// Update the (subject) geotag:depictions array to include depiction_id

	subject_depictions := []int64{
		depiction_id,
	}

	depictions_rsp := gjson.GetBytes(subject_body, "properties.geotag:depictions")

	for _, r := range depictions_rsp.Array() {
		id := r.Int()

		if !slices.Contains(subject_depictions, id) {
			subject_depictions = append(subject_depictions, id)
		}
	}

	subject_updates["properties.geotag:depictions"] = subject_depictions

	// Update the subject geometry

	pov, err := geotag_f.PointOfView()

	if err != nil {
		return nil, fmt.Errorf("Unable to derive camera point of view, %w", err)
	}

	// TO DO: REMOVE mz:is_approximate if present

	target, err := geotag_f.Target()

	if err != nil {
		return nil, fmt.Errorf("Unable to derive camera target, %w", err)
	}

	camera_coords := pov.Coordinates
	target_coords := target.Coordinates

	camera_coord := []float64{
		camera_coords[0],
		camera_coords[1],
	}

	// START OF derive geometry for subject. This is derived from the following:
	// The unique set of "camera" coordinates for each of the geotagged depictions for the subject
	// The unique set of principal centroid for each of the Who's On First IDs for georeferenced depictions of the subject
	// It would be nice to believe this code could be abstracted out and shared
	// with equivalent requirements in ../georeference. It probably can but right
	// now that feels a bit too much like yak-shaving.

	depictions_coords := make([][]float64, 0)
	depictions_coords = append(depictions_coords, camera_coord)

	// Fetch other depictions for a given subject; we do this in order to generate
	// a MultiPoint geometry of all the depictions for a subject

	for _, other_id := range subject_depictions {

		// Skip current depiction as its been added above

		if other_id == depiction_id {
			continue
		}

		other_f, err := wof_reader.LoadBytes(ctx, opts.DepictionReader, other_id)

		if err != nil {
			return nil, fmt.Errorf("Failed to load depiction record %d, %w", other_id, err)
		}

		// TBD: Does this really need to throw a "fatal" error? Is it okay to skip records
		// with missing properties?

		lat_rsp := gjson.GetBytes(other_f, "properties.geotag:camera_latitude")
		lon_rsp := gjson.GetBytes(other_f, "properties.geotag:camera_longitude")

		if !lat_rsp.Exists() {
			return nil, fmt.Errorf("Depiction record %d is missing geotag:camera_latitude property", other_id)
		}

		if !lon_rsp.Exists() {
			return nil, fmt.Errorf("Depiction record %d is missing geotag:camera_longitude property", other_id)
		}

		other_coord := []float64{
			lon_rsp.Float(),
			lat_rsp.Float(),
		}

		// []float64 does not satisfy comparable so live fast...
		// if !slices.Contains(depictions_coords, other_coord){
		depictions_coords = append(depictions_coords, other_coord)

		belongsto_rsp := gjson.GetBytes(other_f, "properties.geotag:whosonfirst_belongsto")

		for _, i := range belongsto_rsp.Array() {

			id := i.Int()

			if !slices.Contains(subject_wof_belongsto, id) {
				subject_wof_belongsto = append(subject_wof_belongsto, id)
			}
		}

		camera_rsp := gjson.GetBytes(other_f, "properties.geotag:whosonfirst_camera")
		target_rsp := gjson.GetBytes(other_f, "properties.geotag:whosonfirst_target")

		if camera_rsp.Exists() {

			camera_id := camera_rsp.Int()

			if camera_id > -1 && !slices.Contains(subject_wof_camera, camera_id) {
				subject_wof_camera = append(subject_wof_camera, camera_id)
			}
		}

		if target_rsp.Exists() {

			target_id := target_rsp.Int()

			if target_id > -1 && !slices.Contains(subject_wof_target, target_id) {
				subject_wof_camera = append(subject_wof_target, target_id)
			}
		}
	}

	subject_geom_ids := make([]int64, 0)

	georefs_path := fmt.Sprintf("properties.%s", geo.RESERVED_GEOREFERENCE_DEPICTIONS)
	georefs_rsp := gjson.GetBytes(subject_body, georefs_path)

	for _, r := range georefs_rsp.Array() {

		r_id := r.Int()

		if !slices.Contains(subject_geom_ids, r_id) {
			subject_geom_ids = append(subject_geom_ids, r_id)
		}
	}

	// derive subject_geom_ids from georeferernces...

	if len(subject_geom_ids) > 0 {

		subject_orb_geom, err := geometry.DeriveMultiPointFromIds(ctx, opts.WhosOnFirstReader, subject_geom_ids...)

		if err != nil {
			return nil, fmt.Errorf("Failed to derive multipoint geometry for subject, %w", err)
		}

		// Now append the geometry for the depiction
		// This extra step is necessary to satisfy Golang generics wah-wah for slices.Contains below
		subject_points := subject_orb_geom.(orb.MultiPoint)

		for _, coord := range depictions_coords {

			pt := orb.Point(coord)

			if !slices.Contains(subject_points, pt) {
				subject_points = append(subject_points, pt)
			}
		}

		new_mp := orb.MultiPoint(subject_points)
		new_geom := geojson.NewGeometry(new_mp)

		subject_updates["geometry"] = new_geom
	} else {
		// No other geometries so just append the geometry for the depiction
		// Which might be the "default" geometry if there are no pointers
		subject_updates["geometry.type"] = "MultiPoint"
		subject_updates["geometry.coordinates"] = depictions_coords
	}

	// END OF...

	subject_updates["properties.geotag:whosonfirst_belongsto"] = subject_wof_belongsto

	if len(subject_wof_camera) > 0 {
		subject_updates["properties.geotag:whosonfirst_camera"] = subject_wof_camera
	}

	if len(subject_wof_target) > 0 {
		subject_updates["properties.geotag:whosonfirst_target"] = subject_wof_target
	}

	// Update the parent ID and hierarchy for the subject

	if camera_parent_f != nil {

		parent_hierarchies := properties.Hierarchies(camera_parent_f)

		for _, parent_h := range parent_hierarchies {

			for _, h_id := range parent_h {

				if h_id == wof.EARTH {
					continue
				}

				if slices.Contains(depiction_wof_belongsto, h_id) {
					continue
				}

				depiction_wof_belongsto = append(depiction_wof_belongsto, h_id)
			}
		}

		to_copy := []string{
			"properties.iso:country",
			"properties.wof:country",
		}

		for _, path := range to_copy {

			rsp := gjson.GetBytes(camera_parent_f, path)

			if rsp.Exists() {
				subject_updates[path] = rsp.Value()
			}
		}
	}

	if target_parent_f != nil {

		parent_hierarchies := properties.Hierarchies(target_parent_f)

		for _, parent_h := range parent_hierarchies {

			for _, h_id := range parent_h {
				if h_id == wof.EARTH {
					continue
				}

				if slices.Contains(depiction_wof_belongsto, h_id) {
					continue
				}

				depiction_wof_belongsto = append(depiction_wof_belongsto, h_id)
			}
		}

		to_copy := []string{
			"properties.iso:country",
			"properties.wof:country",
		}

		for _, path := range to_copy {

			rsp := gjson.GetBytes(target_parent_f, path)

			if rsp.Exists() {
				subject_updates[path] = rsp.Value()
			}
		}
	}

	subject_changed, subject_body, err := export.AssignPropertiesIfChanged(ctx, subject_body, subject_updates)

	if err != nil {
		return nil, fmt.Errorf("Failed to update subject record %d, %w", subject_id, err)
	}

	if subject_changed {

		lastmod_key := fmt.Sprintf("properties.%s", geo.RESERVED_GEOTAG_LASTMODIFIED)
		lastmod := time.Now()

		lastmod_updates := map[string]any{
			lastmod_key: lastmod.Unix(),
		}

		subject_body, err := export.AssignProperties(ctx, subject_body, lastmod_updates)

		if err != nil {
			logger.Error("Failed to assign last mod properties for subject record", "error", err)
			return nil, fmt.Errorf("Failed to assign last mod properties for subject record, %w", err)
		}

		_, err = wof_writer.WriteBytes(ctx, writers.SubjectMultiWriter, subject_body)

		if err != nil {
			return nil, fmt.Errorf("Failed to write subject record %d, %w", subject_id, err)
		}
	}

	// Update the depiction

	depiction_updates := map[string]interface{}{
		"geometry":                                pov,
		"properties.src:geom":                     "sfomuseum",
		"properties.geotag:angle":                 geotag_f.Properties.Angle,
		"properties.geotag:bearing":               geotag_f.Properties.Bearing,
		"properties.geotag:distance":              geotag_f.Properties.Distance,
		"properties.geotag:camera_longitude":      camera_coords[0],
		"properties.geotag:camera_latitude":       camera_coords[1],
		"properties.geotag:target_longitude":      target_coords[0],
		"properties.geotag:target_latitude":       target_coords[1],
		"properties.geotag:whosonfirst_camera":    camera_parent_id,
		"properties.geotag:whosonfirst_target":    target_parent_id,
		"properties.geotag:whosonfirst_belongsto": depiction_wof_belongsto,
		"properties.geotag:subject":               subject_id,
	}

	geom_alt := []string{
		GEOTAG_LABEL,
	}

	geom_alt_rsp := gjson.GetBytes(depiction_body, "properties.src:geom_alt")

	if geom_alt_rsp.Exists() {

		for _, r := range geom_alt_rsp.Array() {

			if r.String() == GEOTAG_LABEL {
				continue
			}

			geom_alt = append(geom_alt, r.String())
		}
	}

	depiction_updates["properties.src:geom_alt"] = geom_alt

	depiction_changed, depiction_body, err := export.AssignPropertiesIfChanged(ctx, depiction_body, depiction_updates)

	if err != nil {
		return nil, fmt.Errorf("Failed to update depiction record %d, %w", depiction_id, err)
	}

	if depiction_changed {

		lastmod_key := fmt.Sprintf("properties.%s", geo.RESERVED_GEOTAG_LASTMODIFIED)
		lastmod := time.Now()

		lastmod_updates := map[string]any{
			lastmod_key: lastmod.Unix(),
		}

		depiction_body, err := export.AssignProperties(ctx, depiction_body, lastmod_updates)

		if err != nil {
			logger.Error("Failed to assign last mod properties for depiction record", "error", err)
			return nil, fmt.Errorf("Failed to assign last mod properties for depiction record, %w", err)
		}

		_, err = wof_writer.WriteBytes(ctx, writers.DepictionMultiWriter, depiction_body)

		if err != nil {
			return nil, fmt.Errorf("Failed to write depiction record %d, %w", depiction_id, err)
		}
	}

	// Update the alt depiction geometry

	repo_rsp := gjson.GetBytes(depiction_body, "properties.wof:repo")

	alt_props := map[string]interface{}{
		"wof:id":        depiction_id,
		"wof:repo":      repo_rsp.String(),
		"src:alt_label": GEOTAG_LABEL,
		"src:geom":      "sfomuseum",
	}

	fov_geom, err := geotag_f.FieldOfView()

	if err != nil {
		return nil, fmt.Errorf("Failed to derive field of view geometry, %w", err)
	}

	enc_fov, err := json.Marshal(fov_geom)

	if err != nil {
		return nil, fmt.Errorf("Failed to marshal field of view geometry, %w", err)
	}

	geojson_geom, err := geojson.UnmarshalGeometry(enc_fov)

	if err != nil {
		return nil, fmt.Errorf("Failed to unmarshal field of view geometry, %w", err)
	}

	alt_feature := &alt.WhosOnFirstAltFeature{
		Type:       "Feature",
		Id:         depiction_id,
		Properties: alt_props,
		Geometry:   geojson_geom,
	}

	alt_body, err := alt.FormatAltFeature(alt_feature)

	if err != nil {
		return nil, fmt.Errorf("Failed to format alt feature, %w", err)
	}

	alt_uri_geom := &uri.AltGeom{
		Source: GEOTAG_LABEL,
	}

	alt_uri_args := &uri.URIArgs{
		IsAlternate: true,
		AltGeom:     alt_uri_geom,
	}

	alt_uri, err := uri.Id2RelPath(depiction_id, alt_uri_args)

	if err != nil {
		return nil, fmt.Errorf("Failed to derive rel path for alt file, %w", err)
	}

	alt_br := bytes.NewReader(alt_body)
	alt_fh, err := ioutil.NewReadSeekCloser(alt_br)

	if err != nil {
		return nil, fmt.Errorf("Failed to create new ReadSeekCloser, %w", err)
	}

	// Note: We are writing to the DepictionWriter and not the DepictionMultiWriter since this
	// is the alt file

	_, err = writers.DepictionWriter.Write(ctx, alt_uri, alt_fh)

	if err != nil {
		return nil, fmt.Errorf("Failed to write alt file %s, %w", alt_uri, err)
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

	//

	fc, err := writers.AsFeatureCollection()

	if err != nil {
		return nil, fmt.Errorf("Failed to derive feature collection, %w", err)
	}

	new_alt_f, err := geojson.UnmarshalFeature(alt_body)

	if err != nil {
		return nil, fmt.Errorf("Failed to unmarshal feature from alt body, %w", err)
	}

	fc.Append(new_alt_f)

	fc_body, err := fc.MarshalJSON()

	if err != nil {
		return nil, fmt.Errorf("Failed to marshal feature collection, %w", err)
	}

	return fc_body, nil
}
