package update

import (
	"context"
	"flag"
	"fmt"

	"github.com/sfomuseum/go-flags/flagset"
	"github.com/sfomuseum/go-sfomuseum-geo/geotag"
	"github.com/whosonfirst/go-reader/v2"
	gh_writer "github.com/whosonfirst/go-writer-github/v3"
	"github.com/whosonfirst/go-writer/v3"
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

	depiction_writer, err := writer.NewWriter(ctx, depiction_writer_uri)

	if err != nil {
		return fmt.Errorf("Failed to create depiction writer, %v", err)
	}

	subject_reader, err := reader.NewReader(ctx, subject_reader_uri)

	if err != nil {
		return fmt.Errorf("Failed to create subject reader, %v", err)
	}

	subject_writer, err := writer.NewWriter(ctx, subject_writer_uri)

	if err != nil {
		return fmt.Errorf("Failed to create subject writer, %v", err)
	}

	parent_reader, err := reader.NewReader(ctx, parent_reader_uri)

	if err != nil {
		return fmt.Errorf("Failed to create architecture reader, %v", err)
	}

	opts := &geotag.GeotagDepictionOptions{
		DepictionReader:    depiction_reader,
		DepictionWriter:    depiction_writer,
		SubjectReader:      subject_reader,
		SubjectWriter:      subject_writer,
		ParentReader:       parent_reader,
		DepictionWriterURI: depiction_writer_uri, // to be remove post writer/v3 (Clone) release
		SubjectWriterURI:   subject_writer_uri,   // to be remove post writer/v3 (Clone) release
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
