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
	"log"

	"github.com/sfomuseum/go-sfomuseum-geo/app/geotag/update"
)

func main() {

	ctx := context.Background()

	err := update.Run(ctx)

	if err != nil {
		log.Fatalf("Failed to update depiction, %v", err)
	}
}
