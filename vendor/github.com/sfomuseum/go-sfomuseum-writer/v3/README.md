# go-sfomuseum-writer

Common methods for writing SFO Museum (Who's On First) documents with `whosonfirst/go-writer.Writer` instances.

## Documentation

[![Go Reference](https://pkg.go.dev/badge/github.com/sfomuseum/go-sfomuseum-writer.svg)](https://pkg.go.dev/github.com/sfomuseum/go-sfomuseum-writer)

## Examples

_Note that error handling has been removed for the sake of brevity._

### WriteFeature

```
import (
	"context"
	"github.com/paulmach/orb/geojson"
	"github.com/whosonfirst/go-writer"	
	sfom_writer "github.com/sfomuseum/go-sfomuseum-writer"	
)

func main() {

	ctx := context.Background()
	wr, _ := writer.NewWriter(ctx, "stdout://")
	
	for _, feature_path := range flag.Args() {
	
		fh, _ := os.Open(feature_path)
		body, _ := io.ReadAll(fh)
		f, _ := geojson.UnmarshalFeature(body)

		sfom_writer.WriteFeature(ctx, wr, f)
	}
```

### WriteBytes

```
import (
	"context"
	"github.com/whosonfirst/go-writer"	
	sfom_writer "github.com/sfomuseum/go-sfomuseum-writer"
	"io"
)

func main() {

	ctx := context.Background()
	wr, _ := writer.NewWriter(ctx, "stdout://")
	
	for _, feature_path := range flag.Args() {
	
		fh, _ := os.Open(feature_path)
		body, _ := io.ReadAll(fh)
		
		sfom_writer.WriteBytes(ctx, wr, body)
	}
```

## See also

* https://github.com/whosonfirst/go-writer
* https://github.com/whosonfirst/go-whosonfirst-writer
* https://github.com/whosonfirst/go-whosonfirst-export
* https://github.com/sfomuseum/go-sfomuseum-export