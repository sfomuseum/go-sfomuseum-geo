package georeference

import (
	"github.com/whosonfirst/go-reader"
	"github.com/whosonfirst/go-writer/v2"
)

// UpdateDepictionOptions defines a struct for reading/writing options when updating geo-related information in depictions.
// A depiction is assumed to be the record for an image or some other piece of media. A subject is assumed to be
// the record for an object.
type UpdateDepictionOptions struct {
	// A valid whosonfirst/go-reader.Reader instance for reading depiction features.
	DepictionReader reader.Reader
	// A valid whosonfirst/go-writer.Writer instance for writing depiction features.
	DepictionWriter writer.Writer
	// A valid whosonfirst/go-reader.Reader instance for reading subject features.
	SubjectReader reader.Reader
	// A valid whosonfirst/go-writer.Writer instance for writing subject features.
	SubjectWriter writer.Writer
	// A valid whosonfirst/go-reader.Reader instance for reading "parent" features.
	WhosOnFirstReader  reader.Reader
	Author             string
	DepictionWriterURI string // To be removed with the go-writer/v3 (clone) release
	SubjectWriterURI   string // To be removed with the go-writer/v3 (clone) release
}
