package geometry

import (
	"context"
	"fmt"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
	"github.com/paulmach/orb/planar"
	"github.com/whosonfirst/go-reader"
	"github.com/whosonfirst/go-whosonfirst-feature/geometry"
	wof_reader "github.com/whosonfirst/go-whosonfirst-reader"
)

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

			geom, err := geometry.Geometry(body)

			if err != nil {
				err_ch <- fmt.Errorf("Failed to derive geometry for %d, %w", id, err)
				return
			}

			orb_geom := geom.Geometry()
			pt, _ := planar.CentroidArea(orb_geom)

			centroid_ch <- pt
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
