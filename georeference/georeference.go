package georeference

// Eventually it would be good to abstract out all of the SFO Museum stuff from this
// but not today...

import (
	"github.com/sfomuseum/go-sfomuseum-geo/alt"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/paulmach/orb/geojson"
	sfom_writer "github.com/sfomuseum/go-sfomuseum-writer/v2"
	"github.com/tidwall/gjson"
	"github.com/whosonfirst/go-reader"
	"github.com/whosonfirst/go-whosonfirst-export/v2"
	"github.com/whosonfirst/go-whosonfirst-feature/properties"
	wof_reader "github.com/whosonfirst/go-whosonfirst-reader"
	"github.com/whosonfirst/go-whosonfirst-uri"
	"github.com/whosonfirst/go-writer/v2"
	_ "log"
	"net/url"
	"sync"
)

type Reference struct {
	Id       int64  `json:"id"`
	Property string `json:"property"`
	AltLabel string `json:"alt_label"`
}

type AssignReferencesOptions struct {
	WhosOnFirstReader  reader.Reader
	SFOMuseumReader    reader.Reader
	SFOMuseumWriter    writer.Writer
	SFOMuseumWriterURI string // To be removed once the go-writer/v2 interface is complete
	Author             string
}

func AssignReferences(ctx context.Context, opts *AssignReferencesOptions, wof_id int64, refs ...*Reference) ([]byte, error) {

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	body, err := wof_reader.LoadBytes(ctx, opts.SFOMuseumReader, wof_id)

	if err != nil {
		return nil, fmt.Errorf("Failed to read target, %w", err)
	}

	wof_repo, err := properties.Repo(body)

	if err != nil {
		return nil, fmt.Errorf("Unabled to derive wof:repo, %w", err)
	}

	// START OF to be removed once the go-writer/v2 interface is complete

	wr_u, err := url.Parse(opts.SFOMuseumWriterURI)

	if err != nil {
		return nil, fmt.Errorf("Failed to parse URI, %w", err)
	}

	switch wr_u.Scheme {

	case "githubapi":

		update_msg := fmt.Sprintf("[%s] updated georeferences for ", opts.Author)
		update_msg = update_msg + "%s" // I wish I knew how to include a literal '%s' in fmt.Sprintf...

		wr_q := wr_u.Query()

		wr_q.Del("new")
		wr_q.Del("update")

		wr_q.Set("new", update_msg)
		wr_q.Set("update", update_msg)

		// branch...
		wr_u.RawQuery = wr_q.Encode()

	case "githubapi-pr":

		title := fmt.Sprintf("[%s] update georeferences for %d", opts.Author, wof_id)
		description := title

		branch := fmt.Sprintf("%s-%d", opts.Author, wof_id)

		wr_q := wr_u.Query()

		wr_q.Del("pr-branch")
		wr_q.Del("pr-title")
		wr_q.Del("pr-description")

		wr_q.Set("pr-branch", branch)
		wr_q.Set("pr-title", title)
		wr_q.Set("pr-description", description)

		// branch...
		wr_u.RawQuery = wr_q.Encode()
	}

	wr, err := writer.NewWriter(ctx, wr_u.String())

	if err != nil {
		return nil, fmt.Errorf("Failed to update writer, %w", err)
	}

	// END OF to be removed once the go-writer/v2 interface is complete

	// START OF use this once the go-writer/v2 interface is complete
	// To do: Update to account for githubapi-pr writer scheme (see above)

	/*

		wr_u := new(url.URL)
		wr_q := wr_q.Query()

		wr_q.Set("new", update_msg)
		wr_q.Set("update", update_msg)

		wr_u.RawQuery = wr_q.Encode()

		wr, err := opts.SFOMWriter.Clone(ctx, wr_u.String())

		if err != nil {
			return nil, fmt.Errorf("Failed to create new writer, %w", err)
		}
	*/

	// END OF use this once the go-writer/v2 interface is complete

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
				"wof:id":        wof_id,
				"wof:repo":      wof_repo,
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

	refs_rsp := gjson.GetBytes(body, "properties.sfomuseum:references")

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

	existing_alt, err := properties.AltGeometries(body)

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

				alt_uri, err := uri.Id2RelPath(wof_id, alt_uri_args)

				if err != nil {
					err_ch <- fmt.Errorf("Failed to derive rel path for alt file, %w", err)
					return
				}

				r, err := opts.SFOMuseumReader.Read(ctx, alt_uri)

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

		alt_uri, err := uri.Id2RelPath(wof_id, alt_uri_args)

		if err != nil {
			return nil, fmt.Errorf("Failed to derive rel path for alt file, %w", err)
		}

		enc_f, err := alt.FormatAltFeature(f)

		if err != nil {
			return nil, fmt.Errorf("Failed to format %s, %w", alt_uri, err)
		}

		r := bytes.NewReader(enc_f)

		_, err = wr.Write(ctx, alt_uri, r)

		if err != nil {
			return nil, fmt.Errorf("Failed to write %s, %w", alt_uri, err)
		}
	}

	// END OF resolve alt files

	has_changed, new_body, err := export.AssignPropertiesIfChanged(ctx, body, updates)

	if err != nil {
		return nil, fmt.Errorf("Failed to assign new properties, %w", err)
	}

	if has_changed {

		_, err = sfom_writer.WriteBytes(ctx, wr, new_body)

		if err != nil {
			return nil, fmt.Errorf("Failed to write update, %w", err)
		}
	}

	err = wr.Close(ctx)

	if err != nil {
		return nil, fmt.Errorf("Failed to close writer, %w", err)
	}

	return new_body, nil
}
