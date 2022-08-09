package properties

import (
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func EnsureWOFDepicts(feature []byte) ([]byte, error) {

	depicts_map := make(map[int64]bool)

	paths := []string{
		"properties.wof:depicts",
		"properties.millsfield:airline_id",
		"properties.millsfield:airport_id",
		"properties.millsfield:aircraft_id",
		"properties.millsfield:company_id",		
	}

	for _, p := range paths {

		rsp := gjson.GetBytes(feature, p)

		if !rsp.Exists() {
			continue
		}

		for _, id_rsp := range rsp.Array() {
			depicts_map[id_rsp.Int()] = true
		}
	}

	depicts := make([]int64, 0)

	for id, _ := range depicts_map {

		if id <= 0 {
			continue
		}

		depicts = append(depicts, id)
	}

	if len(depicts) == 0 {
		return feature, nil
	}

	return sjson.SetBytes(feature, "properties.wof:depicts", depicts)
}
