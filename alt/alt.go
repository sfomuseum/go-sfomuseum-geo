// Package alt provides methods for formatting Who's On First alternate geometry files.
package alt

import (
	"context"
	"fmt"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
	"github.com/paulmach/orb/planar"
	"github.com/sfomuseum/go-sfomuseum-geo/geometry"
	"github.com/tidwall/sjson"
	"github.com/whosonfirst/go-whosonfirst-format"
)

// WhosOnFirstAltFeature is a struct defining a GeoJSON Feature for alternate geometries.
type WhosOnFirstAltFeature struct {
	Id         int64                  `json:"id"`
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties"`
	Geometry   *geojson.Geometry      `json:"geometry"`
}

// FormatAltFeature formats 'f' and removes any top-level `bbox` properties.
func FormatAltFeature(f *WhosOnFirstAltFeature) ([]byte, error) {

	// please standardize on a common whosonfirst geojson/feature package
	// (20200413/thisisaaronland)

	ff := &format.Feature{
		Type:       f.Type,
		ID:         f.Id,
		Properties: f.Properties,
		Geometry:   f.Geometry,
	}

	body, err := format.FormatFeature(ff)

	if err != nil {
		return nil, fmt.Errorf("Failed to format feature, %w", err)
	}

	// TODO: Add omitempty hooks for bbox in go-whosonfirst-feature

	body, err = sjson.DeleteBytes(body, "bbox")

	if err != nil {
		return nil, fmt.Errorf("Failed to delete bbox property, %w", err)
	}

	return body, nil
}

func DeriveMultiPointGeometry(ctx context.Context, features ...*WhosOnFirstAltFeature) (orb.MultiPoint, error) {

	points := make([]orb.Point, 0)

	for _, f := range features {

		geojson_geom := f.Geometry
		orb_geom := geojson_geom.Geometry()

		switch orb_geom.GeoJSONType() {
		case "Point":
			pt, _ := orb_geom.(orb.Point)
			points = geometry.AddPointIfNotExist(points, pt)
		case "MultiPoint":

			for _, pt := range orb_geom.(orb.MultiPoint) {
				points = geometry.AddPointIfNotExist(points, pt)
			}

		default:
			pt, _ := planar.CentroidArea(orb_geom)
			points = geometry.AddPointIfNotExist(points, pt)
		}
	}

	return orb.MultiPoint(points), nil
}

func multipointsFromGeometry(ctx context.Context, geom orb.Geometry) (orb.MultiPoint, error) {

	switch geom.GeoJSONType() {
	case "Point":

		points := []orb.Point{
			geom.(orb.Point),
		}

		return orb.MultiPoint(points), nil

	case "MultiPoint":
		return geom.(orb.MultiPoint), nil
	case "MultiPolygon":

		points := make([]orb.Point, 0)

		for _, poly := range geom.(orb.MultiPolygon) {
			pt, _ := planar.CentroidArea(poly)
			points = geometry.AddPointIfNotExist(points, pt)
		}

		return orb.MultiPoint(points), nil

	case "Polygon":

		pt, _ := planar.CentroidArea(geom)
		points := []orb.Point{pt}

		return orb.MultiPoint(points), nil

	default:
		return nil, fmt.Errorf("Weirdo geometry type, %s", geom.GeoJSONType())
	}
}
