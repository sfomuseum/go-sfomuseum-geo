package findingaid

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"strings"

	"github.com/jtacoma/uritemplates"
	wof_reader "github.com/whosonfirst/go-reader/v2"
	"github.com/whosonfirst/go-whosonfirst-findingaid/v2/resolver"
	wof_uri "github.com/whosonfirst/go-whosonfirst-uri"
)

// WHOSONFIRST_DATA_TEMPLATE is a URL template for the root `data` directory in Who's On First data repositories.
const WHOSONFIRST_DATA_TEMPLATE string = "https://raw.githubusercontent.com/whosonfirst-data/{repo}/master/data/"

// findingaid is a struct defining a resolver.Resolver and *uritemplates.UriTemplate pair
type findingaid struct {
	// A resolver.Resolver instance used to derive the Who's On First repository name for an ID.
	resolver resolver.Resolver
	// A compiled `uritemplates.UriTemplate` to use resolving Who's On First finding aid URIs.
	template *uritemplates.UriTemplate
}

// type FindingAidReader implements the `whosonfirst/go-reader` interface for use with Who's On First finding aids.
type FindingAidReader struct {
	wof_reader.Reader
	findingaids []*findingaid
}

func init() {
	ctx := context.Background()
	wof_reader.RegisterReader(ctx, "findingaid", NewFindingAidReader)
}

// NewFindingAidReader will return a new `whosonfirst/go-reader.Reader` instance for reading Who's On First
// documents by first resolving a URL using a Who's On First finding aid.
func NewFindingAidReader(ctx context.Context, uri string) (wof_reader.Reader, error) {

	u, err := url.Parse(uri)

	if err != nil {
		return nil, fmt.Errorf("Failed to parse URL, %w", err)
	}

	q := u.Query()

	findingaids := make([]*findingaid, 0)

	logger := slog.Default()
	logger = logger.With("scheme", u.Host)
	
	switch u.Host {
	case "multi":

		for _, rt_uri := range q["resolver"] {

			rt_u, err := url.Parse(rt_uri)

			if err != nil {
				return nil, fmt.Errorf("Failed to parse resolver URI, %w", err)
			}

			fa_u := url.URL{}
			fa_u.Scheme = "findingaid"
			fa_u.Host = rt_u.Scheme
			fa_u.Path = rt_u.Host + rt_u.Path
			fa_u.RawQuery = rt_u.RawQuery

			fa_uri := fa_u.String()

			r, t, err := deriveResolverAndTemplate(ctx, fa_uri)

			if err != nil {
				return nil, fmt.Errorf("Failed to derive resolver and template from ?resolver= URI, %w", err)
			}

			fa := &findingaid{
				resolver: r,
				template: t,
			}

			logger.Debug("Add findingaid reader", "uri", fa_uri)
			findingaids = append(findingaids, fa)
		}

	default:

		r, t, err := deriveResolverAndTemplate(ctx, uri)

		if err != nil {
			return nil, err
		}

		logger.Debug("Add findingaid reader", "uri", uri)
		
		findingaids = []*findingaid{
			&findingaid{
				resolver: r,
				template: t,
			},
		}
	}

	r := &FindingAidReader{
		findingaids: findingaids,
	}

	return r, nil
}

func (r *FindingAidReader) Exists(ctx context.Context, uri string) (bool, error) {

	new_r, rel_path, err := r.getReaderAndPath(ctx, uri)

	if err != nil {
		return false, fmt.Errorf("Failed to derive reader and path, %w", err)
	}

	return new_r.Exists(ctx, rel_path)
}

// Read returns an `io.ReadSeekCloser` instance for the document resolved by `uri`.
func (r *FindingAidReader) Read(ctx context.Context, uri string) (io.ReadSeekCloser, error) {

	new_r, rel_path, err := r.getReaderAndPath(ctx, uri)

	if err != nil {
		return nil, fmt.Errorf("Failed to derive reader and path, %w", err)
	}

	fh, err := new_r.Read(ctx, rel_path)

	if err != nil {
		return nil, fmt.Errorf("Failed to read %s (%s), %w", uri, rel_path, err)
	}

	return fh, nil
}

// ReaderURI returns final URI resolved by `uri` for this reader.
func (r *FindingAidReader) ReaderURI(ctx context.Context, uri string) string {
	return uri
}

// getReaderAndPath returns a new `whosonfirst/go-reader.Reader` instance, and the relative path,
// to use for reading documents resolved by `uri`.
func (r *FindingAidReader) getReaderAndPath(ctx context.Context, uri string) (wof_reader.Reader, string, error) {

	reader_uri, rel_path, err := r.getReaderURIAndPath(ctx, uri)

	if err != nil {
		return nil, "", fmt.Errorf("Failed to derive reader URI and path, %w", err)
	}

	new_r, err := wof_reader.NewReader(ctx, reader_uri)

	if err != nil {
		return nil, "", fmt.Errorf("Failed to create reader, %w", err)
	}

	return new_r, rel_path, nil
}

// getReaderAndPath returns a new `whosonfirst/go-reader.Reader` URI, and the relative path,
// to use for reading documents resolved by `uri`.
func (r *FindingAidReader) getReaderURIAndPath(ctx context.Context, uri string) (string, string, error) {

	// TBD: cache this?

	logger := slog.Default()
	logger = logger.With("uri", uri)

	id, uri_args, err := wof_uri.ParseURI(uri)

	if err != nil {
		logger.Error("Failed to parse URI", "error", err)
		return "", "", fmt.Errorf("Failed to parse URI, %w", err)
	}

	rel_path, err := wof_uri.Id2RelPath(id, uri_args)

	if err != nil {
		logger.Error("Failed to derive relative path for ID", "id", id, "error", err)
		return "", "", fmt.Errorf("Failed to derive path, %w", err)
	}

	for idx, fa := range r.findingaids {

		logger.Debug("Get repo", "findingaid", idx, "id", id)
		repo, err := fa.resolver.GetRepo(ctx, id)

		if err != nil {

			if err == resolver.ErrNotFound {
				logger.Debug("Failed to derive repo with resolver, not found", "id", id)
				continue
			}

			logger.Error("Failed to derive repo with resolver", "id", id, "error", err)
			return "", "", fmt.Errorf("Failed to derive repo, %w", err)
		}

		values := map[string]interface{}{
			"repo": repo,
		}

		reader_uri, err := fa.template.Expand(values)

		if err != nil {
			logger.Error("Failed to expand template for resolver", "repo", repo, "template", fa.template, "error", err)
			return "", "", fmt.Errorf("Failed to derive reader URI, %w", err)
		}

		logger.Debug("Return reader URI for resolver", "reader_uri", reader_uri, "rel_path", rel_path)
		return reader_uri, rel_path, nil
	}

	return "", "", fmt.Errorf("Failed to derive repo, no findingaid matches")
}

func deriveResolverAndTemplate(ctx context.Context, uri string) (resolver.Resolver, *uritemplates.UriTemplate, error) {

	u, err := url.Parse(uri)

	if err != nil {
		return nil, nil, err
	}

	q := u.Query()

	uri_template := WHOSONFIRST_DATA_TEMPLATE

	if q.Get("template") != "" {
		uri_template = q.Get("template")
	}

	uri_template, err = url.QueryUnescape(uri_template)

	if err != nil {
		return nil, nil, fmt.Errorf("Failed to unescape ?template= parameter, %w", err)
	}

	t, err := uritemplates.Parse(uri_template)

	if err != nil {
		return nil, nil, fmt.Errorf("Failed to parse URI template, %w", err)
	}

	q.Del("template")
	u.RawQuery = q.Encode()

	// findingaid://sqlite?dsn={DSN}
	// findingaid://awsdynamo/{TABLENAME}
	// findingaid://http(s)/{HOST}/{PATH}

	// Set up resolver

	var ru *url.URL

	switch u.Host {
	case "http", "https":

		path := u.Path
		path = strings.TrimLeft(path, "/")

		parts := strings.Split(path, "/")

		ru = &url.URL{}
		ru.Scheme = u.Host
		ru.Host = parts[0]

		if len(parts) > 1 {
			path = strings.Join(parts[1:], "/")
			ru.Path = fmt.Sprintf("/%s", path)
		}

		ru.RawQuery = u.RawQuery

	default:

		path := u.Path
		path = strings.TrimLeft(path, "/")

		ru = &url.URL{}
		ru.Scheme = u.Host
		ru.Host = path
		ru.RawQuery = u.RawQuery
	}

	r_uri := ru.String()

	r, err := resolver.NewResolver(ctx, r_uri)

	if err != nil {
		return nil, nil, fmt.Errorf("Failed to create resolver, %w", err)
	}

	return r, t, nil
}
