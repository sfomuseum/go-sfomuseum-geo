package georeference

import (
	"testing"
)

func TestDeriveAltLabel(t *testing.T) {

	tests := map[string]string{
		"hello world":      "hello_world",
		"Hello world":      "hello_world",
		"Hello world, Bob": "hello_world_bob",
		"This is a-test":   "this_is_a_test",
	}

	for raw, expected := range tests {

		label := DeriveAltLabel(raw)

		if label != expected {
			t.Fatalf("Unexpected value for '%s'. Expected '%s' but got '%s'", raw, expected, label)
		}
	}
}
