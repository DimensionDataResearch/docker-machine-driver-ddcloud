VERSION = 0.5

default: fmt build test

fmt:
	go fmt github.com/DimensionDataResearch/docker-machine-driver-ddcloud/...

# Peform a development (current-platform-only) build.
dev: version fmt
	go build -o _bin/docker-machine-driver-ddcloud

install: dev
	go install

# Perform a full (all-platforms) build.
build: version build-windows64 build-linux64 build-mac64

build-windows64:
	GOOS=windows GOARCH=amd64 go build -o _bin/windows-amd64/docker-machine-driver-ddcloud.exe

build-linux64:
	GOOS=linux GOARCH=amd64 go build -o _bin/linux-amd64/docker-machine-driver-ddcloud

build-mac64:
	GOOS=darwin GOARCH=amd64 go build -o _bin/darwin-amd64/docker-machine-driver-ddcloud

# Produce archives for a GitHub release.
dist: build
	zip -9 _bin/windows-amd64.zip _bin/windows-amd64/docker-machine-driver-ddcloud.exe
	zip -9 _bin/linux-amd64.zip _bin/linux-amd64/docker-machine-driver-ddcloud
	zip -9 _bin/darwin-amd64.zip _bin/darwin-amd64/docker-machine-driver-ddcloud

test: fmt
	go test -v github.com/DimensionDataResearch/docker-machine-driver-ddcloud/...

version:
	echo "package main\n\n// DriverVersion is the current version of the CloudControl driver for Docker Machine.\nconst DriverVersion = \"v${VERSION} (`git rev-parse HEAD`)\"" > ./version-info.go
