package recompile

import (
	"context"
	"flag"
	"fmt"

	"github.com/sfomuseum/go-flags/flagset"
)

// subject: a collection object, for example
// depiction: an image of a collection object, for example

type RunOptions struct {
	Verbose              bool
	SubjectReaderURI     string
	SubjectWriterURI     string
	DepictionReaderURI   string
	WhosOnFirstReaderURI string
	SFOMuseumReaderURI   string
	GitHubAccessTokenURI string
	SubjectIds           []int64
}

func RunOptionsFromFlagSet(ctx context.Context, fs *flag.FlagSet) (*RunOptions, error) {

	flagset.Parse(fs)

	err := flagset.SetFlagsFromEnvVars(fs, "SFOMUSEUM")

	if err != nil {
		return nil, fmt.Errorf("Failed to set flags from environment variables, %w", err)
	}

	opts := &RunOptions{
		Verbose:              verbose,
		SubjectReaderURI:     subject_reader_uri,
		SubjectWriterURI:     subject_writer_uri,
		DepictionReaderURI:   depiction_reader_uri,
		WhosOnFirstReaderURI: whosonfirst_reader_uri,
		SFOMuseumReaderURI:   sfomuseum_reader_uri,
		GitHubAccessTokenURI: access_token_uri,
		SubjectIds:           subject_ids,
	}

	return opts, nil
}
