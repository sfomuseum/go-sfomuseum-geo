package geotag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/paulmach/orb/geojson"
	"github.com/sfomuseum/go-sfomuseum-geo/alt"
	sfom_writer "github.com/sfomuseum/go-sfomuseum-writer/v2"
	"github.com/tidwall/gjson"
	"github.com/whosonfirst/go-ioutil"
	"github.com/whosonfirst/go-reader"
	"github.com/whosonfirst/go-whosonfirst-export/v2"
	wof_reader "github.com/whosonfirst/go-whosonfirst-reader"
	"github.com/whosonfirst/go-whosonfirst-uri"
	"github.com/whosonfirst/go-writer"
)

// UpdateDepictionOptions defines a struct for reading/writing options when updating geotagging information in depictions.
type UpdateDepictionOptions struct {
	// A valid whosonfirst/go-reader.Reader instance for reading depiction features.
	DepictionReader reader.Reader
	// A valid whosonfirst/go-writer.Writer instance for writing depiction features.
	DepictionWriter writer.Writer
	// A valid whosonfirst/go-reader.Reader instance for reading subject features.
	SubjectReader reader.Reader
	// A valid whosonfirst/go-writer.Writer instance for writing subject features.
	SubjectWriter writer.Writer
	// A valid whosonfirst/go-reader.Reader instance for reading "parent" features.
	ParentReader reader.Reader
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
func UpdateDepiction(ctx context.Context, opts *UpdateDepictionOptions, update *Depiction) error {

	depiction_id := update.DepictionId
	parent_id := update.ParentId
	geotag_f := update.Feature

	depiction_f, err := wof_reader.LoadBytes(ctx, opts.DepictionReader, depiction_id)

	if err != nil {
		return fmt.Errorf("Failed to load depiction record %d, %w", depiction_id, err)
	}

	parent_rsp := gjson.GetBytes(depiction_f, "properties.wof:parent_id")

	if !parent_rsp.Exists() {
		return fmt.Errorf("Failed to determine wof:parent_id for depiction")
	}

	subject_id := parent_rsp.Int()

	subject_f, err := wof_reader.LoadBytes(ctx, opts.SubjectReader, subject_id)

	if err != nil {
		return fmt.Errorf("Failed to load subject record %d, %w", subject_id, err)
	}

	var parent_f []byte

	if parent_id != -1 {

		f, err := wof_reader.LoadBytes(ctx, opts.ParentReader, parent_id)

		if err != nil {
			return fmt.Errorf("Failed to load parent record %d, %w", parent_id, err)
		}

		parent_f = f
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
		return fmt.Errorf("Unable to derive camera point of view, %w", err)
	}

	// TO DO: REMOVE mz:is_approximate if present

	target, err := geotag_f.Target()

	if err != nil {
		return fmt.Errorf("Unable to derive camera target, %w", err)
	}

	camera_coords := pov.Coordinates
	target_coords := target.Coordinates

	// START OF new geometry stuff
	// This remains untested and undocumented (20220308/thisisaaronland)

	camera_coord := []float64{
		camera_coords[0],
		camera_coords[1],
	}

	coords := make([][]float64, 0)
	coords = append(coords, camera_coord)

	// Fetch others

	for _, other_id := range depictions {

		// Skip current depiction as its been added above

		if other_id == depiction_id {
			continue
		}

		other_f, err := wof_reader.LoadBytes(ctx, opts.DepictionReader, other_id)

		if err != nil {
			return fmt.Errorf("Failed to load depiction record %d, %w", other_id, err)
		}

		// TBD: Does this really need to throw a "fatal" error? Is it okay to skip records
		// with missing properties?

		lat_rsp := gjson.GetBytes(other_f, "properties.geotag:camera_latitude")
		lon_rsp := gjson.GetBytes(other_f, "properties.geotag:camera_longitude")

		if !lat_rsp.Exists() {
			return fmt.Errorf("Depiction record %d is missing geotag:camera_latitude property", other_id)
		}

		if !lon_rsp.Exists() {
			return fmt.Errorf("Depiction record %d is missing geotag:camera_longitude property", other_id)
		}

		other_coord := []float64{
			lon_rsp.Float(),
			lat_rsp.Float(),
		}

		coords = append(coords, other_coord)
	}

	subject_updates["geometry.type"] = "MultiPoint"
	subject_updates["geometry.coordinates"] = coords

	// END OF new geometry stuff

	// START OF existing geometry stuff
	/*

		geom_rsp := gjson.GetBytes(subject_f, "geometry")
		type_rsp := geom_rsp.Get("type")

		switch type_rsp.String() {
		case "MultiPoint":

			tmp := make(map[string][]float64)

			k := fmt.Sprintf("%v,%v", camera_coords[0], camera_coords[1])
			tmp[k] = []float64{camera_coords[0], camera_coords[1]}

			coords_rsp := geom_rsp.Get("coordinates")

			for _, c := range coords_rsp.Array() {

				pt := make([]float64, 0)

				for _, v := range c.Array() {
					pt = append(pt, v.Float())
				}

				k := fmt.Sprintf("%v,%v", pt[0], pt[1])
				tmp[k] = pt
			}

			coords := make([][]float64, 0)

			for _, pt := range tmp {
				coords = append(coords, pt)
			}

			subject_updates["geometry.coordinates"] = coords

		case "Point":

			subject_updates["geometry.type"] = "MultiPoint"

			subject_updates["geometry.coordinates"] = [][]float64{
				[]float64{camera_coords[0], camera_coords[1]},
			}

		default:
			return fmt.Errorf("Unsupported geometry type (%s) for subject record", type_rsp.String())
		}

	*/
	// END OF existing geometry stuff

	// Update the parent ID and hierarchy for the subject

	if parent_f != nil {

		id_rsp := gjson.GetBytes(parent_f, "properties.wof:id")
		subject_updates["properties.wof:parent_id"] = id_rsp.Int()

		to_copy := []string{
			"properties.wof:hierarchy",
			"properties.iso:country",
			"properties.wof:country",
		}

		for _, path := range to_copy {
			rsp := gjson.GetBytes(subject_f, path)
			subject_updates[path] = rsp.Value()
		}
	}

	subject_changed, subject_f, err := export.AssignPropertiesIfChanged(ctx, subject_f, subject_updates)

	if err != nil {
		return fmt.Errorf("Failed to update subject record %d, %w", subject_id, err)
	}

	if subject_changed {

		_, err := sfom_writer.WriteBytes(ctx, opts.SubjectWriter, subject_f)

		if err != nil {
			return fmt.Errorf("Failed to write subject record %d, %w", subject_id, err)
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
		"properties.wof:hierarchy",
		"properties.iso:country",
		"properties.wof:country",
		"properties.edtf:inception",
		"properties.edtf:cessation",
		"properties.edtf:date",
	}

	for _, path := range to_copy {
		rsp := gjson.GetBytes(subject_f, path)
		depiction_updates[path] = rsp.Value()
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
		return fmt.Errorf("Failed to update depiction record %d, %w", depiction_id, err)
	}

	if depiction_changed {

		_, err := sfom_writer.WriteBytes(ctx, opts.DepictionWriter, depiction_f)

		if err != nil {
			return fmt.Errorf("Failed to write depiction record %d, %w", depiction_id, err)
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
		return fmt.Errorf("Failed to derive field of view geometry, %w", err)
	}

	enc_fov, err := json.Marshal(fov_geom)

	if err != nil {
		return err
	}

	geojson_geom, err := geojson.UnmarshalGeometry(enc_fov)

	if err != nil {
		return err
	}

	alt_feature := &alt.WhosOnFirstAltFeature{
		Type:       "Feature",
		Id:         depiction_id,
		Properties: alt_props,
		Geometry:   geojson_geom,
	}

	alt_body, err := alt.FormatAltFeature(alt_feature)

	if err != nil {
		return fmt.Errorf("Failed to format alt feature, %w", err)
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
		return fmt.Errorf("Failed to derive rel path for alt file, %w", err)
	}

	alt_br := bytes.NewReader(alt_body)
	alt_fh, err := ioutil.NewReadSeekCloser(alt_br)

	if err != nil {
		return fmt.Errorf("Failed to create new ReadSeekCloser, %w", err)
	}

	_, err = opts.DepictionWriter.Write(ctx, alt_uri, alt_fh)

	if err != nil {
		return fmt.Errorf("Failed to write alt file %s, %w", alt_uri, err)
	}

	//

	return nil
}