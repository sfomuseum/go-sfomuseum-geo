# go-geojson-geotag

Go package for working with GeoJSON features produced by the `Leaflet.GeotagPhoto` JavaScript package.

## Documentation

[![Go Reference](https://pkg.go.dev/badge/github.com/sfomuseum/go-geojson-geotag.svg)](https://pkg.go.dev/github.com/sfomuseum/go-geojson-geotag)

## Example

```
package main

import (
	"encoding/json"
	"github.com/sfomuseum/go-geojson-geotag"
	"io"
	"os"
)

func main() {

	body, _ := io.ReadAll(os.Stdin)

	f, _ := geotag.NewGeotagFeature(body)
	fov, _ := f.FieldOfView()

	enc := json.NewEncoder(os.Stdout)
	enc.Encode(fov)
}
```

_Error handling omitted for the sake of brevity._

## See also

* https://github.com/sfomuseum/Leaflet.GeotagPhoto
