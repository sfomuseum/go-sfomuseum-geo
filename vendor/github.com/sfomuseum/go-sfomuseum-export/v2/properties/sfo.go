package properties

import (
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func EnsureSFOLevel(feature []byte) ([]byte, error) {

	path := "properties.sfo:level"

	rsp := gjson.GetBytes(feature, path)

	if !rsp.Exists() {
		return feature, nil
	}

	return sjson.SetBytes(feature, path, rsp.Int())
}
