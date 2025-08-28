package main

import (
	"context"
	"flag"
	"io"
	"log"
	"log/slog"

	"github.com/tidwall/gjson"
	"github.com/whosonfirst/go-whosonfirst-iterate/v3"
	"github.com/whosonfirst/go-whosonfirst-feature/properties"	
)

func main() {

	var iterator_uri string

	flag.StringVar(&iterator_uri, "iterator-uri", "repo://?include=properties.georef:depiction=.*", "...")
	flag.Parse()

	ctx := context.Background()

	iter, err := iterate.NewIterator(ctx, iterator_uri)

	if err != nil {
		log.Fatalf("Failed to create iterator, %w", err)
	}

	sources := flag.Args()
	
	for rec, err := range iter.Iterate(ctx, sources...) {

		if err != nil {
			log.Fatalf("Iterator reported error, %v", err)
		}

		defer rec.Body.Close()

		logger := slog.Default()
		logger = logger.With("path", rec.Path)
		
		body, err := io.ReadAll(rec.Body)

		if err != nil {
			logger.Error("Failed to read record", "error", err)
			log.Fatal(err)
		}

		parent_id, err := properties.ParentId(body)

		if err != nil {
			logger.Error("Failed to derive parent", "error", err)
			log.Fatal(err)
		}
		
		to_update := map[string]any{
			"properties.georef:subject": parent_id,
		}
		
		to_remove := make([]string, 0)
		
		wof_refs := gjson.GetBytes(body, "properties.wof:references")

		if wof_refs.Exists(){
			to_remove = append(to_remove, "properties.wof:references")

			belongs_to := make([]int64, 0)

			for _, i := range wof_refs.Array(){
				
				id := i.Int()
				
				if id > -1 {
					belongs_to = append(belongs_to, id)
				}
			}
			
			to_update["properties:georef:whosonfirst_belongsto"] = belongs_to
		} else {

			belongs_to := make([]int64, 0)
			
			refs := gjson.GetBytes(body, "properties.georef:depictions")

			for _, r := range refs.Array(){

				for _, i := range r.Get("wof:depicts").Array(){
					
					id := i.Int()

					if id > -1 {
						belongs_to = append(belongs_to, id)
					}
				}
			}

			to_update["properties:georef:whosonfirst_belongsto"] = belongs_to
		}
		
	}
}

