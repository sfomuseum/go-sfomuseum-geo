package export

import (
	"context"
	"encoding/json"
	wof_export "github.com/whosonfirst/go-whosonfirst-export/v2"
)

type SFOMuseumExporter struct {
	wof_export.Exporter
	options *wof_export.Options
}

func init() {

	ctx := context.Background()

	err := wof_export.RegisterExporter(ctx, "sfomuseum", NewSFOMuseumExporter)

	if err != nil {
		panic(err)
	}
}

func NewSFOMuseumExporter(ctx context.Context, uri string) (wof_export.Exporter, error) {

	opts, err := wof_export.NewDefaultOptions(ctx)

	if err != nil {
		return nil, err
	}

	ex := &SFOMuseumExporter{
		options: opts,
	}

	return ex, nil
}

func (ex *SFOMuseumExporter) ExportFeature(ctx context.Context, feature interface{}) ([]byte, error) {

	body, err := json.Marshal(feature)

	if err != nil {
		return nil, err
	}

	return ex.Export(ctx, body)
}

func (ex *SFOMuseumExporter) Export(ctx context.Context, feature []byte) ([]byte, error) {

	var err error

	feature, err = Prepare(feature, ex.options)

	if err != nil {
		return nil, err
	}

	feature, err = wof_export.Format(feature, ex.options)

	if err != nil {
		return nil, err
	}

	return feature, nil
}
