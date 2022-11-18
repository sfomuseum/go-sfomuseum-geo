package geometry

import (
	"github.com/paulmach/orb"
)

// AddPointIfNotExist adds 'new' to 'points' unless it is already present.
func AddPointIfNotExist(points []orb.Point, new orb.Point) []orb.Point {

	exists := false

	for _, pt := range points {

		if pt.Equal(new) {
			exists = true
			break
		}
	}

	if !exists {
		points = append(points, new)
	}

	return points
}
