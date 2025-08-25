package main

import (
	"context"
	"flag"
	"log"

	"github.com/whosonfirst/go-whosonfirst-iterate/v3"
)

func main() {

	var iterator_uri string

	flag.StringVar(&iterator_uri, "iterator-uri", "repo://?include=properties.geotag:depictions=.*", "A registered whosonfirst/go-whosonfirst-iterate/v3.Iterator URI.")

	flag.Parse()

	ctx := context.Background()

	iter, err := iterate.NewIterator(ctx, iterator_uri)

	if err != nil {
		log.Fatalf("Failed to create iterator, %v", err)
	}

	sources := flag.Args()

	for rec, err := range iter.Iterate(ctx, sources...) {

		if err != nil {
			log.Fatalf("Iterator yield an error, %v", err)
		}

		defer rec.Body.Close()

		log.Printf("Process %s\n", rec.Path)
	}

}
