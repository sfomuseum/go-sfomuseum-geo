package georeference

// type Reference is a struct that encapusulates data about a place being georeferenced.
type Reference struct {
	// Ids are the Who's On First ID of the place being referenced
	Ids []int64 `json:"id"`
	// Property is the (relative) label to use for the class of georeference.
	Property string `json:"property"`
	// AltLabel is the alternate geometry label to use for the class of georeference.
	AltLabel string `json:"alt_label"`
}
