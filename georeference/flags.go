package georeference

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/sfomuseum/go-flags/multi"
	"github.com/sfomuseum/go-sfomuseum-geo"
)

// MultiKeyValueStringsToReferences converts a list of `multi.KeyValueString` key-value pairs in to a list of `Reference` instances.
func MultiKeyValueStringsToReferences(kv_references multi.KeyValueString) ([]*Reference, error) {

	refs := make([]*Reference, len(kv_references))

	for refs_idx, kv := range kv_references {

		k := kv.Key()
		v := kv.Value().(string)

		prop := k

		switch prop {
		case geo.RESERVED_GEOTAG_DEPICTIONS, geo.RESERVED_GEOREFERENCE_DEPICTIONS:
			return nil, fmt.Errorf("'%s' is a reserved property", prop)
		default:
			// pass
		}

		str_ids := strings.Split(v, ",")

		ids := make([]int64, len(str_ids))

		for ids_idx, str_id := range str_ids {

			id, err := strconv.ParseInt(str_id, 10, 64)

			if err != nil {
				return nil, fmt.Errorf("Failed to parse ID '%s', %w", str_id, err)
			}

			ids[ids_idx] = id
		}

		label := strings.Replace(prop, ":", "_", -1)
		label = strings.Replace(label, ".", "_", -1)

		r := &Reference{
			Ids:      ids,
			Property: prop,
			AltLabel: label,
		}

		refs[refs_idx] = r
	}

	return refs, nil
}
