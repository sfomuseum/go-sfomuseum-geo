package remove

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/sfomuseum/go-flags/flagset"
	"github.com/sfomuseum/go-flags/multi"
)

var mode string

var depiction_reader_uri string
var depiction_writer_uri string

var subject_reader_uri string
var subject_writer_uri string

var wof_reader_uri string

var default_geometry_id int64

var access_token_uri string

var depictions multi.MultiInt64
var verbose bool

func DefaultFlagSet(ctx context.Context) *flag.FlagSet {

	fs := flagset.NewFlagSet("geotag")

	fs.StringVar(&mode, "mode", "cli", "Valid options are: cli, lambda.")

	fs.StringVar(&depiction_reader_uri, "depiction-reader-uri", "", "A valid whosonfirst/go-reader URI.")
	fs.StringVar(&depiction_writer_uri, "depiction-writer-uri", "", "A valid whosonfirst/go-writer URI.")

	fs.StringVar(&subject_reader_uri, "subject-reader-uri", "", "A valid whosonfirst/go-reader URI.")
	fs.StringVar(&subject_writer_uri, "subject-writer-uri", "", "A valid whosonfirst/go-writer URI.")

	fs.StringVar(&wof_reader_uri, "whosonfirst-reader-uri", "", "A valid whosonfirst/go-reader URI.")

	fs.StringVar(&access_token_uri, "access-token", "", "A valid gocloud.dev/runtimevar URI")

	fs.Var(&depictions, "depiction-id", "One or more valid Who's On First IDs for the records being depicted.")

	fs.Int64Var(&default_geometry_id, "default-geometry-id", 1, "A valid Who's On First (or equivalent) ID whose geometry will be used as default geometry for records, if necessary")

	fs.BoolVar(&verbose, "verbose", false, "Enable verbose (debug) logging.")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "update-depiction is a command-tool for applying geotagging updates to one or more depictions.\n")
		fmt.Fprintf(os.Stderr, "Usage:\n\t %s [options]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Valid options are:\n")
		fs.PrintDefaults()
	}

	return fs
}
