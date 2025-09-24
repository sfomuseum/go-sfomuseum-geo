# geotag

_Work in progress_

## Roles

### Depiction

A depiction "depicts" a subject. For example an image (depiction) depicts an object (subject).

### Subject

A subject is a "thing" with zero or more depictions. For example an object (subject) has two depictions (images).

## Properties

### Depiction

| Name | Type | Notes |
| --- | --- | --- |
| `geotag:angle` | float64 | The angle of the `geotag:camera` view. |
| `geotag:bearing` | float64 | The bearing of the `geotag:camera` view. |
| `geotag:camera_latitude` | float64 | The latitude for "camera" view used to geotag a depiction. |
| `geotag:camera_longitude` | float64 | The longitude for "camera" view used to geotag a depiction. |
| `geotag:distance` | float64 | The distance between the `geotag:camera_latitude|longitude` and `geotag:target_latitude|longitude` values. |
| `geotag:subject` | float64 | The subject (object) that this depiction (image) represents. |
| `geotag:target_latitude` | float64 | The latitude for "field of view" point used to geotag a depiction. |
| `geotag:target_longitude` | float64 | The longitude for "field of view" point used to geotag a depiction. |
| `geotag:whosonfirst_belongsto` | []float64 | The unique set of Who's on First IDs that are parents or ancestors for the set of geotagged depictions (images) IDs for this subject (object) |
| `geotag:whosonfirst_camera` | float64 | The Who's On First ID associated with `geotag:camera_latitude|longitude` (point-in-polygon) |
| `geotag:whosonfirst_target` | float64 | The Who's On First ID associated with `geotag:target_latitude|longitude` (point-in-polygon) |

### Subject

| Name | Type | Notes |
| --- | --- | --- |
| `geotag:depictions | []float64 | The list of geotagged depictions (images) IDs for this subject (object) |
| `geotag:whosonfirst_belongsto | []float64 | The unique set of Who's on First IDs that are parents or ancestors for the set of geotagged depictions (images) IDs for this subject (object) |
| `geotag:whosonfirst_camera | []float64 | The unique set of `geotag:whosonfirst_camera` values for the geotagged depictions (images) IDs for this subject (object). _This is currently being encoded as a single ID rather than a list._ |
| `geotag:whosonfirst_target | []float64 | The unique set of `geotag:whosonfist_target` values for the geotagged depictions (images) IDs for this subject (object). _This is currently being encoded as a single ID rather than a list._ |

## Geometries

### Depiction

`Point`

...stored in a `alt-geotag-fov` alternate geometry file.

### Subject

A `MultiPoint` geometry derived from the (`Point`) geometries of all the depictions associated with a subject.
