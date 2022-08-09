// build-update emits a JSON-encoded `geotag.Depiction` struct to STDOUT derived from command-line flags and a "geotag"
// GeoJSON Feature read from STDIN.
//
//	$> cat fixtures/geotag/1527827539.geojson | bin/build-depiction-update -depiction-id 1527827539 -parent-id 1159396131 | jq
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	geojson "github.com/sfomuseum/go-geojson-geotag"
	geotag "github.com/sfomuseum/go-sfomuseum-geo/geotag"
	"log"
	"os"
)

func main() {

	depiction_id := flag.Int64("depiction-id", -1, "A valid Who's On First ID of the record being depicted.")
	parent_id := flag.Int64("parent-id", -1, "A valid Who's On First ID of the record \"parenting\" the record being depicted.")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "build-update emits a JSON-encoded `geotag.Depiction` struct to STDOUT derived from command-line flags and a \"geotag\" GeoJSON Feature read from STDIN.\n")
		fmt.Fprintf(os.Stderr, "Usage:\n\t %s [options]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Valid options are:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	var f *geojson.GeotagFeature

	dec := json.NewDecoder(os.Stdin)
	err := dec.Decode(&f)

	if err != nil {
		log.Fatalf("Failed to decode geotag feature, %v", err)
	}

	update := &geotag.Depiction{
		DepictionId: *depiction_id,
		ParentId:    *parent_id,
		Feature:     f,
	}

	enc := json.NewEncoder(os.Stdout)
	err = enc.Encode(update)

	if err != nil {
		log.Fatalf("Failed to encode depiction update feature, %v", err)
	}

}
