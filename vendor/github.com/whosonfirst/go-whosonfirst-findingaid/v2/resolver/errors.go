package resolver

import (
	"errors"
)

// ErrNotFound is an error indicating that an item is not present or was not found.
var ErrNotFound = errors.New("Not found")
