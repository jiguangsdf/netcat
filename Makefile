#!/usr/bin/make -f

GOCMD = go
GOBUILD = $(GOCMD) build
GOMOD = $(GOCMD) mod
VERSION := $(shell echo $(shell git describe --always) | sed 's/^v//')
COMMIT := $(shell git log -1 --format='%H')
BUILDDIR ?= $(CURDIR)/build

export GO111MODULE = on

build_tags = netgo

ifeq ($(WITH_CLEVELDB),yes)
  build_tags += gcc
endif
build_tags += $(BUILD_TAGS)
build_tags := $(strip $(build_tags))

whitespace :=
whitespace += $(whitespace)
comma := ,
build_tags_comma_sep := $(subst $(whitespace),$(comma),$(build_tags))

NAME=netcat
PACKAGENAME=main

### -X package.name
ldflags = -X $(PACKAGENAME).Name=$(NAME) \
		  -X $(PACKAGENAME).Version=$(VERSION) \
		  -X $(PACKAGENAME).Commit=$(COMMIT) \
		  -X $(PACKAGENAME).BuildTags=$(build_tags_comma_sep) \

BUILD_FLAGS := -tags "$(build_tags)" -ldflags '$(ldflags) -s -w' 

GOMODCACHE := $(shell go env |grep GOMODCACHE|sed 's/GOMODCACHE=//'|tr -d '"')
BUILD_FLAGS += -gcflags='all=-trimpath=${PWD}' -asmflags='all=-trimpath=${PWD}'
BUILD_FLAGS += -gcflags='all=-trimpath=$(GOMODCACHE)' -asmflags='all=-trimpath=$(GOMODCACHE)'

# darwin amd64
build: gofmt clean go.sum
	mkdir -p $(BUILDDIR)
	go build -mod=readonly $(BUILD_FLAGS) -o $(BUILDDIR)/$(NAME)

# linux amd64
build-linux:
	GOOS=linux GOARCH=amd64 $(MAKE) build

# linux arm64
build-linux-arm64:
	GOOS=linux GOARCH=arm64 $(MAKE) build

# windows
build-windows: gofmt clean go.sum
	mkdir -p $(BUILDDIR)
	GOOS=windows GOARCH=amd64 go build -mod=readonly $(BUILD_FLAGS) -o $(BUILDDIR)/$(NAME)

install: go.sum
	go install -mod=readonly -v $(BUILD_FLAGS)

install-debug: go.sum
	go install -mod=readonly $(BUILD_FLAGS)

clean:
	rm -rf build/

gofmt:
	find . -name '*.go' -type f -not -path "./vendor*" -not -path "*.git*" | xargs gofmt -w -s

go.sum: go.mod
	@go mod verify
	@go mod tidy

.PHONY: build