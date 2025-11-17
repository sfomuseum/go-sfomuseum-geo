package georeference

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
	"github.com/sfomuseum/go-sfomuseum-geo"
	"github.com/sfomuseum/go-sfomuseum-geo/geometry"
	"github.com/tidwall/gjson"
	"github.com/whosonfirst/go-reader/v2"
	"github.com/whosonfirst/go-whosonfirst-export/v3"
	"github.com/whosonfirst/go-whosonfirst-feature/properties"
	wof_reader "github.com/whosonfirst/go-whosonfirst-reader/v2"
)

type SkipListItem struct {
	Geometry orb.Geometry
	Depicted map[string][]int64
}

type RecompileGeorefencesForSubjectOptions struct {
	DepictionReader          reader.Reader
	SFOMuseumReader          reader.Reader // just settle on WhosOnFirstReader and assume it's a MultiReader... maybe?
	DefaultGeometryFeatureId int64
	SkipList                 map[int64]*SkipListItem
}

func RecompileGeorefencesForSubject(ctx context.Context, opts *RecompileGeorefencesForSubjectOptions, subject_body []byte) (bool, []byte, error) {

	// IMPORTANT: THIS DOES NOT ACCOUNT FOR DEPICTIONS THAT ARE BEING UPDATED AT THE SAME TIME YET
	// Specifically, we need to be able to pass in both:
	// image IDs to skip
	// geometries to include when deriving the final subject geom
	//
	// The point is to be able to call this code directly from AssignGeoreferences (replacing code that is already there)

	subject_depicted_key := fmt.Sprintf("properties.%s", geo.RESERVED_GEOREFERENCE_DEPICTED)
	subject_depictions_key := fmt.Sprintf("properties.%s", geo.RESERVED_GEOREFERENCE_DEPICTIONS)
	subject_belongsto_key := fmt.Sprintf("properties.%s", geo.RESERVED_GEOREFERENCE_BELONGSTO)
	geotag_depicted_key := fmt.Sprintf("properties.%s", geo.RESERVED_GEOTAG_DEPICTIONS)

	logger := slog.Default()

	subject_updates := map[string]any{
		"properties.src:geom": "sfomuseum",
	}

	// The new new
	subject_depicted := make(map[string][]int64)
	subject_depictions := make([]int64, 0)
	subject_belongsto := make([]int64, 0)

	type image_ref struct {
		// The depiction of the subject
		depiction int64
		// The georeference label
		label string
		// The place being depicted
		place_id int64
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
		im_remaining += 1

		logger.Debug("Derive georef details from image", "id", image_id)

		skiplist_item, exists := opts.SkipList[image_id]

		if exists {

			logger.Debug("Image is listed in skip list, process now", "id", image_id)

			if !slices.Contains(subject_depictions, image_id) {
				subject_depictions = append(subject_depictions, image_id)
			}

			// belongs to and depicted here...

			for label, ids := range skiplist_item.Depicted {

				for _, place_id := range ids {

					depicted_ids, exists := subject_depicted[label]

					if !exists {
						depicted_ids = make([]int64, 0)
					}

					if !slices.Contains(depicted_ids, place_id) {
						depicted_ids = append(depicted_ids, place_id)
					}

					subject_depicted[label] = depicted_ids

					if !slices.Contains(subject_belongsto, place_id) {
						subject_belongsto = append(subject_belongsto, place_id)
					}
				}
			}

			continue
		}

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
					place_id := i.Int()
					logger.Debug("Dispatch image", "image", image_id, "key", label, "place", place_id)
					im_ref_ch <- image_ref{label: label, place_id: place_id, depiction: image_id}
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
			return false, nil, fmt.Errorf("Failed to denormalize georeference properties, %w", err)
		case ref := <-im_ref_ch:

			label := ref.label
			depiction_id := ref.depiction
			place_id := ref.place_id

			logger.Debug("Update subject (geo)refs for image", "id", place_id, "label", label)

			if !slices.Contains(subject_depictions, depiction_id) {
				subject_depictions = append(subject_depictions, depiction_id)
			}

			// Update wof:references for subject
			// subject_belongsto_lookup.Store(id, true)

			depicted_ids, exists := subject_depicted[label]

			if !exists {
				depicted_ids = make([]int64, 0)
			}

			if !slices.Contains(depicted_ids, place_id) {
				depicted_ids = append(depicted_ids, place_id)
			}

			subject_depicted[label] = depicted_ids

			if !slices.Contains(subject_belongsto, place_id) {
				subject_belongsto = append(subject_belongsto, place_id)
			}
		}
	}

	subject_updates[subject_depicted_key] = subject_depicted
	subject_updates[subject_depictions_key] = subject_depictions

	// INFLATE BELONGS TO HERE

	subject_updates[subject_belongsto_key] = subject_belongsto

	// START OF derive geometry from geotags and georeferences in depictions
	// It would be nice to believe this code could be abstracted out and shared
	// with equivalent requirements in ../geotag. It probably can but right
	// now that feels a bit too much like yak-shaving.

	geom_ids := subject_depictions

	// Read geotag pointers from subject file

	geotag_depicted_rsp := gjson.GetBytes(subject_body, geotag_depicted_key)

	for _, r := range geotag_depicted_rsp.Array() {

		id := r.Int()

		if !slices.Contains(geom_ids, id) {
			logger.Debug("Add subject geom ID (geotag) to lookup", "id", id)
			geom_ids = append(geom_ids, id)
		}
	}

	logger.Debug("Additional geometries", "count", len(geom_ids))

	var subject_geom orb.Geometry

	if len(geom_ids) == 0 {

		logger.Debug("No geometries (WEIRD), assign geometry and hierarchies from default geometry record", "id", opts.DefaultGeometryFeatureId)

		body, err := wof_reader.LoadBytes(ctx, opts.SFOMuseumReader, opts.DefaultGeometryFeatureId)

		if err != nil {
			logger.Error("Failed to read default geometry record", "id", opts.DefaultGeometryFeatureId, "error", err)
			return false, nil, fmt.Errorf("Failed to read default geometry record, %w", err)
		}

		centroid, _, err := properties.Centroid(body)

		if err != nil {
			logger.Error("Failed to derive centroid for default geometry record", "id", opts.DefaultGeometryFeatureId, "error", err)
			return false, nil, fmt.Errorf("Failed to unmarshal default geometry record, %w", err)
		}

		subject_geom = centroid

	} else {

		logger.Debug("Derive multipoint from geometries (with WOF reader)", "count", len(geom_ids))

		geom, err := geometry.DeriveMultiPointFromIds(ctx, opts.SFOMuseumReader, geom_ids...)

		if err != nil {
			logger.Error("Failed to derive multipoint from geometries (with WOF reader)", "error", err)
			return false, nil, fmt.Errorf("Failed to derive multipoint geometry for subject, %w", err)
		}

		subject_geom = geom
	}

	// Merge subject geom with any geoms explicitly defined in the "skip geom" list

	skip_geoms := make([]orb.Geometry, 0)

	for _, skiplist_item := range opts.SkipList {
		skip_geoms = append(skip_geoms, skiplist_item.Geometry)
	}

	if len(skip_geoms) > 0 {

		logger.Debug("Derive combined geometry from skip list", "count", len(skip_geoms))

		combined_geom, err := geometry.DeriveMultiPointFromGeoms(ctx, skip_geoms...)

		if err != nil {
			logger.Error("Failed to derive multipoint from combined subject and skip geoms", "error", err)
			return false, nil, fmt.Errorf("Failed to derive multipoint from combined subject and skip geoms, %w", err)
		}

		subject_geom = combined_geom
	}

	subject_updates["geometry"] = geojson.NewGeometry(subject_geom)

	// END OF derive geometry from geotags and georeferences in depiction(s)

	subject_has_changed, new_subject, err := export.AssignPropertiesIfChanged(ctx, subject_body, subject_updates)

	if err != nil {
		logger.Error("Failed to assign properties for subject record", "error", err)
		return false, nil, fmt.Errorf("Failed to assign subject properties, %w", err)
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
			return false, nil, fmt.Errorf("Failed to assign last mod properties for subject record, %w", err)
		}
	}

	return subject_has_changed, new_subject, nil
}
