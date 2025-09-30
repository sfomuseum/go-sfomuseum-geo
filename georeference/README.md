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

### Subject

| Name | Type | Notes |
| --- | --- | --- |
| `georef:depictions` | []int64 | The list of georeferenced depiction (image) IDs for the subject. |
| `georef:depicted` | map[string]int64 | The list of georeferenced labels and Who's On First IDs for this subject (object) for all the depictions (images) of this subject. |
| `georef:whosonfirst_belongsto` | []int64 | The unique set of Who's on First IDs that are parents or ancestors for the set of georeferenced (images) IDs for this subject (object) |

## Geometries

### Depiction

...stored as a `MultiPoint` geometry in a `alt-georef-{LABEL}` alternate geometry file.

A `MultiPoint` geometry derived from the `alt-georef-{LABEL}` alternate geometry files.

### Subject

A `MultiPoint` geometry derived from the (`Point`) geometries of all the depictions associated with a subject.
