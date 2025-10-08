package remove

import (
	"context"
	"flag"
	"fmt"
	"log/slog"

	"github.com/paulmach/orb/geojson"
	"github.com/sfomuseum/go-flags/flagset"
	"github.com/sfomuseum/go-sfomuseum-geo/geotag"
	"github.com/whosonfirst/go-reader/v2"
	wof_reader "github.com/whosonfirst/go-whosonfirst-reader/v2"
	gh_writer "github.com/whosonfirst/go-writer-github/v3"
)

func Run(ctx context.Context) error {
	fs := DefaultFlagSet(ctx)
	return RunWithFlagSet(ctx, fs)
}

func RunWithFlagSet(ctx context.Context, fs *flag.FlagSet) error {

	flagset.Parse(fs)

	err := flagset.SetFlagsFromEnvVars(fs, "SFOMUSEUM")

	if err != nil {
		return fmt.Errorf("Failed to set flags from environment variables, %w", err)
	}

	if verbose {
		slog.SetLogLoggerLevel(slog.LevelDebug)
		slog.Debug("Verbose logging enabled")
	}

	depiction_writer_uri, err = gh_writer.EnsureGitHubAccessToken(ctx, depiction_writer_uri, access_token_uri)

	if err != nil {
		return fmt.Errorf("Failed to ensure access token for depiction writer URI, %v", err)
	}

	subject_writer_uri, err = gh_writer.EnsureGitHubAccessToken(ctx, subject_writer_uri, access_token_uri)

	if err != nil {
		return fmt.Errorf("Failed to ensure access token for subject writer URI, %v", err)
	}

	depiction_reader, err := reader.NewReader(ctx, depiction_reader_uri)

	if err != nil {
		return fmt.Errorf("Failed to create depiction reader, %v", err)
	}

	subject_reader, err := reader.NewReader(ctx, subject_reader_uri)

	if err != nil {
		return fmt.Errorf("Failed to create subject reader, %v", err)
	}

	whosonfirst_reader, err := reader.NewReader(ctx, wof_reader_uri)

	if err != nil {
		return fmt.Errorf("Failed to create whosonfirst reader, %v", err)
	}

	default_geom_record, err := wof_reader.LoadBytes(ctx, whosonfirst_reader, default_geometry_id)

	if err != nil {
		return fmt.Errorf("Failed to load feature for default geometry ID, %w", err)
	}

	default_geom_f, err := geojson.UnmarshalFeature(default_geom_record)

	if err != nil {
		return fmt.Errorf("Failed to unmarshal feature for default geometry, %w", err)
	}

	default_geom := geojson.NewGeometry(default_geom_f.Geometry)

	opts := &geotag.RemoveGeotagDepictionOptions{
		DepictionReader:    depiction_reader,
		DepictionWriterURI: depiction_writer_uri,
		SubjectReader:      subject_reader,
		SubjectWriterURI:   subject_writer_uri,
		WhosOnFirstReader:  whosonfirst_reader,
		DefaultGeometry:    default_geom,
		Author:             "",
	}

	switch mode {
	case "cli":
		return runCommandLine(ctx, opts)
	case "lambda":
		return runLambda(ctx, opts)
	default:
		return fmt.Errorf("Invalid or unsupported mode")
	}
}
