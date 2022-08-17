package assign

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/sfomuseum/go-sfomuseum-geo/georeference"
	"os"
)

func runCommandLine(ctx context.Context, opts *georeference.AssignReferencesOptions) error {

	var refs []*georeference.Reference

	dec := json.NewDecoder(os.Stdin)
	err := dec.Decode(&refs)

	if err != nil {
		return fmt.Errorf("Failed to decode references, %v", err)
	}

	for _, depiction_id := range depictions {

		_, err := georeference.AssignReferences(ctx, opts, depiction_id, refs...)

		if err != nil {
			return fmt.Errorf("Failed to assign references for depiction %d, %v", depiction_id, err)
		}
	}

	return nil
}
