cli:
	rm -f bin/*
	go build -mod vendor -o bin/update-depiction cmd/update-depiction/main.go
	go build -mod vendor -o bin/build-depiction-update cmd/build-depiction-update/main.go
