package geotag

import (
	"context"
	"fmt"
	"slices"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
	"github.com/sfomuseum/go-sfomuseum-geo/geometry"
	"github.com/tidwall/gjson"
	"github.com/whosonfirst/go-reader/v2"
	wof_reader "github.com/whosonfirst/go-whosonfirst-reader/v2"
)

type DeriveGeoreferenceCoordsOptions struct {
	WhosOnFirstReader reader.Reader
}

type DeriveGeometryForDepictionOptions struct {
	WhosOnFirstReader reader.Reader
}

type DeriveGeometryForSubjectOptions struct {
	WhosOnFirstReader reader.Reader
	DepictionReader   reader.Reader
}

func DeriveGeoreferenceCoords(ctx context.Context, opts *DeriveGeoreferenceCoordsOptions, body []byte) (orb.MultiPoint, error) {

	// Note: Defining this as *orb.MultiPoint makes all the slice.Contains stuff below sad.
	coords := orb.MultiPoint(make([]orb.Point, 0))

	georef_rsp := gjson.GetBytes(body, "properties.georef:depictions")
	georef_ids := make([]int64, 0)

	for _, ids_rsp := range georef_rsp.Map() {

		for _, id_rsp := range ids_rsp.Array() {

			id := id_rsp.Int()

			if !slices.Contains(georef_ids, id) {
				georef_ids = append(georef_ids, id)
			}
		}
	}

	if len(georef_ids) > 0 {

		geom, err := geometry.DeriveMultiPointFromIds(ctx, opts.WhosOnFirstReader, georef_ids...)

		if err != nil {
			return nil, fmt.Errorf("Failed to derive multipoint geometry for subject, %w", err)
		}

		coords = geom.Geometry().(orb.MultiPoint)
	}

	return coords, nil
}

func DeriveGeometryForDepiction(ctx context.Context, opts *DeriveGeometryForDepictionOptions, body []byte) (*geojson.Geometry, error) {

	georef_opts := &DeriveGeoreferenceCoordsOptions{
		WhosOnFirstReader: opts.WhosOnFirstReader,
	}

	coords, err := DeriveGeoreferenceCoords(ctx, georef_opts, body)

	if err != nil {
		return nil, fmt.Errorf("Failed to derive georeference coordinates, %w", err)
	}

	lat_rsp := gjson.GetBytes(body, "properties.geotag:camera_latitude")
	lon_rsp := gjson.GetBytes(body, "properties.geotag:camera_longitude")

	if lat_rsp.Exists() && lon_rsp.Exists() {

		lat := lat_rsp.Float()
		lon := lon_rsp.Float()

		pt := orb.Point([2]float64{lon, lat})

		if !slices.Contains(coords, pt) {
			coords = append(coords, pt)
		}
	}

	switch len(coords) {
	case 0:
		return nil, nil
	case 1:
		new_pt := orb.Point(coords[0])
		new_geom := geojson.NewGeometry(new_pt)
		return new_geom, nil
	default:
		new_mp := orb.MultiPoint(coords)
		new_geom := geojson.NewGeometry(new_mp)
		return new_geom, nil
	}
}

func DeriveGeometryForSubject(ctx context.Context, opts *DeriveGeometryForSubjectOptions, body []byte) (*geojson.Geometry, error) {

	georef_opts := &DeriveGeoreferenceCoordsOptions{
		WhosOnFirstReader: opts.WhosOnFirstReader,
	}

	coords, err := DeriveGeoreferenceCoords(ctx, georef_opts, body)

	if err != nil {
		return nil, fmt.Errorf("Failed to derive georeference coordinates, %w", err)
	}

	geotag_rsp := gjson.GetBytes(body, "properties.geotag:depictions")

	for _, id_rsp := range geotag_rsp.Array() {

		depiction_id := id_rsp.Int()

		depiction_body, err := wof_reader.LoadBytes(ctx, opts.DepictionReader, depiction_id)

		if err != nil {
			return nil, fmt.Errorf("Failed to load record for depiction (%d), %w", depiction_id, err)
		}

		lat_rsp := gjson.GetBytes(depiction_body, "properties.geotag:camera_latitude")
		lon_rsp := gjson.GetBytes(depiction_body, "properties.geotag:camera_longitude")

		if lat_rsp.Exists() && lon_rsp.Exists() {

			lat := lat_rsp.Float()
			lon := lon_rsp.Float()

			pt := orb.Point([2]float64{lon, lat})

			if !slices.Contains(coords, pt) {
				coords = append(coords, pt)
			}
		}
	}

	switch len(coords) {
	case 0:
		return nil, nil
	case 1:
		new_pt := orb.Point(coords[0])
		new_geom := geojson.NewGeometry(new_pt)
		return new_geom, nil
	default:
		new_mp := orb.MultiPoint(coords)
		new_geom := geojson.NewGeometry(new_mp)
		return new_geom, nil
	}
}
