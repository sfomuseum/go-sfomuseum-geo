package assign

import (
	"context"
	"flag"
	"fmt"
	_ "log/slog"

	"github.com/sfomuseum/go-sfomuseum-geo/georeference"
	"github.com/whosonfirst/go-reader"
	gh_writer "github.com/whosonfirst/go-writer-github/v3"
	"github.com/whosonfirst/go-writer/v3"
)

// Run executes the "assign flight cover georeferences" application with a default `flag.FlagSet` instance.
func Run(ctx context.Context) error {
	fs := DefaultFlagSet(ctx)
	return RunWithFlagSet(ctx, fs)
}

// RunWithFlagSet executes the "assign flight cover georeferences" application with a `flag.FlagSet` instance defined by 'fs'.
func RunWithFlagSet(ctx context.Context, fs *flag.FlagSet) error {

	opts, err := RunOptionsFromFlagSet(ctx, fs)

	if err != nil {
		return err
	}

	return RunWithOptions(ctx, opts)
}

func RunWithOptions(ctx context.Context, opts *RunOptions) error {

	var err error

	opts.DepictionWriterURI, err = gh_writer.EnsureGitHubAccessToken(ctx, opts.DepictionWriterURI, opts.GitHubAccessTokenURI)

	if err != nil {
		return fmt.Errorf("Failed to ensure access token for depiction writer URI, %v", err)
	}

	opts.SubjectWriterURI, err = gh_writer.EnsureGitHubAccessToken(ctx, opts.SubjectWriterURI, opts.GitHubAccessTokenURI)

	if err != nil {
		return fmt.Errorf("Failed to ensure access token for subject writer URI, %v", err)
	}

	depiction_reader, err := reader.NewReader(ctx, opts.DepictionReaderURI)

	if err != nil {
		return fmt.Errorf("Failed to create depiction reader, %v", err)
	}

	depiction_writer, err := writer.NewWriter(ctx, opts.DepictionWriterURI)

	if err != nil {
		return fmt.Errorf("Failed to create depiction writer, %v", err)
	}

	subject_reader, err := reader.NewReader(ctx, opts.SubjectReaderURI)

	if err != nil {
		return fmt.Errorf("Failed to create subject reader, %v", err)
	}

	subject_writer, err := writer.NewWriter(ctx, opts.SubjectWriterURI)

	if err != nil {
		return fmt.Errorf("Failed to create subject writer, %v", err)
	}

	whosonfirst_reader, err := reader.NewReader(ctx, opts.WhosOnFirstReaderURI)

	if err != nil {
		return fmt.Errorf("Failed to create whosonfirst reader, %v", err)
	}

	sfomuseum_reader, err := reader.NewReader(ctx, opts.SFOMuseumReaderURI)

	if err != nil {
		return fmt.Errorf("Failed to create architecture reader, %v", err)
	}

	assign_opts := &georeference.AssignReferencesOptions{
		DepictionReader:    depiction_reader,
		DepictionWriter:    depiction_writer,
		SubjectReader:      subject_reader,
		SubjectWriter:      subject_writer,
		WhosOnFirstReader:  whosonfirst_reader,
		SFOMuseumReader:    sfomuseum_reader,
		DepictionWriterURI: depiction_writer_uri, // to be remove post writer/v4 (Clone) release
		SubjectWriterURI:   subject_writer_uri,   // to be remove post writer/v4 (Clone) release
	}

	switch mode {
	case "cli":

		for _, id := range opts.Depictions {

			_, err := georeference.AssignReferences(ctx, assign_opts, id, opts.References...)

			if err != nil {
				return fmt.Errorf("Failed to georeference depiction %d, %w", id, err)
			}
		}

	default:
		return fmt.Errorf("Invalid or unsupported mode")
	}

	return nil
}
