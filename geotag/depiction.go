package geotag

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/paulmach/orb/geojson"
	"github.com/sfomuseum/go-geojson-geotag/v2"
	"github.com/sfomuseum/go-sfomuseum-geo/alt"
	"github.com/sfomuseum/go-sfomuseum-geo/github"
	sfom_writer "github.com/sfomuseum/go-sfomuseum-writer/v3"
	"github.com/tidwall/gjson"
	"github.com/whosonfirst/go-ioutil"
	"github.com/whosonfirst/go-reader"
	"github.com/whosonfirst/go-whosonfirst-export/v2"
	"github.com/whosonfirst/go-whosonfirst-feature/properties"
	wof "github.com/whosonfirst/go-whosonfirst-id"
	wof_reader "github.com/whosonfirst/go-whosonfirst-reader"
	"github.com/whosonfirst/go-whosonfirst-uri"
	"github.com/whosonfirst/go-writer/v3"
	"sync"
)

// type Depiction is a struct definining properties for updating geotagging information in an depiction and its parent subject.
type Depiction struct {
	// The unique numeric identifier of the depiction being geotagged
	DepictionId int64 `json:"depiction_id"`
	// The unique numeric identifier of the Who's On First feature that parents the subject being geotagged
	// DEPRECATED - commented out to trigger errors; to be removed
	// ParentId int64 `json:"parent_id,omitempty"`
	// The GeoJSON Feature containing geotagging information
	Feature *geotag.GeotagFeature `json:"feature"`
}

// UpdateDepictionOptions defines a struct for reading/writing options when updating geotagging information in depictions.
type UpdateDepictionOptions struct {
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
	// A valid whosonfirst/go-reader.Reader instance for reading "parent" features.
	ParentReader reader.Reader
	// The name of the person (or process) updating a depiction.
	Author string
}

// UpdateDepiction will update the geometries and relevant properties for SFOM/WOF records 'depiction_id' and 'subject_id' using
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
func UpdateDepiction(ctx context.Context, opts *UpdateDepictionOptions, update *Depiction) ([]byte, error) {

	depiction_id := update.DepictionId
	geotag_f := update.Feature

	geotag_props := geotag_f.Properties
	camera_parent_id := geotag_props.Camera.ParentId
	target_parent_id := geotag_props.Target.ParentId

	// START OF to refactor with go-writer/v4 (clone) release

	update_opts := &github.UpdateWriterURIOptions{
		WhosOnFirstId: depiction_id,
		Author:        opts.Author,
		Action:        github.GeotagAction,
	}

	depiction_writer_uri, err := github.UpdateWriterURI(ctx, update_opts, opts.DepictionWriterURI)

	if err != nil {
		return nil, fmt.Errorf("Failed to update depiction writer URI, %w", err)
	}

	subject_writer_uri, err := github.UpdateWriterURI(ctx, update_opts, opts.SubjectWriterURI)

	if err != nil {
		return nil, fmt.Errorf("Failed to update subject writer URI, %w", err)
	}

	depiction_writer, err := writer.NewWriter(ctx, depiction_writer_uri)

	if err != nil {
		return nil, fmt.Errorf("Failed to create new depiction writer for '%s', %w", depiction_writer_uri, err)
	}

	subject_writer, err := writer.NewWriter(ctx, subject_writer_uri)

	if err != nil {
		return nil, fmt.Errorf("Failed to create new subject writer for '%s', %w", subject_writer_uri, err)
	}

	// END OF to refactor with go-writer/v4 (clone) release

	// START OF hooks to capture updates/writes so we can parrot them back in the method response
	// We're doing it this way because the code, as written, relies on sfomuseum/go-sfomuseum-writer
	// which hides the format-and-export stages and modifies the document being written. To account
	// for this we'll just keep local copies of those updates in *_buf and reference them at the end.
	// Note that we are not doing this for the alternate geometry feature (alt_body) since are manually
	// formatting, exporting and writing a byte slice in advance of better support for alternate
	// geometries in the tooling.

	// The buffers where we will write updated Feature information
	var local_depiction_buf bytes.Buffer
	var local_subject_buf bytes.Buffer

	// The io.Writers where we will write updated Feature information
	local_depiction_buf_writer := bufio.NewWriter(&local_depiction_buf)
	local_subject_buf_writer := bufio.NewWriter(&local_subject_buf)

	// The writer.Writer where we will write updated Feature information
	local_depiction_writer, err := writer.NewIOWriterWithWriter(ctx, local_depiction_buf_writer)

	if err != nil {
		return nil, fmt.Errorf("Failed to create IOWriter for depiction, %w", err)
	}

	local_subject_writer, err := writer.NewIOWriterWithWriter(ctx, local_subject_buf_writer)

	if err != nil {
		return nil, fmt.Errorf("Failed to create IOWriter for subject, %w", err)
	}

	// The writer.MultiWriter where we will write updated Feature information
	depiction_mw, err := writer.NewMultiWriter(ctx, depiction_writer, local_depiction_writer)

	if err != nil {
		return nil, fmt.Errorf("Failed to create multi writer for depiction, %w", err)
	}

	subject_mw, err := writer.NewMultiWriter(ctx, subject_writer, local_subject_writer)

	if err != nil {
		return nil, fmt.Errorf("Failed to create multi writer for subject, %w", err)
	}

	// END OF hooks to capture updates/writes so we can parrot them back in the method response

	depiction_f, err := wof_reader.LoadBytes(ctx, opts.DepictionReader, depiction_id)

	if err != nil {
		return nil, fmt.Errorf("Failed to load depiction record %d, %w", depiction_id, err)
	}

	parent_rsp := gjson.GetBytes(depiction_f, "properties.wof:parent_id")

	if !parent_rsp.Exists() {
		return nil, fmt.Errorf("Failed to determine wof:parent_id for depiction")
	}

	subject_id := parent_rsp.Int()

	// subject_f is the feature that parents the depiction (for example an object that has one or more depictions)

	subject_f, err := wof_reader.LoadBytes(ctx, opts.SubjectReader, subject_id)

	if err != nil {
		return nil, fmt.Errorf("Failed to load subject record %d, %w", subject_id, err)
	}

	// *_parent_f are the Who's On First place features that parent/contains a geotagging geometry
	// camera is point at which a depiction was created; target is what the depiction is pointing at

	var camera_parent_f []byte
	var target_parent_f []byte

	if camera_parent_id != -1 {

		f, err := wof_reader.LoadBytes(ctx, opts.ParentReader, camera_parent_id)

		if err != nil {
			return nil, fmt.Errorf("Failed to load parent record %d, %w", camera_parent_id, err)
		}

		camera_parent_f = f
	}

	if target_parent_id != -1 {

		f, err := wof_reader.LoadBytes(ctx, opts.ParentReader, target_parent_id)

		if err != nil {
			return nil, fmt.Errorf("Failed to load parent record %d, %w", target_parent_id, err)
		}

		target_parent_f = f

		fmt.Println("DEBUG", target_parent_f)
	}

	// Update the subject

	subject_updates := map[string]interface{}{
		"properties.src:geom": "sfomuseum#geotagged",
	}

	// Update geotag:depictions array to include depiction_id

	tmp := map[int64]bool{
		depiction_id: true,
	}

	depictions_rsp := gjson.GetBytes(subject_f, "properties.geotag:depictions")

	for _, r := range depictions_rsp.Array() {
		id := r.Int()
		tmp[id] = true
	}

	depictions := make([]int64, 0)

	for id, _ := range tmp {
		depictions = append(depictions, id)
	}

	subject_updates["properties.geotag:depictions"] = depictions

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

	coords := make([][]float64, 0)
	coords = append(coords, camera_coord)

	// Fetch other depictions for a given subject; we do this in order to generate
	// a MultiPoint geometry of all the depictions for a subject

	for _, other_id := range depictions {

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

		coords = append(coords, other_coord)
	}

	subject_updates["geometry.type"] = "MultiPoint"
	subject_updates["geometry.coordinates"] = coords

	subject_camera_wof := map[string]interface{}{
		"wof:id": camera_parent_id,
	}

	subject_target_wof := map[string]interface{}{
		"wof:id": target_parent_id,
	}

	// Update the parent ID and hierarchy for the subject

	if camera_parent_f != nil {

		parent_hierarchies := properties.Hierarchies(camera_parent_f)
		subject_camera_wof["wof:hierarchy"] = parent_hierarchies

		belongsto_map := new(sync.Map)

		for _, parent_h := range parent_hierarchies {

			for _, h_id := range parent_h {
				if h_id >= wof.EARTH {
					belongsto_map.Store(h_id, true)
				}
			}
		}

		belongsto := make([]int64, 0)

		belongsto_map.Range(func(k interface{}, v interface{}) bool {
			belongsto = append(belongsto, k.(int64))
			return true
		})

		subject_camera_wof["wof:belongsto"] = belongsto

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
		subject_target_wof["wof:hierarchy"] = parent_hierarchies

		belongsto_map := new(sync.Map)

		for _, parent_h := range parent_hierarchies {

			for _, h_id := range parent_h {
				if h_id >= wof.EARTH {
					belongsto_map.Store(h_id, true)
				}
			}
		}

		belongsto := make([]int64, 0)

		belongsto_map.Range(func(k interface{}, v interface{}) bool {
			belongsto = append(belongsto, k.(int64))
			return true
		})

		subject_target_wof["wof:belongsto"] = belongsto

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

	// v1 (deprecated)
	subject_updates["properties.geotag:whosonfirst"] = subject_camera_wof

	// v2
	subject_updates["properties.geotag:camera_whosonfirst"] = subject_camera_wof
	subject_updates["properties.geotag:target_whosonfirst"] = subject_target_wof

	subject_changed, subject_f, err := export.AssignPropertiesIfChanged(ctx, subject_f, subject_updates)

	if err != nil {
		return nil, fmt.Errorf("Failed to update subject record %d, %w", subject_id, err)
	}

	if subject_changed {

		_, err := sfom_writer.WriteBytes(ctx, subject_mw, subject_f)

		if err != nil {
			return nil, fmt.Errorf("Failed to write subject record %d, %w", subject_id, err)
		}
	}

	// Update the depiction

	depiction_updates := map[string]interface{}{
		"geometry":                           pov,
		"properties.src:geom":                "sfomuseum",
		"properties.geotag:angle":            geotag_f.Properties.Angle,
		"properties.geotag:bearing":          geotag_f.Properties.Bearing,
		"properties.geotag:distance":         geotag_f.Properties.Distance,
		"properties.geotag:camera_longitude": camera_coords[0],
		"properties.geotag:camera_latitude":  camera_coords[1],
		"properties.geotag:target_longitude": target_coords[0],
		"properties.geotag:target_latitude":  target_coords[1],
	}

	to_copy := []string{

		// v1
		"properties.geotag:whosonfirst",
		// v2
		"properties.geotag:camera_whosonfirst",
		"properties.geotag:target_whosonfirst",

		"properties.geotag:depictions",
		"properties.iso:country",
		"properties.wof:country",
		"properties.edtf:inception",
		"properties.edtf:cessation",
		"properties.edtf:date",
	}

	for _, path := range to_copy {

		rsp := gjson.GetBytes(subject_f, path)

		if rsp.Exists() {
			depiction_updates[path] = rsp.Value()
		}
	}

	geom_alt := []string{
		GEOTAG_LABEL,
	}

	geom_alt_rsp := gjson.GetBytes(depiction_f, "properties.src:geom_alt")

	if geom_alt_rsp.Exists() {

		for _, r := range geom_alt_rsp.Array() {

			if r.String() == GEOTAG_LABEL {
				continue
			}

			geom_alt = append(geom_alt, r.String())
		}
	}

	depiction_updates["properties.src:geom_alt"] = geom_alt

	depiction_changed, depiction_f, err := export.AssignPropertiesIfChanged(ctx, depiction_f, depiction_updates)

	if err != nil {
		return nil, fmt.Errorf("Failed to update depiction record %d, %w", depiction_id, err)
	}

	if depiction_changed {

		// _, err := sfom_writer.WriteBytes(ctx, depiction_writer, depiction_f)
		_, err := sfom_writer.WriteBytes(ctx, depiction_mw, depiction_f)

		if err != nil {
			return nil, fmt.Errorf("Failed to write depiction record %d, %w", depiction_id, err)
		}
	}

	// Update the alt depiction geometry

	repo_rsp := gjson.GetBytes(depiction_f, "properties.wof:repo")

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

	_, err = depiction_writer.Write(ctx, alt_uri, alt_fh)

	if err != nil {
		return nil, fmt.Errorf("Failed to write alt file %s, %w", alt_uri, err)
	}

	err = depiction_writer.Close(ctx)

	if err != nil {
		return nil, fmt.Errorf("Failed to close depiction writer, %w", err)
	}

	err = subject_writer.Close(ctx)

	if err != nil {
		return nil, fmt.Errorf("Failed to close subject writer, %w", err)
	}

	//

	local_depiction_buf_writer.Flush()
	local_subject_buf_writer.Flush()

	fc := geojson.NewFeatureCollection()

	new_subject_f, err := geojson.UnmarshalFeature(local_subject_buf.Bytes())

	if err != nil {
		return nil, fmt.Errorf("Failed to unmarshal feature from depiction buffer, %w", err)
	}

	fc.Append(new_subject_f)

	new_depiction_f, err := geojson.UnmarshalFeature(local_depiction_buf.Bytes())

	if err != nil {
		return nil, fmt.Errorf("Failed to unmarshal feature from depiction buffer, %w", err)
	}

	fc.Append(new_depiction_f)

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
