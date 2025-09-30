package resolver

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
)

// type MultiResolver implements the `Resolver` interface for resolving data using one or more `Resolver` instances.
type MultiResolver struct {
	Resolver
	resolvers []Resolver
}

func init() {
	ctx := context.Background()
	RegisterResolver(ctx, "multi", NewMultiResolver)
}

// NewMultiResolver will return a new `Resolver` instance for resolving	data using one or more `Resolver` instances.
func NewMultiResolver(ctx context.Context, uri string) (Resolver, error) {

	u, err := url.Parse(uri)

	if err != nil {
		return nil, fmt.Errorf("Failed to parse URL, %w", err)
	}

	q := u.Query()

	if !q.Has("resolver") {
		return nil, fmt.Errorf("URI has no ?resolver= parameters")
	}

	resolvers := make([]Resolver, len(q["resolver"]))

	for idx, r_uri := range q["resolver"] {

		r, err := NewResolver(ctx, r_uri)

		if err != nil {
			return nil, fmt.Errorf("Failed to create resolver for '%s', %w", r_uri, err)
		}

		resolvers[idx] = r
	}

	multi_r := &MultiResolver{
		resolvers: resolvers,
	}

	return multi_r, nil
}

// GetRepo returns the name of the repository associated with this ID from the first of one or more matching `Resolver` instances.
func (r *MultiResolver) GetRepo(ctx context.Context, id int64) (string, error) {

	for _, other_r := range r.resolvers {

		repo, err := other_r.GetRepo(ctx, id)

		if err != nil {
			slog.Debug("Failed to get repo", "id", id, "resolver", fmt.Sprintf("%T", other_r), "error", err)
			continue
		}

		return repo, nil
	}

	return "", fmt.Errorf("Not found")
}
