package geometry

import (
	"github.com/paulmach/orb"
	"testing"
)

func TestAddPointIfNotExist(t *testing.T) {

	test_points := []orb.Point{
		orb.Point{-64.764346, 32.296698},
		orb.Point{-118.25703, 34.05513},
		orb.Point{-118.25703, 34.05513},
	}

	points := make([]orb.Point, 0)

	for _, pt := range test_points {
		points = AddPointIfNotExist(points, pt)
	}

	if len(points) != 2 {
		t.Fatalf("Expected 2 points, but got %d", len(points))
	}

}
