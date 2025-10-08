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

// Writers is a struct which encapsulates a pair of `whosonfirst/go-writer/v3.Writer` instances for
// writing data to depictions and subjects respectively. One is a simple Writer instance where data
// is expected to be persisted to. The second is a `writer.MultiWriter` instance which writes to both
// the default writer and a local in-memory `writer.IOWriter` instance. This allows the `Writers` struct
// to expose an `AsFeatureCollection` methods which can be invoked to return the update depiction and
// subject data (to the calling application) without having to query for that data from source. The
// reason that both the solitary writer and the multi writer are exposed is because there is often the
// need to write "alternate geometry" files is because the 'whosonfirst/go-whosonfirstwriter/v3.WriteBytes`
// function which is typically used to wrap writing data does not support alternate geometies. It should
// and eventually will but for the time being it doesn't.
type Writers struct {
	// A `whosonfirst/go-writer/v3.Writer` instance for writing depiction data to.
	DepictionWriter writer.Writer
	// A `whosonfirst/go-writer/v3.MultiWriter` instance wrapping both the principal `DepictionWriter` instance and an in-memory `writer.IOWriter` instance for writing depiction data to.
	DepictionMultiWriter writer.Writer
	// A `whosonfirst/go-writer/v3.Writer` instance for writing subject data to.
	SubjectWriter writer.Writer
	// A `whosonfirst/go-writer/v3.MultiWriter` instance wrapping both the principal `SubjectWriter` instance and an in-memory `writer.IOWriter` instance for writing subject data to.
	SubjectMultiWriter writer.Writer

	depictionBuf       *bytes.Buffer
	depictionBufWriter *bufio.Writer
	subjectBuf         *bytes.Buffer
	subjectBufWriter   *bufio.Writer
}

// CreateWritersOptions is a struct containing configuration details for the `CreateWriters` method.
type CreateWritersOptions struct {
	// A registered `whosonfirst/go-writer/v3.Writer` URI describing where depiction data will be written to.
	DepictionWriterURI string
	// A registered `whosonfirst/go-writer/v3.Writer` URI describing where subject data will be written to.
	SubjectWriterURI string
	// An option `github.UpdateWriterURIOptions` struct used to append GitHub API / PR specific data to writers.
	GithubWriterOptions *github.UpdateWriterURIOptions
}

// CreateWriters will returns a new `Writers` instance derived from 'opts'.
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
