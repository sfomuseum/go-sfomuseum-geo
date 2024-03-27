package alt

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

func TestFormatAltFeature(t *testing.T) {

	expected := `{
          "id": 1511949289,
          "type": "Feature",
          "properties": {
            "geotag:angle": 55.55772148835706,
            "geotag:bearing": -129.67474888275055,
            "geotag:camera_latitude": 37.61626328776611,
            "geotag:camera_longitude": -122.38344191444918,
            "geotag:distance": 875.1170559376638,
            "geotag:target_latitude": 37.61124006276177,
            "geotag:target_longitude": -122.39108634978712,
            "src:alt_label": "geotag-fov",
            "src:geom": "sfomuseum",
            "wof:id": 1511949289,
            "wof:repo": "sfomuseum-data-collection"
          },
          "geometry": {"type":"Polygon","coordinates":[[[-122.38344191444918,37.61626328776611],[-122.39442677335957,37.61442973681917],[-122.38774592621468,37.60805038870438],[-122.38344191444918,37.61626328776611]]]}
        }`

	rel_path := "../fixtures/sfomuseum-data-collection/data/151/194/928/9/1511949289-alt-geotag-fov.geojson"

	r, err := os.Open(rel_path)

	if err != nil {
		t.Fatalf("Failed to open %s, %v", rel_path, err)
	}

	defer r.Close()

	var f *WhosOnFirstAltFeature

	dec := json.NewDecoder(r)
	err = dec.Decode(&f)

	if err != nil {
		t.Fatalf("Failed to decode %s, %v", rel_path, err)
	}

	body, err := FormatAltFeature(f)

	if err != nil {
		t.Fatalf("Failed to format %s, %v", rel_path, err)
	}

	var tmp interface{}

	err = json.Unmarshal(body, &tmp)

	if err != nil {
		t.Fatalf("Failed to unmarshal formatted result for %s, %v", rel_path, err)
	}

	enc_body, err := json.Marshal(tmp)

	if err != nil {
		t.Fatalf("Failed to marshal result, %v", err)
	}

	err = json.Unmarshal([]byte(expected), &tmp)

	if err != nil {
		t.Fatalf("Failed to unmarshal expected result for %s, %v", rel_path, err)
	}

	enc_expected, err := json.Marshal(tmp)

	if err != nil {
		t.Fatalf("Failed to marshal expected result, %v", err)
	}

	if !bytes.Equal(enc_body, enc_expected) {
		t.Logf("Unexpected output: '%s' (expected: '%s')", string(enc_body), string(enc_expected))
	}

}
