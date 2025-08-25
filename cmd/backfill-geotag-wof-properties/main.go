package main

import (
	"context"
	"flag"
	"io"
	"log"
	"log/slog"
	"slices"

	"github.com/tidwall/gjson"
	"github.com/whosonfirst/go-whosonfirst-export/v3"
	"github.com/whosonfirst/go-whosonfirst-iterate/v3"
	wof_writer "github.com/whosonfirst/go-whosonfirst-writer/v3"
	"github.com/whosonfirst/go-writer/v3"
)

func main() {

	var iterator_uri string
	var writer_uri string

	flag.StringVar(&iterator_uri, "iterator-uri", "repo://?include=properties.geotag:depictions=.*", "A registered whosonfirst/go-whosonfirst-iterate/v3.Iterator URI.")
	flag.StringVar(&writer_uri, "writer-uri", "stdout://", "A registered whosnfirst/go-whosonfirst-writer/v3.Writer URI.")

	flag.Parse()

	ctx := context.Background()

	iter, err := iterate.NewIterator(ctx, iterator_uri)

	if err != nil {
		log.Fatalf("Failed to create iterator, %v", err)
	}

	wr, err := writer.NewWriter(ctx, writer_uri)

	if err != nil {
		log.Fatalf("Failed to create writer, %v", err)
	}

	sources := flag.Args()

	path_camera := "properties.geotag:camera_whosonfirst"
	path_target := "properties.geotag:target_whosonfirst"
	// path_depictions := "properties.geotag:depictions"

	for rec, err := range iter.Iterate(ctx, sources...) {

		if err != nil {
			log.Fatalf("Iterator yield an error, %v", err)
		}

		defer rec.Body.Close()

		logger := slog.Default()
		logger = logger.With("path", rec.Path)

		logger.Info("Process record")

		body, err := io.ReadAll(rec.Body)

		if err != nil {
			logger.Error("Failed to read body", "error", err)
			log.Fatal(err)
		}

		pt := gjson.GetBytes(body, "properties.sfomuseum:placetype").String()

		to_update := make(map[string]any)
		to_remove := make([]string, 0)

		wof_depicts := make([]int64, 0)

		// geotag:camera_whosonfirst
		// geotag:target_whosonfirst

		foo := []string{
			path_camera,
			path_target,
		}

		for _, path := range foo {

			rsp := gjson.GetBytes(body, path)

			if !rsp.Exists() {
				continue
			}

			to_remove = append(to_remove, path)

			id_rsp := rsp.Get("wof:id")
			hier_rsp := rsp.Get("wof:hierarchy")

			if id_rsp.Exists() {
				id := id_rsp.Int()

				switch path {
				case path_camera:
					to_update["properties.geotag:whosonfirst_camera"] = id
				case path_target:
					to_update["properties.geotag:whosonfirst_target"] = id
				default:
					// pass
				}

				wof_depicts = append(wof_depicts, id)
			}

			if hier_rsp.Exists() {

				for _, h := range hier_rsp.Array() {

					for _, i := range h.Map() {

						id := i.Int()

						if !slices.Contains(wof_depicts, id) {
							wof_depicts = append(wof_depicts, id)
						}
					}
				}
			}
		}

		// geotag:depictions (if object)

		if pt == "object" {
			//to_remove = append(to_remove, "properties.geotag:depictions")
		}

		if len(wof_depicts) > 0 {
			to_update["properties.geotag:whosonfirst_depicts"] = wof_depicts
		}

		new_body, err := export.AssignProperties(ctx, body, to_update)

		if err != nil {
			logger.Error("Failed to assign update properties", "error", err)
			log.Fatal(err)
		}

		new_body, err = export.RemoveProperties(ctx, new_body, to_remove)

		if err != nil {
			logger.Error("Failed to remove properties", "error", err)
			log.Fatal(err)
		}

		_, err = wof_writer.WriteBytes(ctx, wr, new_body)

		if err != nil {
			logger.Error("Failed to write changes", "error", err)
			log.Fatal(err)
		}
	}

}
