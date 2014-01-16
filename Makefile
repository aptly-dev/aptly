GOVERSION=$(shell go version | awk '{print $$3;}')

ifeq ($(TRAVIS), true)
GOVERALLS?=$(HOME)/gopath/bin/goveralls
SRCPATH?=$(HOME)/gopath/src
else
GOVERALLS?=goveralls
SRCPATH?=$(GOPATH)/src
endif

ifeq ($(GOVERSION), go1.2)
TRAVIS_TARGET=coveralls
PREPARE_LIST=cover-prepare
else
TRAVIS_TARGET=test
PREPARE_LIST=
endif

all: test check

prepare: $(PREPARE_LIST)
	go get -d -v ./...
	go get launchpad.net/gocheck

cover-prepare:
	go get github.com/golang/lint/golint
	go get github.com/mattn/goveralls
	go get github.com/axw/gocov/gocov
	go get code.google.com/p/go.tools/cmd/cover

coverage.out:
	go test -coverprofile=coverage.debian.out -covermode=count ./debian
	go test -coverprofile=coverage.utils.out -covermode=count ./utils
	go test -coverprofile=coverage.database.out -covermode=count ./database
	echo "mode: count" > coverage.out
	grep -v -h "mode: count" coverage.*.out >> coverage.out

coverage: coverage.out
	go tool cover -html=coverage.out
	rm -f coverage.out

check:
	go tool vet -all=true .
	golint .

travis: $(TRAVIS_TARGET)

test:
	go test -v ./... -gocheck.v=true

coveralls: coverage.out
	@$(GOVERALLS) -service travis-ci.org -coverprofile=coverage.out $(COVERALLS_TOKEN)

.PHONY: coverage.out