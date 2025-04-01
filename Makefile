GOMOD=$(shell test -f "go.work" && echo "readonly" || echo "vendor")
LDFLAGS=-s -w

CWD=$(shell pwd)

vuln:
	govulncheck ./...

cli:
	rm -f bin/*
	go build -mod $(GOMOD) -ldflags="$(LDFLAGS)" -o bin/update-depiction cmd/update-depiction/main.go
	go build -mod $(GOMOD) -ldflags="$(LDFLAGS)" -o bin/build-depiction-update cmd/build-depiction-update/main.go
	go build -mod $(GOMOD) -ldflags="$(LDFLAGS)" -o bin/assign-georeferences cmd/assign-georeferences/main.go

# subject (object):
# https://collection.sfomuseum.org/objects/1897902471/
# https://static.sfomuseum.org/data/189/790/247/1/1897902471.geojson
#
# depiction (image):
# https://collection.sfomuseum.org/images/1897903961/
# https://github.com/sfomuseum-data/sfomuseum-data-media-collection/blob/main/data/189/790/396/1/1897903961.geojson
#
# reference (bangkok):
# https://spelunker.whosonfirst.org/id/102025263

debug-georef-photo:
	go run -mod $(GOMOD) cmd/assign-georeferences/main.go \
		-depiction-reader-uri repo://$(CWD)/fixtures/sfomuseum-data-media-collection \
		-depiction-writer-uri stdout:// \
		-subject-reader-uri repo://$(CWD)/fixtures/sfomuseum-data-collection \
		-subject-writer-uri stdout:// \
		-depiction-id 1897903961 \
		-reference sfomuseum:depicts=102025263


# subject (object):
# https://collection.sfomuseum.org/objects/1511907389
# https://static.sfomuseum.org/data/151/190/738/9/1511907389.geojson
#
# depiction (image):
# https://collection.sfomuseum.org/images/1527829811/
# https://static.sfomuseum.org/data/152/782/981/1/1527829811.geojson
#
# reference (noumea)
# https://spelunker.whosonfirst.org/id/890413117
# reference (sydney)
# https://spelunker.whosonfirst.org/id/101932003

debug-georef-flightcover:
	go run -mod $(GOMOD) cmd/assign-georeferences/main.go \
		-depiction-reader-uri repo://$(CWD)/fixtures/sfomuseum-data-media-collection \
		-depiction-writer-uri repo://$(CWD)/fixtures/debug \
		-subject-reader-uri repo://$(CWD)/fixtures/sfomuseum-data-collection \
		-subject-writer-uri repo://$(CWD)/fixtures/debug \
		-depiction-id 1527829811 \
		-reference sfomuseum:flightcover_to=890413117 \
		-reference sfomuseum:flightcover_from=101932003 \
		-verbose
