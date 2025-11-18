package recompile

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/sfomuseum/go-flags/flagset"
	"github.com/sfomuseum/go-flags/multi"
)

var verbose bool

var depiction_reader_uri string
var whosonfirst_reader_uri string
var sfomuseum_reader_uri string

var subject_reader_uri string
var subject_writer_uri string

var access_token_uri string
var subject_ids multi.MultiInt64

var default_geometry_feature_id int64

var iterator_uri string

func DefaultFlagSet(ctx context.Context) *flag.FlagSet {

	fs := flagset.NewFlagSet("reference")
	fs.BoolVar(&verbose, "verbose", false, "Enable verbose (debug) logging.")

	// Assumed to be something in sfomuseum-data-media-collection

	fs.StringVar(&depiction_reader_uri, "depiction-reader-uri", "repo:///usr/local/data/sfomuseum-data-media-collection", "A valid whosonfirst/go-reader URI.")
	fs.StringVar(&whosonfirst_reader_uri, "whosonfirst-reader-uri", "https://static.sfomuseum.org/geojson/", "A valid whosonfirst/go-reader URI.")
	fs.StringVar(&sfomuseum_reader_uri, "sfomuseum-reader-uri", "https://static.sfomuseum.org/geojson/", "A valid whosonfirst/go-reader URI.")

	fs.StringVar(&subject_reader_uri, "subject-reader-uri", "repo:///usr/local/data/sfomuseum-data-collection", "A valid whosonfirst/go-reader URI.")
	fs.StringVar(&subject_writer_uri, "subject-writer-uri", "repo:///usr/local/data/sfomuseum-data-collection", "A valid whosonfirst/go-writer URI.")

	fs.StringVar(&access_token_uri, "access-token", "", "A valid gocloud.dev/runtimevar URI")

	fs.Int64Var(&default_geometry_feature_id, "default-geometry-feature-id", 1729828959, "The WOF ID for the Feature whose centroid will be used as a default absent any references.")
	fs.Var(&subject_ids, "subject-id", "One or more subject (object) IDs to recompile georeference data for.")

	fs.StringVar(&iterator_uri, "iterator-uri", "repo://?include=properties.georef:depicted=.*", "A valid whosonfirst/go-whosonfirst-iterate/v3.Iterator URI used to derive records whose georeference data should be recompiled.")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "...\n")
		fmt.Fprintf(os.Stderr, "Usage:\n\t %s [options]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Valid options are:\n")
		fs.PrintDefaults()
	}

	return fs
}
