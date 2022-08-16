package georeference

type Reference struct {
	Id       int64  `json:"id"`
	Property string `json:"property"`
	AltLabel string `json:"alt_label"`
}
