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
| `georef:subject` | int64 | The subject (object) that this depiction (image) represents. |
| `georef:depicted` | map[string]int64 | The list of georeferenced labels and Who's On First IDs for this depiction (image) |
| `georef:whosonfirst_belongsto` | []int64 | The unique set of Who's on First IDs that are parents or ancestors for the set of georeferenced (images) IDs for this subject (object) |
| `georef:lastmodified` | int64 | The Unix timestamp when the record's georeference data was last modified. |

### Subject

| Name | Type | Notes |
| --- | --- | --- |
| `georef:depictions` | []int64 | The list of georeferenced depiction (image) IDs for the subject. |
| `georef:depicted` | map[string]int64 | The list of georeferenced labels and Who's On First IDs for this subject (object) for all the depictions (images) of this subject. |
| `georef:whosonfirst_belongsto` | []int64 | The unique set of Who's on First IDs that are parents or ancestors for the set of georeferenced (images) IDs for this subject (object) |
| `georef:lastmodified` | int64 | The Unix timestamp when the record's georeference data was last modified. |

## Geometries

### Depiction

Every depiction has a per-label alternate geometry file for each label `georef:depicted`. These alternate geometry files take the name of `alt-georef-{LABEL}` and has a `MultiPoint` geometry composed of the set of (primary) centroids for each of the Who's On First IDs associated with that label.

The geometry for the depiction itself is a `MultiPoint` geometry composed of all the `alt-georef-{LABEL}` alternate geometries associated with it.

### Subject

A `MultiPoint` geometry derived from the (`MultiPoint`) geometries of all the depictions associated with the subject.
