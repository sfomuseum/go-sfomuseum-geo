package geotag

import (
	"bufio"
	"bytes"
	"context"
	"fmt"

	"github.com/sfomuseum/go-sfomuseum-geo/github"
	"github.com/whosonfirst/go-writer/v3"
)

type GeotagWriterOptions struct {
	DepictionId        int64
	Author             string
	DepictionWriterURI string
	SubjectWriterURI   string
}

func GeotagWriters(ctx context.Context, opts *GeotagWriterOptions) (writer.Writer, writer.Writer, error) {

	// START OF to refactor with go-writer/v4 (clone) release

	update_opts := &github.UpdateWriterURIOptions{
		WhosOnFirstId: opts.DepictionId,
		Author:        opts.Author,
		Action:        github.GeotagAction,
	}

	depiction_writer_uri, err := github.UpdateWriterURI(ctx, update_opts, opts.DepictionWriterURI)

	if err != nil {
		return nil, nil, fmt.Errorf("Failed to update depiction writer URI, %w", err)
	}

	subject_writer_uri, err := github.UpdateWriterURI(ctx, update_opts, opts.SubjectWriterURI)

	if err != nil {
		return nil, nil, fmt.Errorf("Failed to update subject writer URI, %w", err)
	}

	depiction_writer, err := writer.NewWriter(ctx, depiction_writer_uri)

	if err != nil {
		return nil, nil, fmt.Errorf("Failed to create new depiction writer for '%s', %w", depiction_writer_uri, err)
	}

	subject_writer, err := writer.NewWriter(ctx, subject_writer_uri)

	if err != nil {
		return nil, nil, fmt.Errorf("Failed to create new subject writer for '%s', %w", subject_writer_uri, err)
	}

	// END OF to refactor with go-writer/v4 (clone) release

	// START OF hooks to capture updates/writes so we can parrot them back in the method response
	// We're doing it this way because the code, as written, relies on sfomuseum/go-sfomuseum-writer
	// which hides the format-and-export stages and modifies the document being written. To account
	// for this we'll just keep local copies of those updates in *_buf and reference them at the end.
	// Note that we are not doing this for the alternate geometry feature (alt_body) since are manually
	// formatting, exporting and writing a byte slice in advance of better support for alternate
	// geometries in the tooling.

	// The buffers where we will write updated Feature information
	var local_depiction_buf bytes.Buffer
	var local_subject_buf bytes.Buffer

	// The io.Writers where we will write updated Feature information
	local_depiction_buf_writer := bufio.NewWriter(&local_depiction_buf)
	local_subject_buf_writer := bufio.NewWriter(&local_subject_buf)

	// The writer.Writer where we will write updated Feature information
	local_depiction_writer, err := writer.NewIOWriterWithWriter(ctx, local_depiction_buf_writer)

	if err != nil {
		return nil, nil, fmt.Errorf("Failed to create IOWriter for depiction, %w", err)
	}

	local_subject_writer, err := writer.NewIOWriterWithWriter(ctx, local_subject_buf_writer)

	if err != nil {
		return nil, nil, fmt.Errorf("Failed to create IOWriter for subject, %w", err)
	}

	// The writer.MultiWriter where we will write updated Feature information
	depiction_mw, err := writer.NewMultiWriter(ctx, depiction_writer, local_depiction_writer)

	if err != nil {
		return nil, nil, fmt.Errorf("Failed to create multi writer for depiction, %w", err)
	}

	subject_mw, err := writer.NewMultiWriter(ctx, subject_writer, local_subject_writer)

	if err != nil {
		return nil, nil, fmt.Errorf("Failed to create multi writer for subject, %w", err)
	}

	// END OF hooks to capture updates/writes so we can parrot them back in the method response

	return depiction_mw, subject_mw, nil
}
