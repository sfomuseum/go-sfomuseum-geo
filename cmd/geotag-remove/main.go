package main

import (
	"context"
	"log"

	_ "github.com/whosonfirst/go-reader-findingaid/v2"
	_ "github.com/whosonfirst/go-reader-github/v2"
	_ "gocloud.dev/runtimevar/awsparamstore"
	_ "gocloud.dev/runtimevar/constantvar"
	_ "gocloud.dev/runtimevar/filevar"

	"github.com/sfomuseum/go-sfomuseum-geo/app/geotag/remove"
)

func main() {

	ctx := context.Background()

	err := remove.Run(ctx)

	if err != nil {
		log.Fatalf("Failed to remove depiction, %v", err)
	}
}
