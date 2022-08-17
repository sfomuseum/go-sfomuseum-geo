package flightcover

// For example:
// 	./bin/assign-flight-cover -depiction-id 1527853781 -sent 85632163 -to 85977539

import (
	"context"
	"flag"
	"fmt"
	"github.com/sfomuseum/go-flags/flagset"
	"github.com/sfomuseum/go-sfomuseum-geo/georeference"
	"github.com/whosonfirst/go-reader"
	gh_writer "github.com/whosonfirst/go-writer-github/v2"
	"github.com/whosonfirst/go-writer/v2"
)

// Run executes the "assign flight cover georeferences" application with a default `flag.FlagSet` instance.
func Run(ctx context.Context) error {
	fs := DefaultFlagSet(ctx)
	return RunWithFlagSet(ctx, fs)
}

// RunWithFlagSet executes the "assign flight cover georeferences" application with a `flag.FlagSet` instance defined by 'fs'.
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

	whosonfirst_reader, err := reader.NewReader(ctx, whosonfirst_reader_uri)

	if err != nil {
		return fmt.Errorf("Failed to create whosonfirst reader, %v", err)
	}

	opts := &georeference.AssignReferencesOptions{
		DepictionReader:    depiction_reader,
		DepictionWriter:    depiction_writer,
		SubjectReader:      subject_reader,
		SubjectWriter:      subject_writer,
		WhosOnFirstReader:  whosonfirst_reader,
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
