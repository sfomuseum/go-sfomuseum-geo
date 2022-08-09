cli:
	rm -f bin/*
	go build -mod vendor -o bin/update-depiction cmd/update-depiction/main.go
	go build -mod vendor -o bin/build-update cmd/build-update/main.go
	go build -mod vendor -o bin/server cmd/server/main.go

dev:
	go run -mod vendor cmd/server/main.go -server-uri http://localhost:8083 -depiction-writer-uri stdout:// -subject-writer-uri stdout://

docker:
	docker build -t sfomuseum-geotag-server .

lambda:
	@make lambda-server

lambda-server:
	if test -f main; then rm -f main; fi
	if test -f server.zip; then rm -f server.zip; fi
	GOOS=linux go build -mod vendor -o main cmd/server/main.go
	zip server.zip main
	rm -f main

godoc:
	godoc -http=:6060

local-scan:
	/usr/local/bin/sonar-scanner/bin/sonar-scanner -Dsonar.projectKey=go-sfomuseum-geotag -Dsonar.sources=. -Dsonar.host.url=http://localhost:9000 -Dsonar.login=$(TOKEN)
