package geotag

import (
	"context"
	"fmt"
	geojson "github.com/sfomuseum/go-geojson-geotag"
	"github.com/whosonfirst/go-reader"
	"github.com/whosonfirst/go-writer/v3"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestUpdateDepiction(t *testing.T) {

	depiction_id := int64(1527827539) // negative: San Francisco International Airport (SFO), Pancake Palace restaurant
	parent_id := int64(1159396131)    // Central Terminal (1954~ to 1963~)

	ctx := context.Background()

	path_fixtures, err := filepath.Abs("../fixtures")

	if err != nil {
		t.Fatalf("Failed to derive absolute path, %v", err)
	}

	geotag_name := fmt.Sprintf("geotag/%d.geojson", depiction_id)
	geotag_path := filepath.Join(path_fixtures, geotag_name)

	geotag_fh, err := os.Open(geotag_path)

	if err != nil {
		t.Fatalf("Failed to open %s, %v", geotag_path, err)
	}

	defer geotag_fh.Close()

	geotag_body, err := io.ReadAll(geotag_fh)

	if err != nil {
		t.Fatalf("Failed to read %s, %v", geotag_path, err)
	}

	f, err := geojson.NewGeotagFeature(geotag_body)

	if err != nil {
		t.Fatalf("Failed to create geotag feature from %s, %v", geotag_path, err)
	}

	update := &Depiction{
		DepictionId: depiction_id,
		ParentId:    parent_id,
		Feature:     f,
	}

	img_reader_uri := fmt.Sprintf("repo://%s/sfomuseum-data-media-collection", path_fixtures)
	img_writer_uri := "stdout://"

	obj_reader_uri := fmt.Sprintf("repo://%s/sfomuseum-data-collection", path_fixtures)
	obj_writer_uri := "stdout://"

	arch_reader_uri := fmt.Sprintf("repo://%s/sfomuseum-data-architecture", path_fixtures)

	img_reader, err := reader.NewReader(ctx, img_reader_uri)

	if err != nil {
		t.Fatalf("Failed to create depiction reader, %v", err)
	}

	img_writer, err := writer.NewWriter(ctx, img_writer_uri)

	if err != nil {
		t.Fatalf("Failed to create depiction writer, %v", err)
	}

	obj_reader, err := reader.NewReader(ctx, obj_reader_uri)

	if err != nil {
		t.Fatalf("Failed to create subject reader, %v", err)
	}

	obj_writer, err := writer.NewWriter(ctx, obj_writer_uri)

	if err != nil {
		t.Fatalf("Failed to create subject writer, %v", err)
	}

	arch_reader, err := reader.NewReader(ctx, arch_reader_uri)

	if err != nil {
		t.Fatalf("Failed to create architecture reader, %v", err)
	}

	opts := &UpdateDepictionOptions{
		DepictionReader:    img_reader,
		DepictionWriter:    img_writer,
		SubjectReader:      obj_reader,
		SubjectWriter:      obj_writer,
		ParentReader:       arch_reader,
		DepictionWriterURI: img_writer_uri,
		SubjectWriterURI:   obj_writer_uri,
	}

	_, err = UpdateDepiction(ctx, opts, update)

	if err != nil {
		t.Fatalf("Failed to update depiction, %v", err)
	}
}
