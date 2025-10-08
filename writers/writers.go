package writers

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log/slog"

	"github.com/paulmach/orb/geojson"
	"github.com/sfomuseum/go-sfomuseum-geo/github"
	"github.com/whosonfirst/go-writer/v3"
)

type Writers struct {
	DepictionWriter      writer.Writer
	DepictionMultiWriter writer.Writer
	SubjectWriter        writer.Writer
	SubjectMultiWriter   writer.Writer

	depictionBuf       *bytes.Buffer
	depictionBufWriter *bufio.Writer
	subjectBuf         *bytes.Buffer
	subjectBufWriter   *bufio.Writer
}

type CreateWritersOptions struct {
	DepictionWriterURI  string
	SubjectWriterURI    string
	GithubWriterOptions *github.UpdateWriterURIOptions
}

func CreateWriters(ctx context.Context, opts *CreateWritersOptions) (*Writers, error) {

	depiction_writer_uri := opts.DepictionWriterURI
	subject_writer_uri := opts.SubjectWriterURI

	if opts.GithubWriterOptions != nil {

		var err error

		depiction_writer_uri, err = github.UpdateWriterURI(ctx, opts.GithubWriterOptions, depiction_writer_uri)

		if err != nil {
			return nil, fmt.Errorf("Failed to update depiction writer URI, %w", err)
		}

		subject_writer_uri, err = github.UpdateWriterURI(ctx, opts.GithubWriterOptions, subject_writer_uri)

		if err != nil {
			return nil, fmt.Errorf("Failed to update subject writer URI, %w", err)
		}
	}

	depiction_writer, err := writer.NewWriter(ctx, depiction_writer_uri)

	if err != nil {
		return nil, fmt.Errorf("Failed to create new depiction writer for '%s', %w", depiction_writer_uri, err)
	}

	subject_writer, err := writer.NewWriter(ctx, subject_writer_uri)

	if err != nil {
		return nil, fmt.Errorf("Failed to create new subject writer for '%s', %w", subject_writer_uri, err)
	}

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
		return nil, fmt.Errorf("Failed to create IOWriter for depiction, %w", err)
	}

	local_subject_writer, err := writer.NewIOWriterWithWriter(ctx, local_subject_buf_writer)

	if err != nil {
		return nil, fmt.Errorf("Failed to create IOWriter for subject, %w", err)
	}

	// The writer.MultiWriter(s) where we will write updated Feature information

	depiction_mw, err := writer.NewMultiWriter(ctx, depiction_writer, local_depiction_writer)

	if err != nil {
		return nil, fmt.Errorf("Failed to create multi writer for depiction, %w", err)
	}

	subject_mw, err := writer.NewMultiWriter(ctx, subject_writer, local_subject_writer)

	if err != nil {
		return nil, fmt.Errorf("Failed to create multi writer for subject, %w", err)
	}

	// END OF hooks to capture updates/writes so we can parrot them back in the method response

	all_writers := &Writers{
		DepictionWriter:      depiction_writer,
		SubjectWriter:        subject_writer,
		DepictionMultiWriter: depiction_mw,
		SubjectMultiWriter:   subject_mw,
		depictionBufWriter:   local_depiction_buf_writer,
		subjectBufWriter:     local_subject_buf_writer,
		// See the part where we're storing points to local_*_buf?
		// That is important so we can read them later in AsFeatureCollection
		depictionBuf: &local_depiction_buf,
		subjectBuf:   &local_subject_buf,
	}

	return all_writers, nil
}

func (writers *Writers) AsFeatureCollection() (*geojson.FeatureCollection, error) {

	writers.depictionBufWriter.Flush()
	writers.subjectBufWriter.Flush()

	fc := geojson.NewFeatureCollection()

	new_subject_body, err := geojson.UnmarshalFeature(writers.subjectBuf.Bytes())

	if err != nil {
		slog.Error("Bad subject buffer", "body", string(writers.subjectBuf.Bytes()))
		return nil, fmt.Errorf("Failed to unmarshal feature from subject buffer, %w", err)
	}

	fc.Append(new_subject_body)

	new_depiction_body, err := geojson.UnmarshalFeature(writers.depictionBuf.Bytes())

	if err != nil {
		return nil, fmt.Errorf("Failed to unmarshal feature from depiction buffer, %w", err)
	}

	fc.Append(new_depiction_body)

	return fc, nil
}
