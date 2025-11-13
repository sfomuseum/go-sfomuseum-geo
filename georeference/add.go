package georeference

import (
	"context"
)

func AddReferences(ctx context.Context, opts *AssignReferencesOptions, depiction_id int64, refs ...*Reference) ([]byte, error) {
	return AssignReferences(ctx, opts, depiction_id, refs...)
}
