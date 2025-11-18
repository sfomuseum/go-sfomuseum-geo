package recompile

/*

> ./bin/georef-recompile-subject -subject-id 1511954455 -verbose
2025/11/17 12:29:31 DEBUG Verbose logging enabled
2025/11/17 12:29:31 DEBUG Recompile georeferences for subject "subject id"=1511954455
2025/11/17 12:29:31 DEBUG Process images for subject "subject id"=1511954455 count=3
2025/11/17 12:29:31 DEBUG Derive georef details from image "subject id"=1511954455 id=1527852489
2025/11/17 12:29:31 DEBUG Derive georef details from image "subject id"=1511954455 id=1527853759
2025/11/17 12:29:31 DEBUG Derive georef details from image "subject id"=1511954455 id=1527853763
2025/11/17 12:29:31 DEBUG Load image "subject id"=1511954455 "image id"=1527853763
...
2025/11/17 12:29:34 DEBUG Add point if not exist point="[-157.837169 21.330576]"
2025/11/17 12:29:34 DEBUG Add point if not exist point="[-122.273474 37.75478]"
2025/11/17 12:29:34 DEBUG Add point if not exist point="[-122.431272 37.778008]"

*/

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"

	"github.com/sfomuseum/go-sfomuseum-geo/georeference"
	"github.com/whosonfirst/go-reader/v2"
	"github.com/whosonfirst/go-whosonfirst-iterate/v3"
	wof_reader "github.com/whosonfirst/go-whosonfirst-reader/v2"
	wof_writer "github.com/whosonfirst/go-whosonfirst-writer/v3"
	gh_writer "github.com/whosonfirst/go-writer-github/v3"
	"github.com/whosonfirst/go-writer/v3"
)

// Run executes the "geoference-recompile-subject" application with a default `flag.FlagSet` instance.
func Run(ctx context.Context) error {
	fs := DefaultFlagSet(ctx)
	return RunWithFlagSet(ctx, fs)
}

// RunWithFlagSet executes the "geoference-recompile-subject" application with a `flag.FlagSet` instance defined by 'fs'.
func RunWithFlagSet(ctx context.Context, fs *flag.FlagSet) error {

	opts, err := RunOptionsFromFlagSet(ctx, fs)

	if err != nil {
		return err
	}

	return RunWithOptions(ctx, opts)
}

// RunWithFlagSet executes the "geoference-recompile-subject" application with 'opts'.
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

	whosonfirst_reader, err := reader.NewReader(ctx, opts.WhosOnFirstReaderURI)

	if err != nil {
		return fmt.Errorf("Failed to create whosonfirst reader, %w", err)
	}

	sfomuseum_reader, err := reader.NewReader(ctx, opts.SFOMuseumReaderURI)

	if err != nil {
		return fmt.Errorf("Failed to create architecture reader, %w", err)
	}

	subject_writer, err := writer.NewWriter(ctx, opts.SubjectWriterURI)

	if err != nil {
		return fmt.Errorf("Failed to create subject writer, %w", err)
	}

	recompile_opts := &georeference.RecompileGeorefencesForSubjectOptions{
		DepictionReader:          depiction_reader,
		SFOMuseumReader:          sfomuseum_reader,
		WhosOnFirstReader:        whosonfirst_reader,
		DefaultGeometryFeatureId: opts.DefaultGeometryFeatureId,
	}

	if len(opts.SubjectIds) > 0 {

		slog.Debug("Recompile georeference data for specific record IDs", "count", len(opts.SubjectIds))

		subject_reader, err := reader.NewReader(ctx, opts.SubjectReaderURI)

		if err != nil {
			return fmt.Errorf("Failed to create subject reader, %w", err)
		}

		for _, id := range opts.SubjectIds {

			body, err := wof_reader.LoadBytes(ctx, subject_reader, id)

			if err != nil {
				return fmt.Errorf("Failed to read body for %d, %w", id, err)
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
	}

	if len(opts.IteratorSources) > 0 {

		slog.Debug("Recompile georeference data from iterator", "source", len(opts.IteratorURI))

		iter, err := iterate.NewIterator(ctx, opts.IteratorURI)

		if err != nil {
			return fmt.Errorf("Failed to create new iterator, %w", err)
		}

		for rec, err := range iter.Iterate(ctx, opts.IteratorSources...) {

			if err != nil {
				return fmt.Errorf("Iterator signaled an error, %w", err)
			}

			body, err := io.ReadAll(rec.Body)
			rec.Body.Close()

			if err != nil {
				return fmt.Errorf("Failed to close reader for %s, %w", rec.Path, err)
			}

			has_changed, new_body, err := georeference.RecompileGeorefencesForSubject(ctx, recompile_opts, body)

			if err != nil {
				return fmt.Errorf("Failed to recompile georeferences for %s, %w", rec.Path, err)
			}

			if !has_changed {
				continue
			}

			_, err = wof_writer.WriteBytes(ctx, subject_writer, new_body)

			if err != nil {
				return fmt.Errorf("Failed to write changes for %s, %w", rec.Path, err)
			}
		}
	}

	return nil
}
