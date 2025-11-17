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
2025/11/17 12:29:31 DEBUG Load image "subject id"=1511954455 "image id"=1527853759
2025/11/17 12:29:31 DEBUG Load image "subject id"=1511954455 "image id"=1527852489
2025/11/17 12:29:31 DEBUG Depictions for image "subject id"=1511954455 image_id=1527853763 key=georef:depicted count=3
2025/11/17 12:29:31 DEBUG Dispatch image "subject id"=1511954455 image=1527853763 key=to place=85825929
2025/11/17 12:29:31 DEBUG Dispatch image "subject id"=1511954455 image=1527853763 key=from place=85937601
2025/11/17 12:29:31 DEBUG Update subject (geo)refs for image "subject id"=1511954455 id=85825929 label=to
2025/11/17 12:29:31 DEBUG Update subject (geo)refs for image "subject id"=1511954455 id=85937601 label=from
2025/11/17 12:29:31 DEBUG Dispatch image "subject id"=1511954455 image=1527853763 key=postmark place=85922583
2025/11/17 12:29:31 DEBUG Dispatch image "subject id"=1511954455 image=1527853763 key=postmark place=85937601
2025/11/17 12:29:31 DEBUG Update subject (geo)refs for image "subject id"=1511954455 id=85922583 label=postmark
2025/11/17 12:29:31 DEBUG Update subject (geo)refs for image "subject id"=1511954455 id=85937601 label=postmark
2025/11/17 12:29:31 DEBUG Depictions for image "subject id"=1511954455 image_id=1527853759 key=georef:depicted count=1
2025/11/17 12:29:31 DEBUG Dispatch image "subject id"=1511954455 image=1527853759 key=postmark place=85937601
2025/11/17 12:29:31 DEBUG Dispatch image "subject id"=1511954455 image=1527853759 key=postmark place=85921923
2025/11/17 12:29:31 DEBUG Update subject (geo)refs for image "subject id"=1511954455 id=85937601 label=postmark
2025/11/17 12:29:31 DEBUG Update subject (geo)refs for image "subject id"=1511954455 id=85921923 label=postmark
2025/11/17 12:29:31 DEBUG Depictions for image "subject id"=1511954455 image_id=1527852489 key=georef:depicted count=1
2025/11/17 12:29:31 DEBUG Dispatch image "subject id"=1511954455 image=1527852489 key=depiction place=404529563
2025/11/17 12:29:31 DEBUG Update subject (geo)refs for image "subject id"=1511954455 id=404529563 label=depiction
2025/11/17 12:29:31 DEBUG Inflate belongs and derive hierarchy for georeference "subject id"=1511954455 "belongsto id"=85921923
2025/11/17 12:29:31 DEBUG Inflate belongs and derive hierarchy for georeference "subject id"=1511954455 "belongsto id"=85937601
2025/11/17 12:29:31 DEBUG Inflate belongs and derive hierarchy for georeference "subject id"=1511954455 "belongsto id"=85825929
2025/11/17 12:29:31 DEBUG Inflate belongs and derive hierarchy for georeference "subject id"=1511954455 "belongsto id"=404529563
2025/11/17 12:29:31 DEBUG Inflate belongs and derive hierarchy for georeference "subject id"=1511954455 "belongsto id"=85922583
2025/11/17 12:29:33 DEBUG Additional geometries "subject id"=1511954455 count=3
2025/11/17 12:29:33 DEBUG Derive multipoint from geometries (with WOF reader) "subject id"=1511954455 count=3
2025/11/17 12:29:33 DEBUG Derive geometry id=1527853763
2025/11/17 12:29:33 DEBUG Derive geometry id=1527853759
2025/11/17 12:29:33 DEBUG Derive geometry id=1527852489
2025/11/17 12:29:33 DEBUG Derive centroid from properties id=1527853763
2025/11/17 12:29:33 DEBUG Check properties id=1527853763 prefix=geotag
2025/11/17 12:29:33 DEBUG Check properties id=1527853763 prefix=lbl
2025/11/17 12:29:33 DEBUG Derive centroid from geometry id=1527853763
2025/11/17 12:29:33 DEBUG Derive centroid from properties id=1527853759
2025/11/17 12:29:33 DEBUG Check properties id=1527853759 prefix=geotag
2025/11/17 12:29:33 DEBUG Check properties id=1527853759 prefix=lbl
2025/11/17 12:29:33 DEBUG Derive centroid from geometry id=1527853759
2025/11/17 12:29:34 DEBUG Derive centroid from properties id=1527852489
2025/11/17 12:29:34 DEBUG Check properties id=1527852489 prefix=geotag
2025/11/17 12:29:34 DEBUG Check properties id=1527852489 prefix=lbl
2025/11/17 12:29:34 DEBUG Derive centroid from geometry id=1527852489
2025/11/17 12:29:34 DEBUG Return centroids from multipoint
2025/11/17 12:29:34 DEBUG Return centroids from multipoint
2025/11/17 12:29:34 DEBUG Return centroids from multipoint
2025/11/17 12:29:34 DEBUG Add point if not exist point="[-122.299603 37.697078]"
2025/11/17 12:29:34 DEBUG Add point if not exist point="[-73.767075 40.713095]"
2025/11/17 12:29:34 DEBUG Add point if not exist point="[-157.837169 21.330576]"
2025/11/17 12:29:34 DEBUG Add point if not exist point="[-157.837169 21.330576]"
2025/11/17 12:29:34 DEBUG Add point if not exist point="[-122.273474 37.75478]"
2025/11/17 12:29:34 DEBUG Add point if not exist point="[-122.431272 37.778008]"

*/

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

	whosonfirst_reader, err := reader.NewReader(ctx, opts.WhosOnFirstReaderURI)

	if err != nil {
		return fmt.Errorf("Failed to create whosonfirst reader, %w", err)
	}

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
		DepictionReader:          depiction_reader,
		SFOMuseumReader:          sfomuseum_reader,
		WhosOnFirstReader:        whosonfirst_reader,
		DefaultGeometryFeatureId: opts.DefaultGeometryFeatureId,
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

	return nil
}
