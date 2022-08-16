package georeference

// Eventually it would be good to abstract out all of the SFO Museum stuff from this
// but not today...

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/paulmach/orb/geojson"
	"github.com/sfomuseum/go-sfomuseum-geo/alt"
	"github.com/sfomuseum/go-sfomuseum-geo/github"
	sfom_writer "github.com/sfomuseum/go-sfomuseum-writer/v2"
	"github.com/tidwall/gjson"
	// "github.com/whosonfirst/go-reader"
	"github.com/whosonfirst/go-whosonfirst-export/v2"
	"github.com/whosonfirst/go-whosonfirst-feature/properties"
	wof_reader "github.com/whosonfirst/go-whosonfirst-reader"
	"github.com/whosonfirst/go-whosonfirst-uri"
	"github.com/whosonfirst/go-writer/v2"
	_ "log"
	"sync"
)

func AssignReferences2(ctx context.Context, opts *UpdateDepictionOptions, depiction_id int64, refs ...*Reference) ([]byte, error) {

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	depiction_body, err := wof_reader.LoadBytes(ctx, opts.DepictionReader, depiction_id)

	if err != nil {
		return nil, fmt.Errorf("Failed to create depiction reader, %w", err)
	}

	depiction_repo, err := properties.Repo(depiction_body)

	if err != nil {
		return nil, fmt.Errorf("Unabled to derive wof:repo, %w", err)
	}

	subject_id, err := properties.ParentId(depiction_body)

	if err != nil {
		return nil, fmt.Errorf("Failed to derive subject (parent) ID for depiction, %w", err)
	}

	// START OF to be removed once the go-writer/v3 (Clone) interface is complete

	update_opts := &github.UpdateWriterURIOptions{
		WhosOnFirstId: depiction_id,
		Author:        opts.Author,
		Action:        github.GeoreferenceAction,
	}

	depiction_wr_uri, err := github.UpdateWriterURI(ctx, update_opts, opts.DepictionWriterURI)

	if err != nil {
		return nil, fmt.Errorf("Failed to update depiction writer URI, %w", err)
	}

	depiction_wr, err := writer.NewWriter(ctx, depiction_wr_uri)

	if err != nil {
		return nil, fmt.Errorf("Failed to create new depiction writer, %w", err)
	}

	subject_wr_uri, err := github.UpdateWriterURI(ctx, update_opts, opts.SubjectWriterURI)

	if err != nil {
		return nil, fmt.Errorf("Failed to update subject writer URI, %w", err)
	}

	subject_wr, err := writer.NewWriter(ctx, subject_wr_uri)

	if err != nil {
		return nil, fmt.Errorf("Failed to create new subject writer, %w", err)
	}

	// END OF to be removed once the go-writer/v3 (Clone) interface is complete

	done_ch := make(chan bool)
	err_ch := make(chan error)
	alt_ch := make(chan *alt.WhosOnFirstAltFeature)

	references_map := new(sync.Map)
	updates_map := new(sync.Map)

	new_alt_features := make([]*alt.WhosOnFirstAltFeature, 0)
	other_alt_features := make([]*alt.WhosOnFirstAltFeature, 0)

	for _, r := range refs {

		go func(ctx context.Context, r *Reference) {

			defer func() {
				done_ch <- true
			}()

			id := r.Id
			prop_label := r.Property
			alt_label := r.AltLabel

			body, err := wof_reader.LoadBytes(ctx, opts.WhosOnFirstReader, id)

			if err != nil {
				err_ch <- fmt.Errorf("Failed to read body for %d, %w", id, err)
				return
			}

			path := fmt.Sprintf("properties.%s", prop_label)
			updates_map.Store(path, id)

			hiers := properties.Hierarchies(body)

			for _, h := range hiers {

				for _, h_id := range h {
					references_map.Store(h_id, true)
				}
			}

			pt, _, err := properties.Centroid(body)

			if err != nil {
				err_ch <- fmt.Errorf("Failed to derive centroid for %d, %w", id, err)
				return
			}

			alt_geom := geojson.NewGeometry(pt)

			alt_props := map[string]interface{}{
				"wof:id":        depiction_id,
				"wof:repo":      depiction_repo,
				"src:alt_label": alt_label,
				"src:geom":      "sfomuseum#derived-flightcover",
			}

			alt_feature := &alt.WhosOnFirstAltFeature{
				Type:       "Feature",
				Id:         id,
				Properties: alt_props,
				Geometry:   alt_geom,
			}

			alt_ch <- alt_feature

		}(ctx, r)
	}

	remaining := len(refs)

	for remaining > 0 {

		select {
		case <-ctx.Done():
			break
		case <-done_ch:
			remaining -= 1
		case err := <-err_ch:
			return nil, err
		case alt_f := <-alt_ch:
			new_alt_features = append(new_alt_features, alt_f)
		}
	}

	updates := map[string]interface{}{
		"properties.src:geom": "sfomuseum#georeference",
	}

	references := make([]int64, 0)

	refs_rsp := gjson.GetBytes(depiction_body, "properties.wof:references")

	for _, r := range refs_rsp.Array() {
		references_map.Store(r.Int(), true)
	}

	references_map.Range(func(k interface{}, v interface{}) bool {
		id := k.(int64)
		references = append(references, id)
		return true
	})

	updates["properties.wof:references"] = references

	updates_map.Range(func(k interface{}, v interface{}) bool {
		path := k.(string)
		updates[path] = v
		return true
	})

	// START OF resolve alt files

	// Create a lookup table of the new alt geom labels

	lookup := make(map[string]bool)

	for _, f := range new_alt_features {
		label := f.Properties["src:alt_label"].(string)
		lookup[label] = true
	}

	// Fetch the existing alt geom labels associated with this record

	existing_alt, err := properties.AltGeometries(depiction_body)

	if err != nil {
		return nil, fmt.Errorf("Failed to determine existing alt files, %w", err)
	}

	// Determine whether there are any other alt files (not included in the set of new labels)

	to_fetch := make([]string, 0)

	for _, label := range existing_alt {

		_, ok := lookup[label]

		if !ok {
			to_fetch = append(to_fetch, label)
		}
	}

	// Fetch any extra alt geometries, if necessary

	if len(to_fetch) > 0 {

		done_ch := make(chan bool)
		err_ch := make(chan error)
		alt_ch := make(chan *alt.WhosOnFirstAltFeature)

		for _, label := range to_fetch {

			go func(label string) {

				defer func() {
					done_ch <- true
				}()

				alt_uri_geom := &uri.AltGeom{
					Source: label,
				}

				alt_uri_args := &uri.URIArgs{
					IsAlternate: true,
					AltGeom:     alt_uri_geom,
				}

				alt_uri, err := uri.Id2RelPath(depiction_id, alt_uri_args)

				if err != nil {
					err_ch <- fmt.Errorf("Failed to derive rel path for alt file, %w", err)
					return
				}

				r, err := opts.DepictionReader.Read(ctx, alt_uri)

				if err != nil {
					err_ch <- fmt.Errorf("Failed to read %s, %w", alt_uri, err)
					return
				}

				defer r.Close()

				var f *alt.WhosOnFirstAltFeature

				dec := json.NewDecoder(r)
				err = dec.Decode(&f)

				if err != nil {
					err_ch <- fmt.Errorf("Failed to decode %s, %w", alt_uri, err)
					return
				}

				alt_ch <- f

			}(label)

			remaining := len(to_fetch)

			for remaining > 0 {
				select {
				case <-done_ch:
					remaining -= 1
				case err := <-err_ch:
					return nil, err
				case f := <-alt_ch:
					other_alt_features = append(other_alt_features, f)
				}
			}
		}
	}
	// Combine new and other alt features

	alt_features := make([]*alt.WhosOnFirstAltFeature, 0)

	for _, f := range new_alt_features {
		alt_features = append(alt_features, f)
	}

	for _, f := range other_alt_features {
		alt_features = append(alt_features, f)
	}

	// Use this new list to catalog alt geoms and derived a multipoint geometry

	alt_geoms := make([]string, len(alt_features))

	for idx, f := range alt_features {
		label := f.Properties["src:alt_label"].(string)
		alt_geoms[idx] = label
	}

	updates["properties.src:geom_alt"] = alt_geoms

	// Derive a MultiPoint geometry

	mp_geom, err := alt.DeriveMultiPointGeometry(ctx, alt_features...)

	if err != nil {
		return nil, fmt.Errorf("Failed to derive multi point geometry, %w", err)
	}

	mp_geojson_geom := geojson.NewGeometry(mp_geom)

	updates["geometry"] = mp_geojson_geom

	// Now save the new alt files

	for _, f := range new_alt_features {

		label := f.Properties["src:alt_label"].(string)

		alt_uri_geom := &uri.AltGeom{
			Source: label,
		}

		alt_uri_args := &uri.URIArgs{
			IsAlternate: true,
			AltGeom:     alt_uri_geom,
		}

		alt_uri, err := uri.Id2RelPath(depiction_id, alt_uri_args)

		if err != nil {
			return nil, fmt.Errorf("Failed to derive rel path for alt file, %w", err)
		}

		enc_f, err := alt.FormatAltFeature(f)

		if err != nil {
			return nil, fmt.Errorf("Failed to format %s, %w", alt_uri, err)
		}

		r := bytes.NewReader(enc_f)

		// Note how we're invoking depiction_wr directly because sfom_writer doesn't
		// know how to work with alt files yet.

		_, err = depiction_wr.Write(ctx, alt_uri, r)

		if err != nil {
			return nil, fmt.Errorf("Failed to write %s, %w", alt_uri, err)
		}
	}

	// END OF resolve alt files

	has_changed, new_body, err := export.AssignPropertiesIfChanged(ctx, depiction_body, updates)

	if err != nil {
		return nil, fmt.Errorf("Failed to assign depiction properties, %w", err)
	}

	if has_changed {

		_, err = sfom_writer.WriteBytes(ctx, depiction_wr, new_body)

		if err != nil {
			return nil, fmt.Errorf("Failed to write depiction update, %w", err)
		}
	}

	// Update subject (parent) record

	subject_body, err := wof_reader.LoadBytes(ctx, opts.SubjectReader, subject_id)

	if err != nil {
		return nil, fmt.Errorf("Failed to load subject (parent) for depiction, %w", err)
	}

	subject_updates := make(map[string]interface{})

	references_lookup := new(sync.Map)

	for _, r := range references {
		references_lookup.Store(r, true)
	}

	subject_refs := gjson.GetBytes(subject_body, "properties.wof:references")

	for _, r := range subject_refs.Array() {
		references_lookup.Store(r.Int(), true)
	}

	subject_references := make([]int64, 0)

	references_lookup.Range(func(k interface{}, v interface{}) bool {
		id := v.(int64)
		subject_references = append(subject_references, id)
		return true
	})

	subject_updates["properties.wof:references"] = subject_references

	// Something something something geometry here...

	has_changed, new_subject, err := export.AssignPropertiesIfChanged(ctx, subject_body, subject_updates)

	if err != nil {
		return nil, fmt.Errorf("Failed to assign subject properties, %w", err)
	}

	if has_changed {

		_, err = sfom_writer.WriteBytes(ctx, subject_wr, new_subject)

		if err != nil {
			return nil, fmt.Errorf("Failed to write subject update, %w", err)
		}
	}

	// Wrap up

	err = depiction_wr.Close(ctx)

	if err != nil {
		return nil, fmt.Errorf("Failed to close depiction writer, %w", err)
	}

	err = subject_wr.Close(ctx)

	if err != nil {
		return nil, fmt.Errorf("Failed to close subject writer, %w", err)
	}

	return new_body, nil
}
