package georeference

import (
	"fmt"
	"log/slog"
	"regexp"
	"strings"
)

var re_not_alpha = regexp.MustCompile(`[^a-z0-9]`)
var re_underbars = regexp.MustCompile(`__{1,}`)

const GEOREF_ALT_PREFIX string = "georef_"

func DeriveAltLabelFromReference(r *Reference) string {

	prop_label := r.Label
	alt_label := r.AltLabel

	if alt_label == "" {
		alt_label = DeriveAltLabel(prop_label)
		slog.Debug("Alt label derived from property label", "alt label", alt_label)
	}

	if !strings.HasPrefix(alt_label, GEOREF_ALT_PREFIX) {
		alt_label = fmt.Sprintf("%s%s", GEOREF_ALT_PREFIX, alt_label)
		slog.Debug("Alt label assigned 'georef_' prefix", "alt label", alt_label)
	}

	return alt_label
}

func DeriveAltLabel(raw string) string {

	underbar := []byte("_")

	v := []byte(strings.ToLower(raw))

	v = re_not_alpha.ReplaceAll(v, underbar)
	v = re_underbars.ReplaceAll(v, underbar)

	return string(v)
}
