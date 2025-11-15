package recompile

import (
	"context"
	"flag"
	"fmt"
	"log/slog"

	"github.com/sfomuseum/go-sfomuseum-geo/georeference"
	"github.com/whosonfirst/go-reader/v2"
	wof_reader "github.com/whosonfirst/go-whosonfirst-reader/v2"
	wof_writer "github.com/whosonfirst/go-whosonfirst-writer/v3"
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

	if opts.Verbose {
		slog.SetLogLoggerLevel(slog.LevelDebug)
		slog.Debug("Verbose logging enabled")
	}

	var err error

	opts.SubjectWriterURI, err = gh_writer.EnsureGitHubAccessToken(ctx, opts.SubjectWriterURI, opts.GitHubAccessTokenURI)

	if err != nil {
		return fmt.Errorf("Failed to ensure access token for subject writer URI, %w", err)
	}

	depiction_reader, err := reader.NewReader(ctx, opts.DepictionReaderURI)

	if err != nil {
		return fmt.Errorf("Failed to create depiction reader, %w", err)
	}

	/*
		whosonfirst_reader, err := reader.NewReader(ctx, opts.WhosOnFirstReaderURI)

		if err != nil {
			return fmt.Errorf("Failed to create whosonfirst reader, %w", err)
		}
	*/

	sfomuseum_reader, err := reader.NewReader(ctx, opts.SFOMuseumReaderURI)

	if err != nil {
		return fmt.Errorf("Failed to create architecture reader, %w", err)
	}

	subject_reader, err := reader.NewReader(ctx, opts.SubjectReaderURI)

	if err != nil {
		return fmt.Errorf("Failed to create subject reader, %w", err)
	}

	subject_writer, err := writer.NewWriter(ctx, opts.SubjectWriterURI)

	if err != nil {
		return fmt.Errorf("Failed to create subject writer, %w", err)
	}

	recompile_opts := &georeference.RecompileGeorefencesForSubjectOptions{
		DepictionReader: depiction_reader,
		SFOMuseumReader: sfomuseum_reader,
	}

	for _, id := range opts.SubjectIds {

		body, err := wof_reader.LoadBytes(ctx, subject_reader, id)

		if err != nil {
			return fmt.Errorf("Failed to read body for %s, %w", id, err)
		}

		has_changed, new_body, err := georeference.RecompileGeorefencesForSubject(ctx, recompile_opts, body)

		if err != nil {
			return fmt.Errorf("Failed to recompile georeferences for %d, %w", id, err)
		}

		if !has_changed {
			continue
		}

		_, err = wof_writer.WriteBytes(ctx, subject_writer, new_body)

		if err != nil {
			return fmt.Errorf("Failed to write changes for %d, %w", id, err)
		}
	}

	return nil
}
