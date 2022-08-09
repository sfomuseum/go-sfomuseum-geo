package export

import (
	"github.com/sfomuseum/go-sfomuseum-export/v2/properties"
	wof_export "github.com/whosonfirst/go-whosonfirst-export/v2"
)

func Prepare(feature []byte, opts *wof_export.Options) ([]byte, error) {

	var err error

	feature, err = wof_export.Prepare(feature, opts)

	if err != nil {
		return nil, err
	}

	feature, err = properties.EnsurePlacetype(feature)

	if err != nil {
		return nil, err
	}

	feature, err = properties.EnsureIsSFO(feature)

	if err != nil {
		return nil, err
	}

	feature, err = properties.EnsureSFOLevel(feature)

	if err != nil {
		return nil, err
	}

	feature, err = properties.EnsureWOFDepicts(feature)

	if err != nil {
		return nil, err
	}

	return feature, nil
}
