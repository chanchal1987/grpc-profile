# Check for required command tools to build or stop immediately
EXECUTABLES = git go find pwd
K := $(foreach exec,$(EXECUTABLES),\
        $(if $(shell which $(exec)),some string,$(error "No $(exec) in PATH)))

ROOT_DIR:=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

BINARY=grpc-profile
MAIN=cmd/grpc-profile
VERSION=0.0.2
BUILD=`git rev-parse HEAD`
PLATFORMS=darwin freebsd linux openbsd windows
ARCHITECTURES=386 amd64 arm arm64
TARGET=$(if $(GOBIN),$(GOBIN),${ROOT_DIR}/bin)
#TARGET=${ROOT_DIR}/bin

# Setup linker flags option for build that interoperate with variable names in src code
LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.Build=${BUILD} -s -w"

default: build

all: clean build install

build:
	cd ${MAIN}; go build ${LDFLAGS} -o ${TARGET}/${BINARY}

build_all:
	$(foreach GOOS, $(PLATFORMS),\
	$(foreach GOARCH, $(ARCHITECTURES), $(shell export GOOS=$(GOOS); export GOARCH=$(GOARCH); cd ${MAIN}; go build -v -o ${TARGET}/$(BINARY)-$(GOOS)-$(GOARCH))))

install:
	go install ${LDFLAGS}

# Remove only what we've created
clean:
	find ${ROOT_DIR} -name '${BINARY}[-?][a-zA-Z0-9]*[-?][a-zA-Z0-9]*' -delete

.PHONY: clean install build build_all all