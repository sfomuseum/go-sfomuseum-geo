package georeference

// Eventually it would be good to abstract out all of the SFO Museum stuff from this
// but not today...

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
	"github.com/sfomuseum/go-sfomuseum-geo"
	"github.com/sfomuseum/go-sfomuseum-geo/alt"
	// "github.com/sfomuseum/go-sfomuseum-geo/geometry"
	"github.com/sfomuseum/go-sfomuseum-geo/github"
	geo_writers "github.com/sfomuseum/go-sfomuseum-geo/writers"
	// "github.com/tidwall/gjson"
	"github.com/whosonfirst/go-reader/v2"
	"github.com/whosonfirst/go-whosonfirst-export/v3"
	"github.com/whosonfirst/go-whosonfirst-feature/properties"
	wof_reader "github.com/whosonfirst/go-whosonfirst-reader/v2"
	"github.com/whosonfirst/go-whosonfirst-uri"
	wof_writer "github.com/whosonfirst/go-whosonfirst-writer/v3"
)

// AssignReferencesOptions defines a struct for reading/writing options when updating geo-related information in depictions.
// A depiction is assumed to be the record for an image or some other piece of media. A subject is assumed to be
// the record for an object.
type AssignReferencesOptions struct {
	// A valid whosonfirst/go-reader.Reader instance for reading depiction features. A depiction might be an image of a collection object.
	DepictionReader reader.Reader
	// A valid whosonfirst/go-reader.Reader instance for reading subject features. A subject might be a collection object (rather than any one image (depiction) of that objec)
	SubjectReader reader.Reader
	// A valid whosonfirst/go-reader.Reader instance for reading "parent" features.
	// This is the equivalent to ../geotag.GeotagDepictionOptions.ParentReader and should be reconciled one way or the other.
	WhosOnFirstReader reader.Reader
	// A valid whosonfirst/go-reader.Reader instance for reading a "default geometry" features.
	DefaultGeometryFeatureId int64
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
	logger = logger.With("action", "assign georeferences")
	logger = logger.With("depiction id", depiction_id)

	if len(refs) == 0 {
		logger.Warn("No references to assign. This will remove all previous references")
	}

	src_geom := "sfomuseum#georeference"

	if opts.SourceGeomSuffix != "" {
		src_geom = fmt.Sprintf("%s-%s", src_geom, opts.SourceGeomSuffix)
		logger.Debug("Automatically assign source geom suffix", "suffix", src_geom)
	}

	logger.Debug("Set up writers")

	github_opts := &github.UpdateWriterURIOptions{
		Author:        opts.Author,
		WhosOnFirstId: depiction_id,
		Action:        github.GeoreferenceAction,
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
		return nil, fmt.Errorf("Failed to create depiction reader, %w", err)
	}

	logger.Debug("Derive repo for depiction")

	depiction_repo, err := properties.Repo(depiction_body)

	if err != nil {
		return nil, fmt.Errorf("Unable to derive wof:repo for depiction %d, %w", depiction_id, err)
	}

	logger.Debug("Derive parent (subject) for depiction")

	subject_id, err := properties.ParentId(depiction_body)

	if err != nil {
		return nil, fmt.Errorf("Failed to derive subject (parent) ID for depiction, %w", err)
	}

	logger = logger.With("subject id", subject_id)

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

	// Ensure unique reference labels. This is mostly to ensure that any alt files
	// which need to be created are unique. This decision may need to be revisted
	// or finessed in the future. We'll see.
	labels_map := new(sync.Map)

	for _, r := range refs {

		_, exists := labels_map.LoadOrStore(r.Label, true)

		if exists {
			return nil, fmt.Errorf("Multiple references with duplicate label, '%s'", r.Label)
		}
	}

	// Start iterating references to assign

	for _, r := range refs {

		go func(ctx context.Context, r *Reference) {

			logger.Info("Process reference", "ref", r.Label, "ids", r.Ids, "alt", r.AltLabel)

			defer func() {
				done_ch <- true
			}()

			if len(r.Ids) == 0 {
				logger.Error("Ref is missing ids")
				err_ch <- fmt.Errorf("Ref is missing IDs")
				return
			}

			if r.Label == "" {
				logger.Error("Ref is missing label")
				err_ch <- fmt.Errorf("Ref is missing label")
				return
			}

			prop_label := r.Label
			alt_label := DeriveAltLabelFromReference(r)

			// Note we are only assigning the base path for this key (prop_label)
			// updates_map is "range-ed" below and we build a new new_depicted
			// dict which is then assigned to properties.{geo.RESERVED_GEOREFERENCE_DEPICTED}

			logger.Debug("Store in updates map", "label", prop_label, "ids", r.Ids)
			updates_map.Store(prop_label, r.Ids)

			count := len(r.Ids)
			points := make([]orb.Point, count)

			// Remember any given reference (label) can have mutiple WOF IDs
			// Fetch centroid and hierarchy for each ID in a reference

			for idx, id := range r.Ids {

				logger := slog.Default()
				logger = logger.With("depection id", depiction_id)
				logger = logger.With("ref id", id)
				logger = logger.With("label", prop_label)
				logger = logger.With("alt_label", alt_label)

				logger.Debug("Process reference")

				body, err := wof_reader.LoadBytes(ctx, opts.WhosOnFirstReader, id)

				if err != nil {
					logger.Error("Failed to load record for reference", "error", err)
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

			logger.Debug("Return new alt feature")
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

	// START OF assign/update georef:whosonfirst_belongsto for depiction

	georef_belongsto := make([]int64, 0)

	for _, r := range refs {
		for _, i := range r.Ids {
			references_map.Store(i, true)
		}
	}

	references_map.Range(func(k interface{}, v interface{}) bool {
		id := k.(int64)
		georef_belongsto = append(georef_belongsto, id)
		return true
	})

	// logger.Debug("Georef_belongsto for depiction", "count", len(georef_belongsto))

	depiction_updates[fmt.Sprintf("properties.%s", geo.RESERVED_GEOREFERENCE_BELONGSTO)] = georef_belongsto

	// END OF assign/update georef:whosonfirst_belongsto for depictionx

	// START OF assign/update georeference:depictions here

	new_depicted := make([]map[string]any, 0)

	updates_map.Range(func(k interface{}, v interface{}) bool {

		label := k.(string)
		ids := v.([]int64)

		d := map[string]any{
			geo.RESERVED_GEOREFERENCE_LABEL: label,
			geo.RESERVED_WOF_DEPICTS:        ids,
		}

		new_depicted = append(new_depicted, d)
		return true
	})

	depiction_updates[fmt.Sprintf("properties.%s", geo.RESERVED_GEOREFERENCE_DEPICTED)] = new_depicted

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

		// This should never really happen...
		if len(r.Ids) == 0 {
			alt_label := DeriveAltLabelFromReference(r)
			to_remove[alt_label] = r
		}
	}

	// Now build the list of features (used to build alt files) to fetch
	// Note how we are skipping features to remove

	to_fetch := make([]string, 0)

	for _, label := range existing_alt {

		_, ok_lookup := lookup[label]
		_, ok_remove := to_remove[label]

		logger.Debug("Compare existing alt file", "label", label, "ok lookup", ok_lookup, "ok remove", ok_remove)

		if len(refs) == 0 && strings.HasPrefix(label, GEOREF_ALT_PREFIX) {
			logger.Debug("Refs count is 0 and (alt) label is georef, flag alt file for removal", "label", label)
			ok_remove = true
		}

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

	logger.Debug("Compile new and existing alt features", "new", len(new_alt_features), "other", len(other_alt_features))

	alt_features := make([]*alt.WhosOnFirstAltFeature, 0)

	for _, f := range new_alt_features {
		alt_features = append(alt_features, f)
	}

	for _, f := range other_alt_features {
		alt_features = append(alt_features, f)
	}

	// Use this new list to catalog alt geoms and derived a multipoint geometry

	logger.Debug("Calculate multipoint geometry for alt geoms", "count", len(alt_features))

	alt_geoms := make([]string, len(alt_features))

	for idx, f := range alt_features {
		label := f.Properties["src:alt_label"].(string)
		logger.Debug("Assign alt label", "label", label)
		alt_geoms[idx] = label
	}

	depiction_updates["properties.src:geom_alt"] = alt_geoms

	// Derive geometry for depiction. This is either a MultiPoint geometry
	// of all the (not-deprecated) alt files OR the geometry of the "default" feature

	if len(alt_features) == 0 {

		logger.Debug("No alt features, assign geometry and hierarchies from default geometry record", "id", opts.DefaultGeometryFeatureId)

		body, err := wof_reader.LoadBytes(ctx, opts.SFOMuseumReader, opts.DefaultGeometryFeatureId)

		if err != nil {
			logger.Error("Failed to read default geometry record", "id", opts.DefaultGeometryFeatureId, "error", err)
			return nil, fmt.Errorf("Failed to read default geometry record, %w", err)
		}

		centroid, _, err := properties.Centroid(body)

		if err != nil {
			logger.Error("Failed to derive centroid for default geometry record", "id", opts.DefaultGeometryFeatureId, "error", err)
			return nil, fmt.Errorf("Failed to unmarshal default geometry record, %w", err)
		}

		depiction_updates["geometry"] = geojson.NewGeometry(centroid)

		// hierarchies are actually assigned below

		default_h := properties.Hierarchies(body)
		enc_h, err := json.Marshal(default_h)

		if err != nil {
			logger.Error("Failed to marshal hierarchy", "error", err)
			return nil, fmt.Errorf("Failed to marshal hierarchy for default feature, %w", err)
		}

		md5_h := fmt.Sprintf("%x", md5.Sum(enc_h))
		hierarchies_hash_map.Store(md5_h, default_h)

	} else {

		mp_geom, err := alt.DeriveMultiPointGeometry(ctx, alt_features...)

		if err != nil {
			logger.Error("Failed to derive multi point geometry from alt files", "error", err)
			return nil, fmt.Errorf("Failed to derive multi point geometry, %w", err)
		}

		mp_geojson_geom := geojson.NewGeometry(mp_geom)
		depiction_updates["geometry"] = mp_geojson_geom
	}

	// Now save the new alt files

	logger.Debug("Save new alt files", "count", len(new_alt_features))

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

		_, err = writers.DepictionWriter.Write(ctx, alt_uri, r)

		if err != nil {
			return nil, fmt.Errorf("Failed to write new alt feature %s, %w", alt_uri, err)
		}
	}

	// Now rewrite alt files that need to be "removed"

	logger.Debug("Rewrite alt files to \"remove\" (deprecate)", "count", len(to_remove))

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

		logger.Debug("Deprecate alt file", "label", alt_uri)

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

		_, err = writers.DepictionMultiWriter.Write(ctx, alt_uri, r)

		if err != nil {
			return nil, fmt.Errorf("Failed to write deprecate alt feature %s, %w", alt_uri, err)
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
		logger.Error("Failed to assign depiction properties", "error", err)
		return nil, fmt.Errorf("Failed to assign depiction properties, %w", err)
	}

	// Write changes

	logger.Debug("Has depiction changed", "changes", depiction_has_changed)

	if depiction_has_changed {

		lastmod_key := fmt.Sprintf("properties.%s", geo.RESERVED_GEOREFERENCE_LASTMODIFIED)
		lastmod := time.Now()

		lastmod_updates := map[string]any{
			lastmod_key: lastmod.Unix(),
		}

		new_body, err := export.AssignProperties(ctx, new_body, lastmod_updates)

		if err != nil {
			logger.Error("Failed to assign last mod properties for subject record", "error", err)
			return nil, fmt.Errorf("Failed to assign last mod properties for subject record, %w", err)
		}

		_, err = wof_writer.WriteBytes(ctx, writers.DepictionMultiWriter, new_body)

		if err != nil {
			logger.Error("Failed to write depiction", "error", err)
			return nil, fmt.Errorf("Failed to write depiction update, %w", err)
		}
	}

	// END OF update the depiction record

	logger.Debug("Finished updating depiction")
	logger.Debug("Start updating subject")

	// START OF update the subject (parent) record

	subject_body, err := wof_reader.LoadBytes(ctx, opts.SubjectReader, subject_id)

	if err != nil {
		return nil, fmt.Errorf("Failed to load subject (parent) for depiction, %w", err)
	}

	/*
		subject_hierarchies := make([]map[string]int64, 0)

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
	*/

	// START OF denormalize all the georeferenced properties from all the images (depictions) in to the object record

	recompile_opts := &RecompileGeorefencesForSubjectOptions{
		DepictionReader: depiction_reader,
		SFOMuseumReader: opts.WhosOnFirstReader,
		SkipList: map[int64]*SkipListItem{
			depiction_id: &SkipListItem{
				// Please make this better...
				// Geometry: depiction_updates["geometry"].(*geojson.Geometry).Geometry(),
				Depicted: new_depicted,
			},
		},
	}

	subject_has_changed, subject_body, err := RecompileGeorefencesForSubject(ctx, recompile_opts, subject_body)

	if err != nil {
		logger.Error("Failed to recompile georeferences for subject", "error", err)
		return nil, fmt.Errorf("Failed to recompile georeferences for subject, %w", err)
	}

	if subject_has_changed {

		_, err = wof_writer.WriteBytes(ctx, writers.SubjectMultiWriter, subject_body)

		if err != nil {
			logger.Error("Failed to write subject record", "error", err)
			return nil, fmt.Errorf("Failed to write subject update, %w", err)
		}
	}

	// END OF update the subject (parent) record

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

	// Now write the subject (object) being depicted

	fc, err := writers.AsFeatureCollection()

	if err != nil {
		return nil, err
	}

	fc_body, err := fc.MarshalJSON()

	if err != nil {
		return nil, fmt.Errorf("Failed to marshal feature collection, %w", err)
	}

	return fc_body, nil
}
