package reader

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/whosonfirst/go-ioutil"
	wof_reader "github.com/whosonfirst/go-reader/v2"
)

type GitHubReader struct {
	wof_reader.Reader
	owner    string
	repo     string
	branch   string
	prefix   string
	throttle <-chan time.Time
}

func init() {

	ctx := context.Background()
	err := wof_reader.RegisterReader(ctx, "github", NewGitHubReader)

	if err != nil {
		panic(err)
	}
}

func NewGitHubReader(ctx context.Context, uri string) (wof_reader.Reader, error) {

	u, err := url.Parse(uri)

	if err != nil {
		return nil, fmt.Errorf("Failed to parse URI, %w", err)
	}

	rate := time.Second / 3
	throttle := time.Tick(rate)

	r := &GitHubReader{
		throttle: throttle,
	}

	r.owner = u.Host

	path := strings.TrimLeft(u.Path, "/")
	parts := strings.Split(path, "/")

	if len(parts) != 1 {
		return nil, fmt.Errorf("Invalid path")
	}

	r.repo = parts[0]
	r.branch = DEFAULT_BRANCH

	q := u.Query()

	branch := q.Get("branch")

	if branch != "" {
		r.branch = branch
	}

	prefix := q.Get("prefix")

	if prefix != "" {
		r.prefix = prefix
	}

	return r, nil
}

func (r *GitHubReader) Exists(ctx context.Context, uri string) (bool, error) {

	<-r.throttle

	logger := slog.Default()
	logger = logger.With("uri", uri)

	url := r.ReaderURI(ctx, uri)

	logger.Debug("Read URL", "url", url)

	rsp, err := http.Head(url)

	if err != nil {
		return false, fmt.Errorf("Failed to GET uri, %w", err)
	}

	switch rsp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		return false, fmt.Errorf("Unexpected status: %s", rsp.Status)
	}
}

func (r *GitHubReader) Read(ctx context.Context, uri string) (io.ReadSeekCloser, error) {

	<-r.throttle

	logger := slog.Default()
	logger = logger.With("uri", uri)

	url := r.ReaderURI(ctx, uri)

	logger.Debug("Read URL", "url", url)

	rsp, err := http.Get(url)

	if err != nil {
		return nil, fmt.Errorf("Failed to GET uri, %w", err)
	}

	if rsp.StatusCode != 200 {
		return nil, fmt.Errorf("Unexpected status: %s", rsp.Status)
	}

	fh, err := ioutil.NewReadSeekCloser(rsp.Body)

	if err != nil {
		return nil, fmt.Errorf("Failed to create ReadSeekCloser, %w", err)
	}

	return fh, nil
}

func (r *GitHubReader) ReaderURI(ctx context.Context, key string) string {

	if r.prefix != "" {
		key = filepath.Join(r.prefix, key)
	}

	return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", r.owner, r.repo, r.branch, key)
}
