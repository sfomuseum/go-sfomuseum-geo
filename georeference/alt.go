package georeference

import (
	"regexp"
	"strings"
)

var re_not_alpha = regexp.MustCompile(`[^a-z0-9]`)
var re_underbars = regexp.MustCompile(`__{1,}`)

func DeriveAltLabel(raw string) string {

	underbar := []byte("_")

	v := []byte(strings.ToLower(raw))

	v = re_not_alpha.ReplaceAll(v, underbar)
	v = re_underbars.ReplaceAll(v, underbar)

	return string(v)
}
