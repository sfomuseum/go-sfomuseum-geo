package geotag

import (
	"encoding/json"
	geojson "github.com/sfomuseum/go-geojson-geotag/v2"
	"os"
	"testing"
)

func TestGeotagDepiction(t *testing.T) {

	rel_path := "../fixtures/geotag/1527827539.geojson"

	r, err := os.Open(rel_path)

	if err != nil {
		t.Fatalf("Failed to open %s, %v", rel_path, err)
	}

	defer r.Close()

	var f *geojson.GeotagFeature

	dec := json.NewDecoder(r)
	err = dec.Decode(&f)

	if err != nil {
		t.Fatalf("Failed to decode %s, %v", rel_path, err)
	}

	d := Depiction{
		DepictionId: -1,
		ParentId:    -1,
		Feature:     f,
	}

	enc_d, err := json.Marshal(d)

	if err != nil {
		t.Fatalf("Failed to marshal depiction, %v", err)
	}

	err = json.Unmarshal(enc_d, &d)

	if err != nil {
		t.Fatalf("Failed to unmarshal depiction, %v", err)
	}
}
