GOMOD=$(shell test -f "go.work" && echo "readonly" || echo "vendor")
LDFLAGS=-s -w

vuln:
	govulncheck ./...

cli:
	rm -f bin/*
	go build -mod $(GOMOD) -ldflags="$(LDFLAGS)" -o bin/update-depiction cmd/update-depiction/main.go
	go build -mod $(GOMOD) -ldflags="$(LDFLAGS)" -o bin/build-depiction-update cmd/build-depiction-update/main.go
	go build -mod $(GOMOD) -ldflags="$(LDFLAGS)" -o bin/assign-georeferences cmd/assign-georeferences/main.go
