package georeference

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/sfomuseum/go-flags/multi"
)

// MultiKeyValueStringsToReferences converts a list of `multi.KeyValueString` key-value pairs in to a list of `Reference` instances.
func MultiKeyValueStringsToReferences(kv_references multi.KeyValueString) ([]*Reference, error) {

	refs := make([]*Reference, len(kv_references))

	for refs_idx, kv := range kv_references {

		k := kv.Key()
		v := kv.Value().(string)

		label := k

		str_ids := strings.Split(v, ",")

		ids := make([]int64, len(str_ids))

		for ids_idx, str_id := range str_ids {

			id, err := strconv.ParseInt(str_id, 10, 64)

			if err != nil {
				return nil, fmt.Errorf("Failed to parse ID '%s', %w", str_id, err)
			}

			ids[ids_idx] = id
		}

		// START OF make me better, more nuanced

		alt_label := strings.Replace(label, ":", "_", -1)
		alt_label = strings.Replace(alt_label, ".", "_", -1)
		alt_label = strings.Replace(alt_label, " ", "_", -1)

		// END OF make me better, more nuanced

		r := &Reference{
			Ids:      ids,
			Label:    label,
			AltLabel: label,
		}

		refs[refs_idx] = r
	}

	return refs, nil
}
