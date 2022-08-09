# go-whosonfirst-writer

Common methods for writing Who's On First documents.

## Documentation

[![Go Reference](https://pkg.go.dev/badge/github.com/whosonfirst/go-whosonfirst-writer.svg)](https://pkg.go.dev/github.com/whosonfirst/go-whosonfirst-writer)

## Examples

_Note that error handling has been removed for the sake of brevity._

### WriteFeature

```
import (
	"context"
	"flag"
	"github.com/paulmach/orb/geojson"
	"io"
	wof_writer "github.com/whosonfirst/go-whosonfirst-writer"
	"github.com/whosonfirst/go-writer"			
)

func main() {

	flag.Parse()

	ctx := context.Background()
	wr, _ := writer.NewWriter(ctx, "stdout://")
	
	for _, feature_path := range flag.Args() {
	
		r, _ := os.Open(feature_path)
		body, _ := io.ReadAll(r)		    
		f, _ := geojson.UnmarshalFeature(body)

		wof_writer.WriteFeature(ctx, wr, f)
	}
```

### WriteBytes

```
import (
	"context"
	"flag"
	"github.com/whosonfirst/go-writer"	
	wof_writer "github.com/whosonfirst/go-whosonfirst-writer"
	"io"
)

func main() {

	flag.Parse()

	ctx := context.Background()
	wr, _ := writer.NewWriter(ctx, "stdout://")
	
	for _, feature_path := range flag.Args() {
	
		fh, _ := os.Open(feature_path)
		body, _ := io.ReadAll(fh)
		
		wof_writer.WriteBytes(ctx, wr, body)
	}
```

## See also

* https://github.com/whosonfirst/go-writer