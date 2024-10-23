default: image

ifeq ($(OS),Windows_NT)
export SHELL=cmd
DETECTED_OS=windows
EXE=.exe
else
DETECTED_OS=$(shell uname -s | tr '[:upper:]' '[:lower:]')
endif

export IMAGENAME=ghcr.io/richiesams/speedtest-exporter
TAG:=$(shell git describe --tags --dirty 2>/dev/null || echo v0.0.0)


.PHONY: vendor build

###########
# Building the binary locally
###########

build:
	go build -o build/$(DETECTED_OS)/speedtest-exporter$(EXE)


###########
# Creating a docker image
###########

image:
ifeq ($(DETECTED_OS),windows)
	cmd /C "set GORELEASER_CURRENT_TAG=$(TAG)&& goreleaser release --snapshot --clean"
else
	GORELEASER_CURRENT_TAG=$(TAG) goreleaser release --snapshot --clean
endif


###########
# Local testing
###########

run:
	docker run --rm -it \
		-p 8080:8080 \
		$(IMAGENAME):$(TAG)


###########
# Unit / functional testing
###########

test:
	go test -cover ./...


###########
# Creating a release in CI
###########

release:
	goreleaser release --clean


###########
# Miscellaneous
###########

# If you update / add any new dependencies, re-run this command to re-generate the vendor folder
vendor:
	go mod tidy
	go mod vendor
