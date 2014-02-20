GOVERSION=$(shell go version | awk '{print $$3;}')

ifeq ($(TRAVIS), true)
GOVERALLS?=$(HOME)/gopath/bin/goveralls
SRCPATH?=$(HOME)/gopath/src
BINPATH=$(HOME)/gopath/bin
else
GOVERALLS?=goveralls
SRCPATH?=$(GOPATH)/src
BINPATH?=$(GOPATH)/bin
endif

ifeq ($(GOVERSION), go1.2)
TRAVIS_TARGET=coveralls
PREPARE_LIST=cover-prepare
else
TRAVIS_TARGET=test
PREPARE_LIST=
endif

all: test check system-test

prepare: $(PREPARE_LIST)
	go get -d -v ./...
	go get launchpad.net/gocheck

cover-prepare:
	go get github.com/golang/lint/golint
	go get github.com/mattn/goveralls
	go get github.com/axw/gocov/gocov
	go get code.google.com/p/go.tools/cmd/cover

coverage.out:
	rm -f coverage.*.out
	for i in database debian files http utils; do go test -coverprofile=coverage.$$i.out -covermode=count ./$$i; done
	echo "mode: count" > coverage.out
	grep -v -h "mode: count" coverage.*.out >> coverage.out
	rm -f coverage.*.out

coverage: coverage.out
	go tool cover -html=coverage.out
	rm -f coverage.out

check:
	go tool vet -all=true .
	golint .

system-test:
ifeq ($(GOVERSION), go1.2)
	if [ ! -e ~/aptly-fixture-db ]; then git clone https://github.com/aptly-dev/aptly-fixture-db.git ~/aptly-fixture-db/; fi
endif
	if [ ! -e ~/aptly-fixture-pool ]; then git clone https://github.com/aptly-dev/aptly-fixture-pool.git ~/aptly-fixture-pool/; fi
	go install
	PATH=$(BINPATH):$(PATH) python system/run.py --long

travis: $(TRAVIS_TARGET) system-test

test:
	go test -v ./... -gocheck.v=true

coveralls: coverage.out
	@$(GOVERALLS) -service travis-ci.org -coverprofile=coverage.out -repotoken $(COVERALLS_TOKEN)

.PHONY: coverage.out