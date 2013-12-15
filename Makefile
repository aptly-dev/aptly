GOVERSION=$(shell go version | awk '{print $$3;}')

ifeq ($(TRAVIS), true)
GOVERALLS?=$(HOME)/gopath/bin/goveralls
else
GOVERALLS?=goveralls
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
	env
	go get -d -v ./...
	go get launchpad.net/gocheck

cover-prepare:
	go get github.com/golang/lint/golint
	go get github.com/matm/gocov-html
	go get github.com/mattn/goveralls
	go get github.com/axw/gocov/gocov
	go get code.google.com/p/go.tools/cmd/cover

coverage:
	gocov test ./... | gocov-html > coverage.html
	open coverage.html

check:
	go tool vet -all=true .
	golint .

travis: $(TRAVIS_TARGET)

test:
	go test -v ./...

coveralls:
	@$(GOVERALLS) -service travis-ci.org -package="./..." $(COVERALLS_TOKEN)

.PHONY: prepare cover-prepare coverage check test coveralls travis