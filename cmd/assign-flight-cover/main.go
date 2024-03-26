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

	"github.com/sfomuseum/go-sfomuseum-geo/app/georeference/flightcover"	
)

func main() {

	ctx := context.Background()

	err := flightcover.Run(ctx)

	if err != nil {
		log.Fatalf("Failed to assign references, %v", err)
	}
}
