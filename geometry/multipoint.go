package geometry

import (
	"context"
	"fmt"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
	"github.com/paulmach/orb/planar"
	"github.com/tidwall/gjson"
	"github.com/whosonfirst/go-reader"
	"github.com/whosonfirst/go-whosonfirst-feature/geometry"
	wof_reader "github.com/whosonfirst/go-whosonfirst-reader"
)

// DeriveMultiPointFromIds generates a new `geojson.Geometry` MultiPoint instance derived from the (planar)
// centroids of the geometries associated with 'ids'.
//
// If a feature associated with an ID has a 'MultiPoint' geometry each of those points will be assign to the
// final geometry (rather than deriving a centroid).
//
// If a feature associated with an ID has either "lbl:" or "geotag:" latitude and longitude properties those
// will be used in place of a centroid derived from the features geometry.
func DeriveMultiPointFromIds(ctx context.Context, r reader.Reader, ids ...int64) (*geojson.Geometry, error) {

	done_ch := make(chan bool)
	err_ch := make(chan error)
	centroid_ch := make(chan orb.Point)

	for _, id := range ids {

		go func(id int64) {

			defer func() {
				done_ch <- true
			}()

			body, err := wof_reader.LoadBytes(ctx, r, id)

			if err != nil {
				err_ch <- fmt.Errorf("Failed to read %d, %w", id, err)
				return
			}

			prefixes := []string{
				"geotag",
				"lbl",
			}

			for _, prefix := range prefixes {

				lat_path := fmt.Sprintf("properties.%s:latitude", prefix)
				lon_path := fmt.Sprintf("properties.%s:longitude", prefix)

				lat_rsp := gjson.GetBytes(body, lat_path)
				lon_rsp := gjson.GetBytes(body, lon_path)

				if lat_rsp.Exists() && lon_rsp.Exists() {

					pt := orb.Point{lon_rsp.Float(), lat_rsp.Float()}
					centroid_ch <- pt
					return
				}
			}

			geom, err := geometry.Geometry(body)

			if err != nil {
				err_ch <- fmt.Errorf("Failed to derive geometry for %d, %w", id, err)
				return
			}

			orb_geom := geom.Geometry()

			switch orb_geom.GeoJSONType() {
			case "MultiPoint":

				for _, pt := range orb_geom.(orb.MultiPoint) {
					centroid_ch <- pt
				}

			default:
				pt, _ := planar.CentroidArea(orb_geom)
				centroid_ch <- pt
			}

			return
		}(id)

	}

	remaining := len(ids)
	points := make([]orb.Point, 0)

	for remaining > 0 {
		select {
		case <-done_ch:
			remaining -= 1
		case err := <-err_ch:
			return nil, fmt.Errorf("Failed to derive geometry for subject, %w", err)
		case pt := <-centroid_ch:
			points = append(points, pt)
		}
	}

	mp := orb.MultiPoint(points)
	geom := geojson.NewGeometry(mp)

	return geom, nil
}
