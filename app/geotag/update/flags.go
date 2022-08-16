package update

import (
	"github.com/sfomuseum/go-flags/multi"
)

var mode string

var depiction_reader_uri string
var depiction_writer_uri string

var subject_reader_uri string
var subject_writer_uri string

var parent_id int64
var parent_reader_uri string

var access_token_uri string

var depictions multi.MultiInt64
