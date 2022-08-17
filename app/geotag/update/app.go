package update

import (
	"context"
	"flag"
	"fmt"
	"github.com/sfomuseum/go-flags/flagset"
	"github.com/sfomuseum/go-sfomuseum-geo/geotag"
	"github.com/whosonfirst/go-reader"
	gh_writer "github.com/whosonfirst/go-writer-github/v2"
	"github.com/whosonfirst/go-writer/v2"
	"os"
)

func DefaultFlagSet(ctx context.Context) *flag.FlagSet {

	fs := flagset.NewFlagSet("geotag")

	fs.StringVar(&mode, "mode", "cli", "Valid options are: cli, lambda.")

	fs.StringVar(&depiction_reader_uri, "depiction-reader-uri", "", "A valid whosonfirst/go-reader URI.")
	fs.StringVar(&depiction_writer_uri, "depiction-writer-uri", "", "A valid whosonfirst/go-writer URI.")

	fs.StringVar(&subject_reader_uri, "subject-reader-uri", "", "A valid whosonfirst/go-reader URI.")
	fs.StringVar(&subject_writer_uri, "subject-writer-uri", "", "A valid whosonfirst/go-writer URI.")

	fs.StringVar(&parent_reader_uri, "parent-reader-uri", "", "A valid whosonfirst/go-reader URI.")

	fs.StringVar(&access_token_uri, "access-token", "", "A valid gocloud.dev/runtimevar URI")

	fs.Int64Var(&parent_id, "parent-id", -1, "A valid Who's On First ID of the record \"parenting\" the records being depicted.")

	fs.Var(&depictions, "depiction-id", "One or more valid Who's On First IDs for the records being depicted.")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "update-depiction is a command-tool for applying geotagging updates to one or more depictions.\n")
		fmt.Fprintf(os.Stderr, "Usage:\n\t %s [options]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Valid options are:\n")
		fs.PrintDefaults()
	}

	return fs
}

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

	opts := &geotag.UpdateDepictionOptions{
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
