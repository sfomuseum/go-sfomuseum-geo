package flightcover

import (
	"context"
	"flag"
	"fmt"
	"github.com/sfomuseum/go-flags/flagset"
	"github.com/sfomuseum/go-flags/multi"
	"os"
)

var mode string

var depiction_reader_uri string
var depiction_writer_uri string

var subject_reader_uri string
var subject_writer_uri string

var whosonfirst_reader_uri string

var access_token_uri string

var from_id multi.MultiInt64
var to_id multi.MultiInt64
var sent_id multi.MultiInt64
var received_id multi.MultiInt64

var depictions multi.MultiInt64

func DefaultFlagSet(ctx context.Context) *flag.FlagSet {

	fs := flagset.NewFlagSet("geotag")

	fs.StringVar(&mode, "mode", "cli", "Valid options are: cli, lambda.")

	// Assumed to be something in sfomuseum-data-media-collection

	fs.StringVar(&depiction_reader_uri, "depiction-reader-uri", "repo:///usr/local/data/sfomuseum-data-media-collection", "A valid whosonfirst/go-reader URI.")
	fs.StringVar(&depiction_writer_uri, "depiction-writer-uri", "stdout://", "A valid whosonfirst/go-writer URI.")

	// Assumed to be something in sfomuseum-data-collection

	fs.StringVar(&subject_reader_uri, "subject-reader-uri", "repo:///usr/local/data/sfomuseum-data-collection", "A valid whosonfirst/go-reader URI.")
	fs.StringVar(&subject_writer_uri, "subject-writer-uri", "stdout://", "A valid whosonfirst/go-writer URI.")

	// Eventually something in the whosonfirst(.org) findingaid

	// fs.StringVar(&whosonfirst_reader_uri, "whosonfirst-reader-uri", "https://data.whosonfirst.org/", "A valid whosonfirst/go-reader URI.")
	fs.StringVar(&whosonfirst_reader_uri, "whosonfirst-reader-uri", "repo:///usr/local/data/sfomuseum-data-whosonfirst", "A valid whosonfirst/go-reader URI.")

	fs.StringVar(&access_token_uri, "access-token", "", "A valid gocloud.dev/runtimevar URI")

	fs.Var(&from_id, "from", "One or more valid Who's On First IDs representing the place a (flight cover) letter was sent from.")
	fs.Var(&to_id, "to", "One or more valid Who's On First IDs representing the place a (flight cover) letter was sent to.")
	fs.Var(&received_id, "received", "One or more valid Who's On First IDs representing the place a (flight cover) letter was postmarked as received.")
	fs.Var(&sent_id, "sent", "One or more valid Who's On First IDs representing the place a (flight cover) letter was postmarked as sent.")

	fs.Var(&depictions, "depiction-id", "One or more valid Who's On First IDs for the records being depicted.")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "...\n")
		fmt.Fprintf(os.Stderr, "Usage:\n\t %s [options]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Valid options are:\n")
		fs.PrintDefaults()
	}

	return fs
}
