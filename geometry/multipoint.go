package geometry

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/paulmach/orb"
	// "github.com/paulmach/orb/geojson"
	"github.com/paulmach/orb/planar"
	"github.com/tidwall/gjson"
	"github.com/whosonfirst/go-reader/v2"
	"github.com/whosonfirst/go-whosonfirst-feature/geometry"
	wof_reader "github.com/whosonfirst/go-whosonfirst-reader/v2"
)

// DeriveMultiPointFromIds generates a new `geojson.Geometry` MultiPoint instance derived from the (planar)
// centroids of the geometries associated with 'ids'.
//
// If a feature associated with an ID has a 'MultiPoint' geometry each of those points will be assign to the
// final geometry (rather than deriving a centroid).
//
// If a feature associated with an ID has either "lbl:" or "geotag:" latitude and longitude properties those
// will be used in place of a centroid derived from the features geometry.
func DeriveMultiPointFromIds(ctx context.Context, r reader.Reader, ids ...int64) (orb.Geometry, error) {

	done_ch := make(chan bool)
	err_ch := make(chan error)
	geom_ch := make(chan orb.Geometry)

	for _, id := range ids {

		go func(id int64) {

			logger := slog.Default()
			logger = logger.With("id", id)

			logger.Debug("Derive geometry")

			defer func() {
				done_ch <- true
			}()

			body, err := wof_reader.LoadBytes(ctx, r, id)

			if err != nil {
				logger.Error("Failed to read data", "error", err)
				err_ch <- fmt.Errorf("Failed to read %d, %w", id, err)
				return
			}

			prefixes := []string{
				"geotag",
				"lbl",
			}

			logger.Debug("Derive centroid from properties")

			for _, prefix := range prefixes {

				logger.Debug("Check properties", "prefix", prefix)

				lat_path := fmt.Sprintf("properties.%s:latitude", prefix)
				lon_path := fmt.Sprintf("properties.%s:longitude", prefix)

				lat_rsp := gjson.GetBytes(body, lat_path)
				lon_rsp := gjson.GetBytes(body, lon_path)

				if lat_rsp.Exists() && lon_rsp.Exists() {

					lat := lat_rsp.Float()
					lon := lon_rsp.Float()

					logger.Debug("Return centroid", "prefix", prefix)
					pt := orb.Point{lon, lat}
					geom_ch <- pt
					return
				}
			}

			logger.Debug("Derive centroid from geometry")

			geom, err := geometry.Geometry(body)

			if err != nil {
				logger.Error("Failed to derive geometry", "error", err)
				err_ch <- fmt.Errorf("Failed to derive geometry for %d, %w", id, err)
				return
			}

			orb_geom := geom.Geometry()
			geom_ch <- orb_geom
			return
		}(id)

	}

	remaining := len(ids)
	geoms := make([]orb.Geometry, 0)

	for remaining > 0 {
		select {
		case <-done_ch:
			remaining -= 1
		case err := <-err_ch:
			return nil, fmt.Errorf("Failed to derive geometry for subject, %w", err)
		case geom := <-geom_ch:
			geoms = append(geoms, geom)
		}
	}

	return DeriveMultiPointFromGeoms(ctx, geoms...)
}

// DeriveMultiPointFromGeoms generates a new `geojson.Geometry` MultiPoint instance derived from the (planar)
// centroids of the geometries associated with 'geoms'.
//
// If a feature associated with an ID has a 'MultiPoint' geometry each of those points will be assign to the
// final geometry (rather than deriving a centroid).
func DeriveMultiPointFromGeoms(ctx context.Context, geoms ...orb.Geometry) (orb.Geometry, error) {

	done_ch := make(chan bool)
	err_ch := make(chan error)
	centroid_ch := make(chan orb.Point)

	logger := slog.Default()

	for _, geom := range geoms {

		go func(orb_geom orb.Geometry) {

			switch orb_geom.GeoJSONType() {
			case "MultiPoint":

				logger.Debug("Return centroids from multipoint")
				for _, pt := range orb_geom.(orb.MultiPoint) {
					centroid_ch <- pt
				}

			default:

				logger.Debug("Return planar centroid")
				pt, _ := planar.CentroidArea(orb_geom)
				centroid_ch <- pt
			}

		}(geom)
	}

	remaining := len(geoms)
	points := make([]orb.Point, 0)

	for remaining > 0 {
		select {
		case <-done_ch:
			remaining -= 1
		case err := <-err_ch:
			return nil, fmt.Errorf("Failed to derive geometry for subject, %w", err)
		case pt := <-centroid_ch:
			points = AddPointIfNotExist(points, pt)
		}
	}

	mp := orb.MultiPoint(points)
	return mp, nil
}
