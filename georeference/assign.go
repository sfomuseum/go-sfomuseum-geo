package georeference

// Eventually it would be good to abstract out all of the SFO Museum stuff from this
// but not today...

import (
	"bufio"
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
	"github.com/sfomuseum/go-sfomuseum-geo"
	"github.com/sfomuseum/go-sfomuseum-geo/alt"
	"github.com/sfomuseum/go-sfomuseum-geo/geometry"
	"github.com/sfomuseum/go-sfomuseum-geo/github"
	sfom_writer "github.com/sfomuseum/go-sfomuseum-writer/v3"
	"github.com/tidwall/gjson"
	"github.com/whosonfirst/go-reader"
	"github.com/whosonfirst/go-whosonfirst-export/v2"
	"github.com/whosonfirst/go-whosonfirst-feature/properties"
	wof_reader "github.com/whosonfirst/go-whosonfirst-reader"
	"github.com/whosonfirst/go-whosonfirst-uri"
	"github.com/whosonfirst/go-writer-featurecollection/v3"
	"github.com/whosonfirst/go-writer/v3"
)

// AssignReferencesOptions defines a struct for reading/writing options when updating geo-related information in depictions.
// A depiction is assumed to be the record for an image or some other piece of media. A subject is assumed to be
// the record for an object.
type AssignReferencesOptions struct {
	// A valid whosonfirst/go-reader.Reader instance for reading depiction features. A depiction might be an image of a collection object.
	DepictionReader reader.Reader
	// A valid whosonfirst/go-writer.Writer instance for writing depiction features. A depiction might be an image of a collection object.
	DepictionWriter writer.Writer
	// A valid whosonfirst/go-reader.Reader instance for reading subject features. A subject might be a collection object (rather than any one image (depiction) of that object)
	SubjectReader reader.Reader
	// A valid whosonfirst/go-writer.Writer instance for writing subject features. A subject might be a collection object (rather than any one image (depiction) of that object.
	SubjectWriter writer.Writer
	// A valid whosonfirst/go-reader.Reader instance for reading "parent" features.
	WhosOnFirstReader reader.Reader
	// Author is the name of a person to associate with commit messages if using a `githubapi://` writer
	Author string
	// SourceGeomSuffix is an additional suffix to append to 'src:geom' properties (default is "sfomuseum#geoference")
	SourceGeomSuffix string
	// DepictionWriterURI is the URI used to create `DepictionWriter`; it is a temporary necessity to be removed with the go-writer/v3 (clone) release
	DepictionWriterURI string
	// SubjectWriterURI is the URI used to create `SubjectWriter`; it is a temporary necessity to be removed with the go-writer/v3 (clone) release
	SubjectWriterURI string
	// A valid whosonfirst/go-reader.Reader instance for reading "sfomuseum" features (for example the aviation collection).
	SFOMuseumReader reader.Reader
}

// AssignReferences updates records associated with 'depiction_id' (that is the depiction record itself and it's "parent" object record)
// and 'refs'. An alternate geometry file will be created for each element in 'ref' and a multi-point geometry (derived from 'refs') will
// be assigned to the depiction and parent (subject) record.
func AssignReferences(ctx context.Context, opts *AssignReferencesOptions, depiction_id int64, refs ...*Reference) ([]byte, error) {

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	logger := slog.Default()
	logger = logger.With("depiction id", depiction_id)

	src_geom := "sfomuseum#georeference"

	if opts.SourceGeomSuffix != "" {
		src_geom = fmt.Sprintf("%s-%s", src_geom, opts.SourceGeomSuffix)
		logger.Debug("Automatically assign source geom suffix", "suffix", src_geom)
	}

	depiction_body, err := wof_reader.LoadBytes(ctx, opts.DepictionReader, depiction_id)

	if err != nil {
		return nil, fmt.Errorf("Failed to create depiction reader, %w", err)
	}

	depiction_repo, err := properties.Repo(depiction_body)

	if err != nil {
		return nil, fmt.Errorf("Unable to derive wof:repo for depiction %d, %w", depiction_id, err)
	}

	subject_id, err := properties.ParentId(depiction_body)

	if err != nil {
		return nil, fmt.Errorf("Failed to derive subject (parent) ID for depiction, %w", err)
	}

	logger = logger.With("subject id", subject_id)

	// START OF to be removed once the go-writer/v4 (Clone) interface is complete

	update_opts := &github.UpdateWriterURIOptions{
		WhosOnFirstId: depiction_id,
		Author:        opts.Author,
		Action:        github.GeoreferenceAction,
	}

	depiction_wr_uri, err := github.UpdateWriterURI(ctx, update_opts, opts.DepictionWriterURI)

	if err != nil {
		return nil, fmt.Errorf("Failed to update depiction writer URI, %w", err)
	}

	depiction_writer, err := writer.NewWriter(ctx, depiction_wr_uri)

	if err != nil {
		return nil, fmt.Errorf("Failed to create new depiction writer, %w", err)
	}

	subject_wr_uri, err := github.UpdateWriterURI(ctx, update_opts, opts.SubjectWriterURI)

	if err != nil {
		return nil, fmt.Errorf("Failed to update subject writer URI, %w", err)
	}

	subject_writer, err := writer.NewWriter(ctx, subject_wr_uri)

	if err != nil {
		return nil, fmt.Errorf("Failed to create new subject writer, %w", err)
	}

	// END OF to be removed once the go-writer/v4 (Clone) interface is complete

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

	// Note that we are writing to a writer.FeatureCollectionWriter instead of a writer.IOWriter
	// instance. This is because we end writing (potentially) multiple alternate geometries (as
	// well as the depiction (image)) below. A FeatureCollectionWriter allows us to iterate over
	// the results when we are constructing the final response body at the end of this function.

	local_depiction_wr, err := featurecollection.NewFeatureCollectionWriterWithWriter(ctx, local_depiction_buf_writer)

	if err != nil {
		return nil, fmt.Errorf("Failed to create FeatureCollection writer, %w", err)
	}

	local_subject_wr, err := writer.NewIOWriterWithWriter(ctx, local_subject_buf_writer)

	if err != nil {
		return nil, fmt.Errorf("Failed to create IOWriter for subject, %w", err)
	}

	// The writer.MultiWriter where we will write updated Feature information
	depiction_mw, err := writer.NewMultiWriter(ctx, depiction_writer, local_depiction_wr)

	if err != nil {
		return nil, fmt.Errorf("Failed to create multi writer for depiction, %w", err)
	}

	subject_mw, err := writer.NewMultiWriter(ctx, subject_writer, local_subject_wr)

	if err != nil {
		return nil, fmt.Errorf("Failed to create multi writer for subject, %w", err)
	}

	// END OF hooks to capture updates/writes so we can parrot them back in the method response

	// TBD: use of https://github.com/whosonfirst/go-reader-cachereader for reading depictions
	// Maybe check for non-nil opts.DepictionCache and update depiction_reader accordingly?

	var depiction_reader reader.Reader

	depiction_reader = opts.DepictionReader

	// Okay, so there's a lot going on here. Given a depiction (image) and (n) references we want to:
	// * Create or update the list of alt files (one alt file per reference) associated with the depiction
	// * Update the depiction file with the:
	//   * Updated list of alt files
	//   * An updated list of references (the `georeference:depictions` property)
	//   * An updated list of `wof:hierarchy` elements derived from `georeference:depictions` and `geotag:depictions`
	//   * An updated MultiPoint geometry derived from the geometries of the pointers in `georeference:depictions` and `geotag:depictions`
	// * Update the "subject" file of the depiction (for example the object associated with an image) with the:
	//   * Updated list of alt files
	//   * An updated list of references derived from the `georeference:depictions` property of all the depictions (images)
	//   * An updated list of `wof:hierarchy` elements derived from `georeference:depictions` and `geotag:depictions` and... SFO (?) derived from all the depictions (images) <-- this is not being done yet
	//   * An updated MultiPoint geometry derived from all the depictions (images)

	// START OF update the depiction record

	logger.Debug("Start updating depiction record")

	done_ch := make(chan bool)
	err_ch := make(chan error)
	alt_ch := make(chan *alt.WhosOnFirstAltFeature)

	// Map of any given wof:hierarchy dictionary where the value is the dictionary
	// and the key is the hash of the md5 sum of the JSON-encoded dictionary
	hierarchies_hash_map := new(sync.Map)

	// The set of unique hashed hierarchies (see above) across all the references
	hier_hashes := make([]string, 0)

	// Mutex for reading/writing to hier_hashes inside Go routines
	hier_mu := new(sync.RWMutex)

	references_map := new(sync.Map)
	updates_map := new(sync.Map)

	// START OF create/update alt files for references

	new_alt_features := make([]*alt.WhosOnFirstAltFeature, 0)
	other_alt_features := make([]*alt.WhosOnFirstAltFeature, 0)

	// Start iterating references to assign

	for _, r := range refs {

		logger.Info("Process reference", "ref", r.Label, "ids", r.Ids, "alt", r.AltLabel)

		go func(ctx context.Context, r *Reference) {

			defer func() {
				done_ch <- true
			}()

			if len(r.Ids) == 0 {
				return
			}

			prop_label := r.Label
			alt_label := r.AltLabel

			// Note we are only assigning the base path for this key (prop_label)
			// updates_map is "range-ed" below and we build a new new_depictions
			// dict which is then assigned to properties.{geo.RESERVED_GEOREFERENCE_DEPICTIONS}

			updates_map.Store(prop_label, r.Ids)

			count := len(r.Ids)
			points := make([]orb.Point, count)

			// Remember any given reference (label) can have mutiple WOF IDs
			// Fetch centroid and hierarchy for each ID in a reference

			for idx, id := range r.Ids {

				logger := slog.Default()
				logger = logger.With("label", prop_label)
				logger = logger.With("id", id)

				logger.Debug("Process reference")

				body, err := wof_reader.LoadBytes(ctx, opts.WhosOnFirstReader, id)

				if err != nil {
					err_ch <- fmt.Errorf("Failed to read record for WOF ID %d, %w", id, err)
					return
				}

				hiers := properties.Hierarchies(body)

				for _, h := range hiers {

					for _, h_id := range h {
						references_map.Store(h_id, true)
					}

					enc_h, err := json.Marshal(h)

					if err != nil {
						logger.Error("Failed to marshal hierarchy", "error", err)
						err_ch <- fmt.Errorf("Failed to marshal hierarchy for %d, %w", id, err)
						return
					}

					md5_h := fmt.Sprintf("%x", md5.Sum(enc_h))
					hierarchies_hash_map.Store(md5_h, h)

					hier_mu.Lock()

					if !slices.Contains(hier_hashes, md5_h) {
						hier_hashes = append(hier_hashes, md5_h)
					}

					hier_mu.Unlock()
				}

				pt, _, err := properties.Centroid(body)

				if err != nil {
					logger.Error("Failed to derive centroid", "error", err)
					err_ch <- fmt.Errorf("Failed to derive centroid for %d, %w", id, err)
					return
				}

				points[idx] = *pt
			}

			mp := orb.MultiPoint(points)
			alt_geom := geojson.NewGeometry(mp)

			alt_props := map[string]interface{}{
				"wof:id":        depiction_id,
				"wof:repo":      depiction_repo,
				"src:alt_label": alt_label,
				"src:geom":      src_geom,
			}

			alt_props[prop_label] = r.Ids

			alt_feature := &alt.WhosOnFirstAltFeature{
				Type:       "Feature",
				Id:         depiction_id,
				Properties: alt_props,
				Geometry:   alt_geom,
			}

			logger.Debug("Return alt feature", "label", alt_label)
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
			logger.Error("Alt file processing for referent failed", "error", err)
			return nil, err
		case alt_f := <-alt_ch:
			new_alt_features = append(new_alt_features, alt_f)
			logger.Debug("Append new alt feature", "count", len(new_alt_features))
		}
	}

	// START OF create/update alt files for references

	logger.Debug("Create/update alt files for references")

	depiction_updates := map[string]interface{}{
		"properties.src:geom": src_geom,
	}

	// START OF assign/update wof:references for depiction

	references := make([]int64, 0)

	for _, r := range refs {
		for _, i := range r.Ids {
			references_map.Store(i, true)
		}
	}

	references_map.Range(func(k interface{}, v interface{}) bool {
		id := k.(int64)
		references = append(references, id)
		return true
	})

	// logger.Debug("References for depiction", "count", len(references))

	depiction_updates[fmt.Sprintf("properties.%s", geo.RESERVED_WOF_REFERENCES)] = references

	// END OF assign/update wof:references for depictionx

	// START OF assign/update georeference:depictions here

	new_depictions := make([]map[string]any, 0)

	updates_map.Range(func(k interface{}, v interface{}) bool {
		
		label := k.(string)
		ids := v.([]int64)

		d := map[string]any{
			geo.RESERVED_GEOREFERENCE_LABEL:   label,
			geo.RESERVED_WOF_DEPICTS: ids,
		}

		new_depictions = append(new_depictions, d)
		return true
	})

	depiction_updates[fmt.Sprintf("properties.%s", geo.RESERVED_GEOREFERENCE_DEPICTIONS)] = new_depictions

	// END OF assign/update georeference:depictions here

	// START OF resolve alt files

	logger.Debug("Resolve alt files for depictions")

	// Create a lookup table of the new alt geom labels

	lookup := make(map[string]bool)

	for _, f := range new_alt_features {
		label := f.Properties["src:alt_label"].(string)
		lookup[label] = true
	}

	// Fetch the existing alt geom labels associated with this record

	logger.Debug("Fetch existing alt geometries")

	existing_alt, err := properties.AltGeometries(depiction_body)

	if err != nil {
		return nil, fmt.Errorf("Failed to determine existing alt files, %w", err)
	}

	// Determine whether there are any other alt files (not included in the set of new labels)

	// First create a lookup table for alt files that need to be "removed"

	to_remove := make(map[string]*Reference)

	for _, r := range refs {

		if len(r.Ids) == 0 {
			to_remove[r.AltLabel] = r
		}
	}

	// Now build the list of features (used to build alt files) to fetch
	// Note how we are skipping features to remove

	to_fetch := make([]string, 0)

	for _, label := range existing_alt {

		_, ok_lookup := lookup[label]
		_, ok_remove := to_remove[label]

		if !ok_lookup && !ok_remove {
			logger.Debug("Append to fetch", "label", label)
			to_fetch = append(to_fetch, label)
		}
	}

	// Fetch any extra alt geometries, if necessary

	if len(to_fetch) > 0 {

		logger.Debug("Fetch additional alt features", "count", len(to_fetch))

		done_ch := make(chan bool)
		err_ch := make(chan error)
		alt_ch := make(chan *alt.WhosOnFirstAltFeature)

		for _, label := range to_fetch {

			go func(label string) {

				defer func() {
					done_ch <- true
				}()

				logger.Debug("Fetch alt feature", "label", label)

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

				r, err := depiction_reader.Read(ctx, alt_uri)

				if err != nil {
					err_ch <- fmt.Errorf("Failed to read depiction alt file %s, %w", alt_uri, err)
					return
				}

				defer r.Close()

				var f *alt.WhosOnFirstAltFeature

				dec := json.NewDecoder(r)
				err = dec.Decode(&f)

				if err != nil {
					err_ch <- fmt.Errorf("Failed to decode depiction alt_file %s, %w", alt_uri, err)
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

	logger.Debug("Compile new and existing alt features")

	alt_features := make([]*alt.WhosOnFirstAltFeature, 0)

	for _, f := range new_alt_features {
		alt_features = append(alt_features, f)
	}

	for _, f := range other_alt_features {
		alt_features = append(alt_features, f)
	}

	// Use this new list to catalog alt geoms and derived a multipoint geometry

	logger.Debug("Calculate multipoint geometry for alt geoms")

	alt_geoms := make([]string, len(alt_features))

	for idx, f := range alt_features {
		label := f.Properties["src:alt_label"].(string)
		alt_geoms[idx] = label
	}

	depiction_updates["properties.src:geom_alt"] = alt_geoms

	// Derive a MultiPoint geometry

	mp_geom, err := alt.DeriveMultiPointGeometry(ctx, alt_features...)

	if err != nil {
		return nil, fmt.Errorf("Failed to derive multi point geometry, %w", err)
	}

	mp_geojson_geom := geojson.NewGeometry(mp_geom)

	depiction_updates["geometry"] = mp_geojson_geom

	// Now save the new alt files

	logger.Debug("Save alt files")

	for _, f := range new_alt_features {

		label := f.Properties["src:alt_label"].(string)

		logger.Debug("Save alt feature", "label", label)

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

		logger.Debug("Save alt feature", "uri", alt_uri)

		enc_f, err := alt.FormatAltFeature(f)

		if err != nil {
			return nil, fmt.Errorf("Failed to format %s, %w", alt_uri, err)
		}

		r := bytes.NewReader(enc_f)

		// Note how we're invoking depiction_wr directly because sfom_writer doesn't
		// know how to work with alt files yet.

		_, err = depiction_mw.Write(ctx, alt_uri, r)

		if err != nil {
			return nil, fmt.Errorf("Failed to write %s, %w", alt_uri, err)
		}
	}

	// Now rewrite alt files that need to be "removed"

	logger.Debug("Rewrite alt files to \"remove\"")

	for _, ref := range to_remove {

		logger.Debug("Remove alt file", "label", ref.AltLabel)

		alt_uri_geom := &uri.AltGeom{
			Source: ref.AltLabel,
		}

		alt_uri_args := &uri.URIArgs{
			IsAlternate: true,
			AltGeom:     alt_uri_geom,
		}

		alt_uri, err := uri.Id2RelPath(depiction_id, alt_uri_args)

		if err != nil {
			return nil, fmt.Errorf("Failed to derive rel path for alt file, %w", err)
		}

		logger.Debug("Remove alt file", "label", alt_uri)

		// In advance of a generic "exists" method/package this will have to do...

		_, err = depiction_reader.Read(ctx, alt_uri)

		if err != nil {
			continue
		}

		now := time.Now()
		deprecated := now.Format("2006-01-02")

		alt_props := map[string]interface{}{
			"edtf:deprecated": deprecated,
			"src:alt_label":   ref.AltLabel,
			"src:geom":        "sfomuseum#georeference",
			"wof:id":          depiction_id,
			"wof:repo":        "sfomuseum-data-media-collection",
		}

		pt := orb.Point{0.0, 0.0}
		alt_geom := geojson.NewGeometry(pt)

		alt_f := &alt.WhosOnFirstAltFeature{
			Type:       "Feature",
			Id:         depiction_id,
			Properties: alt_props,
			Geometry:   alt_geom,
		}

		enc_f, err := alt.FormatAltFeature(alt_f)

		if err != nil {
			return nil, fmt.Errorf("Failed to format %s, %w", alt_uri, err)
		}

		r := bytes.NewReader(enc_f)

		// Note how we're invoking depiction_wr directly because sfom_writer doesn't
		// know how to work with alt files yet.

		_, err = depiction_mw.Write(ctx, alt_uri, r)

		if err != nil {
			return nil, fmt.Errorf("Failed to write %s, %w", alt_uri, err)
		}

	}

	// END OF resolve alt files

	// Update wof:hierarchy (ies) for depiction

	logger.Debug("Update depiction hierarchies")

	depiction_hierarchies := make([]map[string]int64, 0)

	for _, md5_h := range hier_hashes {

		v, exists := hierarchies_hash_map.Load(md5_h)

		if !exists {
			return nil, fmt.Errorf("Failed to load hashed hierarchy (%s) for %d", md5_h, depiction_id)
		}

		h := v.(map[string]int64)
		depiction_hierarchies = append(depiction_hierarchies, h)
	}

	depiction_updates["properties.wof:hierarchy"] = depiction_hierarchies

	// Has anything changed?

	depiction_has_changed, new_body, err := export.AssignPropertiesIfChanged(ctx, depiction_body, depiction_updates)

	if err != nil {
		return nil, fmt.Errorf("Failed to assign depiction properties, %w", err)
	}

	// Write changes

	logger.Debug("Has depiction changed", "changes", depiction_has_changed)

	if depiction_has_changed {

		_, err = sfom_writer.WriteBytes(ctx, depiction_mw, new_body)

		if err != nil {
			return nil, fmt.Errorf("Failed to write depiction update, %w", err)
		}
	}

	// END OF update the depiction record

	logger.Debug("Finished updating depiction")
	logger.Debug("Start updating subject")

	// START OF update the subject (parent) record

	subject_hierarchies := make([]map[string]int64, 0)

	subject_body, err := wof_reader.LoadBytes(ctx, opts.SubjectReader, subject_id)

	if err != nil {
		return nil, fmt.Errorf("Failed to load subject (parent) for depiction, %w", err)
	}

	// As in: Aviation Museum, Library, etc.
	collection_id, err := properties.ParentId(subject_body)

	if err != nil {
		return nil, fmt.Errorf("Failed to load parent record for subject %d, %w", subject_id, err)
	}

	logger = logger.With("collection", collection_id)

	col_body, err := wof_reader.LoadBytes(ctx, opts.SFOMuseumReader, collection_id)

	if err != nil {
		return nil, fmt.Errorf("Failed to load collection (%d) record, %w", collection_id, err)
	}

	logger.Debug("Derive hierarchies for collection")

	hiers := properties.Hierarchies(col_body)

	for _, h := range hiers {

		for _, h_id := range h {
			references_map.Store(h_id, true)
		}

		enc_h, err := json.Marshal(h)

		if err != nil {
			return nil, fmt.Errorf("Failed to marshal hierarchy for %d, %w", collection_id, err)
		}

		md5_h := fmt.Sprintf("%x", md5.Sum(enc_h))
		hierarchies_hash_map.Store(md5_h, h)

		hier_mu.Lock()

		if !slices.Contains(hier_hashes, md5_h) {
			hier_hashes = append(hier_hashes, md5_h)
		}

		hier_mu.Unlock()
	}

	for _, md5_h := range hier_hashes {

		v, exists := hierarchies_hash_map.Load(md5_h)

		if !exists {
			return nil, fmt.Errorf("Failed to load hashed hierarchy (%s) for %d", md5_h, depiction_id)
		}

		h := v.(map[string]int64)
		subject_hierarchies = append(subject_hierarchies, h)
	}

	logger.Debug("Subject hierarchies", "count", len(subject_hierarchies))

	// Things to update in the subject (object) record

	subject_updates := map[string]interface{}{
		"properties.src:geom":      depiction_updates["properties.src:geom"],
		"properties.wof:hierarchy": subject_hierarchies,
	}

	// Lookup table for all the georeference properties across all the images for an object

	// START OF derive geometry from depictions (media/image files)

	// Build a list of IDs for geotag AND georeference pointer from which
	// a multipoint geometry will be derived for the subject record

	geoms_lookup := new(sync.Map)
	geom_ids := make([]int64, 0)

	path_geotag_depictions := fmt.Sprintf("properties.%s", geo.RESERVED_GEOTAG_DEPICTIONS)
	geotag_depictions_rsp := gjson.GetBytes(subject_body, path_geotag_depictions)

	for _, r := range geotag_depictions_rsp.Array() {
		id := r.Int()
		geoms_lookup.Store(id, true)
	}

	path_georeference_depictions := fmt.Sprintf("properties.%s", geo.RESERVED_GEOREFERENCE_DEPICTIONS)
	georef_depictions_rsp := gjson.GetBytes(subject_body, path_georeference_depictions)

	for _, r := range georef_depictions_rsp.Map() {

		for _, i := range r.Array() {
			id := i.Int()
			geoms_lookup.Store(id, true)
		}
	}

	geoms_lookup.Range(func(k interface{}, v interface{}) bool {
		id := k.(int64)

		// Do not try to fetch the geometry for depiction ID from depiction_reader
		// because it hasn't been written/published yet and we will update things from
		// memory below

		if id != depiction_id {
			geom_ids = append(geom_ids, id)
		}

		return true
	})

	logger.Debug("Additional geometries", "count", len(geom_ids))

	if len(geom_ids) > 0 {

		geom, err := geometry.DeriveMultiPointFromIds(ctx, depiction_reader, geom_ids...)

		if err != nil {
			return nil, fmt.Errorf("Failed to derive multipoint geometry for subject, %w", err)
		}

		// Now append the geometry for the depiction

		subject_orb_geom := geom.Geometry()
		subject_points := subject_orb_geom.(orb.MultiPoint)

		depiction_geom := depiction_updates["geometry"].(*geojson.Geometry)
		depiction_orb_geom := depiction_geom.Geometry()
		depiction_points := depiction_orb_geom.(orb.MultiPoint)

		for _, pt := range depiction_points {
			subject_points = append(subject_points, pt)
		}

		new_mp := orb.MultiPoint(subject_points)
		new_geom := geojson.NewGeometry(new_mp)

		subject_updates["geometry"] = new_geom
	} else {
		// No other gemoetries so just append the geometry for the depiction
		subject_updates["geometry"] = depiction_updates["geometry"]
	}

	// END OF derive geometry from depictions (media/image files)

	// START OF denormalize all the georeferenced properties from all the images (depictions) in to the object record

	// START OF update wof:references and georeference:depictions for subject

	subject_references_lookup := new(sync.Map)
	subject_depictions_lookup := new(sync.Map)

	// Add the references assigned to the depiction being updated

	for _, r := range refs {
		for _, i := range r.Ids {
			subject_references_lookup.Store(i, true)
		}
		subject_depictions_lookup.Store(r.Label, r.Ids)
	}

	type image_ref struct {
		path string
		id   int64
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

		if image_id == depiction_id {
			logger.Debug("Skip current depiction", "id", image_id)
			continue
		}

		im_remaining += 1

		go func(image_id int64) {

			defer func() {
				im_done_ch <- true
			}()

			logger.Debug("Load image", "image id", image_id)

			image_body, err := wof_reader.LoadBytes(ctx, depiction_reader, image_id)

			if err != nil {
				im_err_ch <- fmt.Errorf("Failed to read image ID %d, %w", image_id, err)
				return
			}

			georefs_path := fmt.Sprintf("properties.%s", geo.RESERVED_GEOREFERENCE_DEPICTIONS)
			georefs_rsp := gjson.GetBytes(image_body, georefs_path)

			for k, ids := range georefs_rsp.Map() {

				for _, r := range ids.Array() {
					logger.Debug("Dispatch image", "image", image_id, "key", k, "depiction", r.Int())
					im_ref_ch <- image_ref{path: k, id: r.Int()}
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

			path := ref.path
			id := ref.id

			// Update wof:references for subject
			subject_references_lookup.Store(id, true)

			// Update georeference:depictions for subject
			var ids []int64

			v, exists := subject_depictions_lookup.Load(path)

			if exists {
				ids = v.([]int64)
			} else {
				ids = make([]int64, 0)
			}

			if !slices.Contains(ids, id) {
				ids = append(ids, id)
				subject_depictions_lookup.Store(path, ids)
			}

		}
	}

	// Assign wof:references for subject

	logger.Debug("Assign wof:references for subject")

	subject_wof_references := make([]int64, 0)

	subject_references_lookup.Range(func(k interface{}, v interface{}) bool {
		subject_wof_references = append(subject_wof_references, k.(int64))
		return true
	})

	subject_updates[fmt.Sprintf("properties.%s", geo.RESERVED_WOF_REFERENCES)] = subject_wof_references

	// Assign georeference:depictions for subject

	logger.Debug("Assign georeference:depictions for subject")

	subject_depictions := make(map[string][]int64)

	subject_depictions_lookup.Range(func(k interface{}, v interface{}) bool {
		path := k.(string)
		ids := v.([]int64)
		subject_depictions[path] = ids
		return true
	})

	subject_updates[fmt.Sprintf("properties.%s", geo.RESERVED_GEOREFERENCE_DEPICTIONS)] = subject_depictions

	/*
		for k, v := range subject_updates {
			slog.Debug("UPDATE", "k", k, "v", v)
		}
	*/

	subject_has_changed, new_subject, err := export.AssignPropertiesIfChanged(ctx, subject_body, subject_updates)

	if err != nil {
		return nil, fmt.Errorf("Failed to assign subject properties, %w", err)
	}

	if subject_has_changed {

		_, err = sfom_writer.WriteBytes(ctx, subject_mw, new_subject)

		if err != nil {
			return nil, fmt.Errorf("Failed to write subject update, %w", err)
		}
	}

	// END OF update the subject (parent) record

	// Close the depiction and subject writers - this is a no-op for many writer but
	// required for things like the githubapi-tree:// and githubapi-pr:// writers.

	err = depiction_mw.Close(ctx)

	if err != nil {
		return nil, fmt.Errorf("Failed to close depiction writer, %w", err)
	}

	err = subject_mw.Close(ctx)

	if err != nil {
		return nil, fmt.Errorf("Failed to close subject writer, %w", err)
	}

	// Now write the subject (object) being depicted

	local_depiction_buf_writer.Flush()
	local_subject_buf_writer.Flush()

	fc := geojson.NewFeatureCollection()

	var new_subject_b []byte

	if subject_has_changed {
		new_subject_b = local_subject_buf.Bytes()
	} else {
		new_subject_b = subject_body
	}

	new_subject_f, err := geojson.UnmarshalFeature(new_subject_b)

	if err != nil {
		return nil, fmt.Errorf("Failed to unmarshal feature from depiction buffer, %w", err)
	}

	fc.Append(new_subject_f)

	if depiction_has_changed {

		new_depiction_b := local_depiction_buf.Bytes()

		new_depiction_fc, err := geojson.UnmarshalFeatureCollection(new_depiction_b)

		if err != nil {
			return nil, fmt.Errorf("Failed to unmarshal feature from depiction buffer, %w '%s'", err, string(new_depiction_b))
		}

		for _, f := range new_depiction_fc.Features {
			fc.Append(f)
		}
	}

	fc_body, err := fc.MarshalJSON()

	if err != nil {
		return nil, fmt.Errorf("Failed to marshal feature collection, %w", err)
	}

	return fc_body, nil
}
