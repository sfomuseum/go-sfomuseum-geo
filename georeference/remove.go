package georeference

import (
	"context"
)

func RemoveAllReferences(ctx context.Context, opts *AssignReferencesOptions, depiction_id int64) ([]byte, error) {
	return AssignReferences(ctx, opts, depiction_id)
}
