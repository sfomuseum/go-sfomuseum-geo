package geotag

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// CameraProperties is a struct containing Who's On First pointers for the point where the "camera" is located.
type CameraProperties struct {
	// The ID of the Who's On First (WOF) feature that immediately parents the point where the "camera" is located.
	ParentId int64 `json:"wof:parent_id,omitempty"`
	// The ID of the Who's On First (WOF) hierarchy of ancestors that parents the point where the "camera" is located.
	Hierarchy []map[string]int64 `json:"wof:hierarchy,omitempty"`
}

// TargetProperties is a struct containing Who's On First pointers for the point where the "camera" is pointed.
type TargetProperties struct {
	// The ID of the Who's On First (WOF) feature that immediately parents the point targeted by the "camera".
	ParentId int64 `json:"wof:parent_id,omitempty"`
	// The ID of the Who's On First (WOF) hierarchy of ancestors that parents the point targeted by the "camera".
	Hierarchy []map[string]int64 `json:"wof:hierarchy,omitempty"`
}

// GeotagProperties defines properties exported by the `Leaflet.GeotagPhoto` package
// and, optionally, by the sfomuseum/go-www-geotag package.
type GeotagProperties struct {
	// The angle for the field of view
	Angle float64 `json:"geotag:angle"`
	// The bearing for camera
	Bearing float64 `json:"geotag:bearing"`
	// The distance from the camera to the center point of the field of view's edges.
	Distance float64 `json:"geotag:distance"`
	// Camera is a `CameraProperties` instance containing Who's On First pointers for the point where the "camera" is located.
	Camera *CameraProperties `json:"geotag:camera"`
	// Target is a `TargetProperties` instance containing Who's On First pointers for the point where the "camera" is pointed.
	Target *TargetProperties `json:"geotag:target"`
}

// GeotagCoordinate defines a longitude, latitude coorindate.
type GeotagCoordinate [2]float64

// GeotagPoint defines a GeoJSON `Point` geometry element.
type GeotagPoint struct {
	// The GeoJSON type for this geometry. It is expected to be `Point`
	Type string `json:"type"`
	// The GeoJSON coordinates associated with this struct.
	Coordinates GeotagCoordinate `json:"coordinates"`
}

// GeotagPoint defines a GeoJSON `LineString` geometry element.
type GeotagLineString struct {
	// The GeoJSON type for this geometry. It is expected to be `LineString`
	Type string `json:"type"`
	// The GeoJSON coordinates associated with this struct.
	Coordinates [2]GeotagCoordinate `json:"coordinates"`
}

// GeotagPolygon defines a GeoJSON `Polygon` geometry element.
type GeotagPolygon struct {
	// The GeoJSON type for this geometry. It is expected to be `Polygon`
	Type string `json:"type"`
	// The GeoJSON coordinates associated with this struct.
	Coordinates [][]GeotagCoordinate `json:"coordinates"`
}

// GeotagGeometryCollection defines a GeoJSON `GeometryCollection` geometry element.
type GeotagGeometryCollection struct {
	// The GeoJSON type for this geometry. It is expected to be `GeometryCollection`
	Type string `json:"type"`
	// The GeoJSON coordinates associated with this struct.
	Geometries [2]interface{} `json:"geometries"`
}

// GeotagFeature defines a GeoJSON Feature produced by the `Leaflet.GeotagPhoto` package
type GeotagFeature struct {
	// The unique ID associated with this feature.
	Id string `json:"id,omitempty"`
	// The GeoJSON type for this feature. It is expected to be `Feature`
	Type string `json:"type"`
	// The GeoJSON coordinates associated with this struct.
	Geometry GeotagGeometryCollection `json:"geometry"`
	// The metadata properties associated with this feature.
	Properties GeotagProperties `json:"properties"`
}

// NewGeotagFeatureWithReader returns a new `GeotagFeature` reading data from 'body'.
func NewGeotagFeature(body []byte) (*GeotagFeature, error) {

	br := bytes.NewReader(body)
	return NewGeotagFeatureWithReader(br)
}

// NewGeotagFeatureWithReader returns a new `GeotagFeature` reading data from 'r'.
func NewGeotagFeatureWithReader(r io.Reader) (*GeotagFeature, error) {

	var f *GeotagFeature

	decoder := json.NewDecoder(r)
	err := decoder.Decode(&f)

	if err != nil {
		return nil, fmt.Errorf("Failed to decode feature, %w", err)
	}

	return f, nil
}

// Target returns the coordinate at the center of the "horizon line" for f
// as a point.
func (f *GeotagFeature) Target() (*GeotagPoint, error) {

	hl, err := f.HorizonLine()

	if err != nil {
		return nil, fmt.Errorf("Failed to derive horizon line, %w", err)
	}

	coords := hl.Coordinates

	target_lat := (coords[0][1] + coords[1][1]) / 2.0
	target_lon := (coords[0][0] + coords[1][0]) / 2.0

	target_coords := GeotagCoordinate{
		target_lon,
		target_lat,
	}

	target := &GeotagPoint{
		Type:        "Point",
		Coordinates: target_coords,
	}

	return target, nil
}

// PointOfView returns the focal point of 'f' as a point. The focal
// point is defined as the coordinate where the "camera" is located.
func (f *GeotagFeature) PointOfView() (*GeotagPoint, error) {

	// if there is a better way to do this I wish I
	// knew what it was... (20200410/thisisaaronland)

	raw := f.Geometry.Geometries[0]
	enc, err := json.Marshal(raw)

	if err != nil {
		return nil, fmt.Errorf("Failed to marshal geometries, %w", err)
	}

	var pov *GeotagPoint
	err = json.Unmarshal(enc, &pov)

	if err != nil {
		return nil, fmt.Errorf("Failed to unmarshal point of view, %w", err)
	}

	return pov, nil
}

// HorizonLine returns the "horizon line" for 'f' as a line string. The "horizon" line is
// defined as the line between the two outer points in the field of view.
func (f *GeotagFeature) HorizonLine() (*GeotagLineString, error) {

	raw := f.Geometry.Geometries[1]
	enc, err := json.Marshal(raw)

	if err != nil {
		return nil, fmt.Errorf("Failed to marshal geometry, %w", err)
	}

	var ls *GeotagLineString
	err = json.Unmarshal(enc, &ls)

	if err != nil {
		return nil, fmt.Errorf("Failed to unmarshal line string, %w", err)
	}

	return ls, nil
}

// FieldOfView returns the "field of view" area for 'f' as a polygon. The "field of view"
// is defined as the point from which the camera view begins to its two outer extremities.
func (f *GeotagFeature) FieldOfView() (*GeotagPolygon, error) {

	pov, err := f.PointOfView()

	if err != nil {
		return nil, fmt.Errorf("Failed to derive point of view, %w", err)
	}

	hl, err := f.HorizonLine()

	if err != nil {
		return nil, fmt.Errorf("Failed to derive horizon line, %w", err)
	}

	coords := make([]GeotagCoordinate, 0)

	coords = append(coords, pov.Coordinates)
	coords = append(coords, hl.Coordinates[1]) // right-hand rule
	coords = append(coords, hl.Coordinates[0])
	coords = append(coords, pov.Coordinates)

	rings := [][]GeotagCoordinate{coords}

	poly := &GeotagPolygon{
		Type:        "Polygon",
		Coordinates: rings,
	}

	return poly, nil
}
