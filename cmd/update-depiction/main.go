// update-depiction is a command-tool for applying geotagging updates to one or more depictions. For example:
//
//	$> cat fixtures/geotag/1527827539.geojson | bin/update-depiction \
//		-depiction-id 1527827539 -parent-id 1159396131 \
//		-depiction-reader-uri repo:///usr/local/sfomuseum/go-sfomuseum-geotag/fixtures/sfomuseum-data-media-collection \
//		-subject-reader-uri repo:///usr/local/sfomuseum/go-sfomuseum-geotag/fixtures/sfomuseum-data-collection \
//		-parent-reader-uri repo:///usr/local/sfomuseum/go-sfomuseum-geotag/fixtures/sfomuseum-data-architecture
//
// Or, reading from the default remote endpoints and updating local data:
//
//	$> cat fixtures/geotag/1527827539.geojson | bin/update-depiction \
//		-depiction-id 1527827539 -parent-id 1159396131 \
//		-depiction-writer-uri repo:///usr/local/data/sfomuseum-data-media-collection \
//		-subject-writer-uri repo:///usr/local/data/sfomuseum-data-collection
//
// Or, reading and writing to the default remote endpoints:
//
//	$> cat fixtures/geotag/1527827539.geojson | bin/update-depiction \
//		-depiction-id 1527827539 -parent-id 1159396131 \
//		-access-token 'file:///usr/local/sfomuseum/lockedbox/github/geotag'
//
// This tool may still change. Specifically rather than reading depiction and parent parameters from flags and "geotag"
// records from STDIN, it may be updated to expect one or more JSON-encoded `geotag.Depiction` records as series of paths
// or read from STDIN in concert with a `-stdin` flag. This would allow the tool to be run as a Lambda function reading
// files from S3 (or any valid gocloud.dev/blob source).
package main

import (
	_ "github.com/whosonfirst/go-reader-findingaid"
	_ "github.com/whosonfirst/go-reader-github"
	_ "gocloud.dev/runtimevar/awsparamstore"
	_ "gocloud.dev/runtimevar/constantvar"
	_ "gocloud.dev/runtimevar/filevar"
)

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/sfomuseum/go-flags/flagset"
	"github.com/sfomuseum/go-flags/multi"
	geojson "github.com/sfomuseum/go-geojson-geotag"
	"github.com/sfomuseum/go-sfomuseum-geo/geotag"
	"github.com/whosonfirst/go-reader"
	gh_writer "github.com/whosonfirst/go-writer-github/v2"
	"github.com/whosonfirst/go-writer/v2"
	"log"
	"os"
)

func main() {

	fs := flagset.NewFlagSet("geotag")

	img_reader_uri := fs.String("depiction-reader-uri", "", "A valid whosonfirst/go-reader URI.")
	img_writer_uri := fs.String("depiction-writer-uri", "", "A valid whosonfirst/go-writer URI.")

	obj_reader_uri := fs.String("subject-reader-uri", "", "A valid whosonfirst/go-reader URI.")
	obj_writer_uri := fs.String("subject-writer-uri", "", "A valid whosonfirst/go-writer URI.")

	parent_reader_uri := fs.String("parent-reader-uri", "", "A valid whosonfirst/go-reader URI.")

	access_token_uri := fs.String("access-token", "", "A valid gocloud.dev/runtimevar URI")

	parent_id := fs.Int64("parent-id", -1, "A valid Who's On First ID of the record \"parenting\" the records being depicted.")

	var depictions multi.MultiInt64

	fs.Var(&depictions, "depiction-id", "One or more valid Who's On First IDs for the records being depicted.")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "update-depiction is a command-tool for applying geotagging updates to one or more depictions.\n")
		fmt.Fprintf(os.Stderr, "Usage:\n\t %s [options]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Valid options are:\n")
		fs.PrintDefaults()
	}

	flagset.Parse(fs)

	ctx := context.Background()

	var err error

	*img_writer_uri, err = gh_writer.EnsureGitHubAccessToken(ctx, *img_writer_uri, *access_token_uri)

	if err != nil {
		log.Fatalf("Failed to ensure access token for depiction writer URI, %v", err)
	}

	*obj_writer_uri, err = gh_writer.EnsureGitHubAccessToken(ctx, *obj_writer_uri, *access_token_uri)

	if err != nil {
		log.Fatalf("Failed to ensure access token for subject writer URI, %v", err)
	}

	img_reader, err := reader.NewReader(ctx, *img_reader_uri)

	if err != nil {
		log.Fatalf("Failed to create depiction reader, %v", err)
	}

	img_writer, err := writer.NewWriter(ctx, *img_writer_uri)

	if err != nil {
		log.Fatalf("Failed to create depiction writer, %v", err)
	}

	obj_reader, err := reader.NewReader(ctx, *obj_reader_uri)

	if err != nil {
		log.Fatalf("Failed to create subject reader, %v", err)
	}

	obj_writer, err := writer.NewWriter(ctx, *obj_writer_uri)

	if err != nil {
		log.Fatalf("Failed to create subject writer, %v", err)
	}

	parent_reader, err := reader.NewReader(ctx, *parent_reader_uri)

	if err != nil {
		log.Fatalf("Failed to create architecture reader, %v", err)
	}

	var f *geojson.GeotagFeature

	dec := json.NewDecoder(os.Stdin)
	err = dec.Decode(&f)

	if err != nil {
		log.Fatalf("Failed to decode geotag feature, %v", err)
	}

	opts := &geotag.UpdateDepictionOptions{
		DepictionReader:    img_reader,
		DepictionWriter:    img_writer,
		DecpitionWriterURI: *img_writer_uri, // to be remove post writer/v3 (Clone) release
		SubjectReader:      obj_reader,
		SubjectWriter:      obj_writer,
		SubjectWriterURI:   *obj_writer_uri, // to be remove post writer/v3 (Clone) release
		ParentReader:       parent_reader,
	}

	for _, depiction_id := range depictions {

		update := &geotag.Depiction{
			DepictionId: depiction_id,
			ParentId:    *parent_id,
			Feature:     f,
		}

		_, err := geotag.UpdateDepiction(ctx, opts, update)

		if err != nil {
			log.Fatalf("Failed to update depiction %d, %v", depiction_id, err)
		}
	}
}
