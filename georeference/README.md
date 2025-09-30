# georeference

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
| `georef:depictions` | map[string]float64 | The list of georeferenced labels and Who's On First IDs for this depiction (image) |
| `georef:whosonfirst_belongsto` | []float64 | The unique set of Who's on First IDs that are parents or ancestors for the set of georeferenced (images) IDs for this subject (object) |
| `georef:subject` | float64 | The subject (object) that this depiction (image) represents. |

### Subject

| Name | Type | Notes |
| --- | --- | --- |
| `georef:depictions` | map[string]float64 | The list of georeferenced labels and Who's On First IDs for this subject (object) for all the depictions (images) of this subject. |
| `georef:whosonfirst_belongsto` | []float64 | The unique set of Who's on First IDs that are parents or ancestors for the set of georeferenced (images) IDs for this subject (object) |

## Geometries

### Depiction

A `MultiPoint` geometry derived from the (principal) centroid for each the places referenced in `georef:depictions`.

...stored in a `alt-georef-{LABEL}` alternate geometry file. _This may be retired._

### Subject

A `MultiPoint` geometry derived from the (`Point`) geometries of all the depictions associated with a subject.
