package georeference

import (
	"fmt"
	"strconv"
	"strings"
)

// type Reference is a struct that encapusulates data about a place being georeferenced.
type Reference struct {
	// Ids are the Who's On First ID of the place being referenced
	Ids []int64 `json:"ids"`
	// Property is the string label to use for the class of georeference.
	Label string `json:"label"`
	// AltLabel is the alternate geometry label to use for the class of georeference.
	AltLabel string `json:"alt_label"`
}

func (r *Reference) String() string {

	str_ids := make([]string, len(r.Ids))

	for idx, id := range r.Ids {
		str_ids[idx] = strconv.FormatInt(id, 10)
	}

	return fmt.Sprintf("%s: %s", r.Label, strings.Join(str_ids, ","))
}
