package github

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

type Action int

func (a Action) String() string {

	switch a {
	case GeoreferenceAction:
		return "georeference"
	case GeotagAction:
		return "geotag"
	default:
		return ""
	}
}

const (
	GeoreferenceAction = Action(iota)
	GeotagAction
)

type UpdateWriterURIOptions struct {
	WhosOnFirstId int64
	Author        string
	Action        Action
}

func UpdateWriterURI(ctx context.Context, opts *UpdateWriterURIOptions, writer_uri string) (string, error) {

	wr_u, err := url.Parse(writer_uri)

	if err != nil {
		return "", fmt.Errorf("Failed to parse URI, %w", err)
	}

	switch wr_u.Scheme {

	case "githubapi":

		update_msg := fmt.Sprintf("[%s] update %s data for ", opts.Author, opts.Action)
		update_msg = update_msg + "%s" // I wish I knew how to include a literal '%s' in fmt.Sprintf...

		wr_q := wr_u.Query()

		wr_q.Del("new")
		wr_q.Del("update")

		wr_q.Set("new", update_msg)
		wr_q.Set("update", update_msg)

		// branch...
		wr_u.RawQuery = wr_q.Encode()

	case "githubapi-pr":

		title := fmt.Sprintf("[%s] update %s data for %d", opts.Author, opts.Action, opts.WhosOnFirstId)
		description := title

		now := time.Now()
		ts := now.Unix()

		branch := fmt.Sprintf("%s-%d-%s-%d", opts.Author, ts, opts.Action, opts.WhosOnFirstId)

		wr_q := wr_u.Query()

		wr_q.Del("pr-branch")
		wr_q.Del("pr-title")
		wr_q.Del("pr-description")

		wr_q.Set("pr-branch", branch)
		wr_q.Set("pr-title", title)
		wr_q.Set("pr-description", description)

		// branch...
		wr_u.RawQuery = wr_q.Encode()
	}

	return wr_u.String(), nil
}
