package georeference

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
	"github.com/sfomuseum/go-sfomuseum-geo"
	"github.com/sfomuseum/go-sfomuseum-geo/geometry"
	"github.com/tidwall/gjson"
	"github.com/whosonfirst/go-reader/v2"
	"github.com/whosonfirst/go-whosonfirst-export/v3"
	"github.com/whosonfirst/go-whosonfirst-feature/geometry"
	wof_reader "github.com/whosonfirst/go-whosonfirst-reader/v2"
)

type RecompileGeorefencesForSubjectOptions struct {
	DepictionReader reader.Reader
	SFOMuseumReader reader.Reader
	Skip            []int64
}

func RecompileGeorefencesForSubject(ctx context.Context, opts *RecompileGeorefencesForSubjectOptions, subject_body []byte) ([]byte, error) {

	logger := slog.Default()

	subject_updates := make(map[string]any)

	subject_references_lookup := new(sync.Map)
	subject_depicted_lookup := new(sync.Map)

	// Add the references assigned to the depiction being updated

	/*
		for _, r := range refs {
			for _, i := range r.Ids {
				subject_references_lookup.Store(i, true)
			}
			subject_depicted_lookup.Store(r.Label, r.Ids)
		}
	*/

	type image_ref struct {
		label string
		id    int64
	}

	im_done_ch := make(chan bool)
	im_err_ch := make(chan error)
	im_ref_ch := make(chan image_ref)

	im_remaining := 0

	// Replace with georeferences array defined above?

	images_rsp := gjson.GetBytes(subject_body, "properties.millsfield:images")
	images_list := images_rsp.Array()

	logger.Debug("Process images for subject", "count", len(images_list))

	for _, r := range images_list {

		image_id := r.Int()

		// Remember these have been assigned above

		if slices.Contains(opts.Skip, image_id) {
			logger.Debug("Skip current depiction", "id", image_id)
			continue
		}

		im_remaining += 1

		logger.Debug("Derive georef details from image", "id", image_id)

		go func(image_id int64) {

			defer func() {
				im_done_ch <- true
			}()

			logger.Debug("Load image", "image id", image_id)

			image_body, err := wof_reader.LoadBytes(ctx, opts.DepictionReader, image_id)

			if err != nil {
				im_err_ch <- fmt.Errorf("Failed to read image ID %d, %w", image_id, err)
				return
			}

			georefs_path := fmt.Sprintf("properties.%s", geo.RESERVED_GEOREFERENCE_DEPICTED)
			georefs_rsp := gjson.GetBytes(image_body, georefs_path)

			logger.Debug("Depictions for image", "image_id", image_id, "key", geo.RESERVED_GEOREFERENCE_DEPICTED, "count", len(georefs_rsp.Array()))

			for _, r := range georefs_rsp.Array() {

				label := r.Get("georef:label").String()
				ids := r.Get("wof:depicts")

				for _, i := range ids.Array() {
					logger.Debug("Dispatch image", "image", image_id, "key", label, "depiction", i.Int())
					im_ref_ch <- image_ref{label: label, id: i.Int()}
				}
			}

		}(image_id)
	}

	// Wait...

	for im_remaining > 0 {
		select {
		case <-im_done_ch:
			im_remaining -= 1
		case err := <-im_err_ch:
			return nil, fmt.Errorf("Failed to denormalize georeference properties, %w", err)
		case ref := <-im_ref_ch:

			label := ref.label
			id := ref.id

			logger.Debug("Update subject (geo)refs for image", "id", id, "label", label)

			// Update wof:references for subject
			subject_references_lookup.Store(id, true)

			// Update georeference:depictions for subject
			var ids []int64

			v, exists := subject_depicted_lookup.Load(label)

			logger.Debug("Depicted for label", "id", id, "label", label, "exists", exists, "depicted", v)

			if exists {
				ids = v.([]int64)
			} else {
				ids = make([]int64, 0)
			}

			if !slices.Contains(ids, id) {
				ids = append(ids, id)
				subject_depicted_lookup.Store(label, ids)
				logger.Debug("Append ID for label", "id", id, "label", label, "ids", id)
			}

		}
	}

	// Assign wof:references (belongs to) for subject

	logger.Debug("Assign georef belongsto for subject")

	subject_wof_references := make([]int64, 0)

	subject_references_lookup.Range(func(k interface{}, v interface{}) bool {
		subject_wof_references = append(subject_wof_references, k.(int64))
		return true
	})

	subject_belongsto_key := fmt.Sprintf("properties.%s", geo.RESERVED_GEOREFERENCE_BELONGSTO)
	subject_updates[subject_belongsto_key] = subject_wof_references

	logger.Debug("Assign subject belongs to", "key", geo.RESERVED_GEOREFERENCE_BELONGSTO, "data", subject_wof_references)

	// Assign georef:depicted for subject

	logger.Debug("Assign georeference:depictions for subject")

	subject_depicted := make(map[string][]int64)

	subject_depicted_lookup.Range(func(k interface{}, v interface{}) bool {
		path := k.(string)
		ids := v.([]int64)
		subject_depicted[path] = ids
		return true
	})

	subject_depicted_key := fmt.Sprintf("properties.%s", geo.RESERVED_GEOREFERENCE_DEPICTED)
	subject_updates[subject_depicted_key] = subject_depicted

	logger.Debug("Assign depicted for subject", "key", geo.RESERVED_GEOREFERENCE_DEPICTED, "data", subject_depicted)

	// Assign georef:depictions for subject

	logger.Debug("Assign georef:depictions for subject")
	subject_depictions_key := fmt.Sprintf("properties.%s", geo.RESERVED_GEOREFERENCE_DEPICTIONS)

	subject_depictions := []int64{
		depiction_id,
	}

	subject_depictions_rsp := gjson.GetBytes(subject_body, subject_depictions_key)

	for _, r := range subject_depictions_rsp.Array() {

		r_id := r.Int()

		if r_id > 0 && !slices.Contains(subject_depictions, r_id) {
			subject_depictions = append(subject_depictions, r_id)
		}
	}

	subject_updates[subject_depictions_key] = subject_depictions
	logger.Debug("Assign depictions for subject", "key", geo.RESERVED_GEOREFERENCE_DEPICTIONS, "data", subject_depictions)

	// START OF derive geometry from geotags and georeferences in depictions
	// It would be nice to believe this code could be abstracted out and shared
	// with equivalent requirements in ../geotag. It probably can but right
	// now that feels a bit too much like yak-shaving.

	geom_ids := subject_depictions

	// Read geotag pointers from subject file

	path_geotag_depicted := fmt.Sprintf("properties.%s", geo.RESERVED_GEOTAG_DEPICTIONS)
	geotag_depicted_rsp := gjson.GetBytes(subject_body, path_geotag_depicted)

	for _, r := range geotag_depicted_rsp.Array() {
		id := r.Int()

		if !slices.Contains(geom_ids, id) {
			logger.Debug("Add subject geom ID (geotag) to lookup", "id", id)
			geom_ids = append(geom_ids, id)
		}
	}

	logger.Debug("Additional geometries", "count", len(geom_ids))

	if len(geom_ids) > 0 {

		logger.Debug("Derive multipoint from geometries (with WOF reader)", "count", len(geom_ids))

		// should we just make a "multi" reader here?
		// geom, err := geometry.DeriveMultiPointFromIds(ctx, opts.WhosOnFirstReader, geom_ids...)
		geom, err := geometry.DeriveMultiPointFromIds(ctx, opts.SFOMuseumReader, geom_ids...)

		if err != nil {
			logger.Error("Failed to derive multipoint from geometries (with WOF reader)", "error", err)
			return nil, fmt.Errorf("Failed to derive multipoint geometry for subject, %w", err)
		}

		// Now append the geometry for the depiction

		subject_orb_geom := geom.Geometry()
		subject_points := subject_orb_geom.(orb.MultiPoint)

		switch depiction_orb_geom.GeoJSONType() {
		case "Point":

			// Note: It's just easier to mint a new Point than to faff around
			// with pointer syntax...

			pt := depiction_orb_geom.(*orb.Point)
			subject_points = append(subject_points, orb.Point{pt.X(), pt.Y()})

		case "MultiPoint":

			depiction_points := depiction_orb_geom.(orb.MultiPoint)

			for _, pt := range depiction_points {
				subject_points = append(subject_points, pt)
			}
		default:
			logger.Error("Unsupported geometry type for depiction", "type", depiction_orb_geom.GeoJSONType())
		}

		new_mp := orb.MultiPoint(subject_points)
		new_geom := geojson.NewGeometry(new_mp)

		subject_updates["geometry"] = new_geom
	} else {
		// No other geometries so just append the geometry for the depiction
		// Which might be the "default" geometry if there are no pointers
		subject_updates["geometry"] = depiction_updates["geometry"]
	}

	// END OF derive geometry from geotags and georeferences in depiction(s)

	subject_has_changed, new_subject, err := export.AssignPropertiesIfChanged(ctx, subject_body, subject_updates)

	if err != nil {
		logger.Error("Failed to assign properties for subject record", "error", err)
		return nil, fmt.Errorf("Failed to assign subject properties, %w", err)
	}

	if subject_has_changed {

		lastmod_key := fmt.Sprintf("properties.%s", geo.RESERVED_GEOREFERENCE_LASTMODIFIED)
		lastmod := time.Now()

		lastmod_updates := map[string]any{
			lastmod_key: lastmod.Unix(),
		}

		new_subject, err = export.AssignProperties(ctx, new_subject, lastmod_updates)

		if err != nil {
			logger.Error("Failed to assign last mod properties for subject record", "error", err)
			return nil, fmt.Errorf("Failed to assign last mod properties for subject record, %w", err)
		}
	}

	return new_subject, nil
}
