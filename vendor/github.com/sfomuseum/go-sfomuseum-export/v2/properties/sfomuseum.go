package properties

import (
	"errors"
	
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func EnsurePlacetype(feature []byte) ([]byte, error) {

	rsp := gjson.GetBytes(feature, "properties.sfomuseum:placetype")

	if !rsp.Exists() {
		return feature, errors.New("missing sfomuseum:placetype")
	}

	pt_alt := []string{
		rsp.String(),
	}
	
	return sjson.SetBytes(feature, "wof:placetype_alt", pt_alt)
}

func EnsureIsSFO(feature []byte) ([]byte, error) {

	rsp := gjson.GetBytes(feature, "properties.sfomuseum:is_sfo")

	if rsp.Exists() {
		return feature, nil
	}

	return sjson.SetBytes(feature, "sfomuseum:is_sfo", -1)
}
