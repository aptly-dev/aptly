GOVERSION=$(shell go version | awk '{print $$3;}')
ifdef TRAVIS_TAG
	TAG=$(TRAVIS_TAG)
else
	TAG="$(shell git describe --tags)"
endif
VERSION=$(shell echo $(TAG) | sed 's@^v@@' | sed 's@-@+@g')
PACKAGES=context database deb files gpg http query swift s3 utils
PYTHON?=python
TESTS?=
BINPATH?=$(GOPATH)/bin
RUN_LONG_TESTS?=yes

all: test bench check system-test

prepare:
	curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.17.1

dev:
	go get -u github.com/golang/dep/...
	go get -u github.com/laher/goxc

check: system/env
ifeq ($(RUN_LONG_TESTS), yes)
	golangci-lint run
	. system/env/bin/activate && flake8 --max-line-length=200 --exclude=system/env/ system/
endif

install:
	go install -v -ldflags "-X main.Version=$(VERSION)"

system/env: system/requirements.txt
ifeq ($(RUN_LONG_TESTS), yes)
	rm -rf system/env
	virtualenv system/env
	system/env/bin/pip install -r system/requirements.txt
endif

system-test: install system/env
ifeq ($(RUN_LONG_TESTS), yes)
	if [ ! -e ~/aptly-fixture-db ]; then git clone https://github.com/aptly-dev/aptly-fixture-db.git ~/aptly-fixture-db/; fi
	if [ ! -e ~/aptly-fixture-pool ]; then git clone https://github.com/aptly-dev/aptly-fixture-pool.git ~/aptly-fixture-pool/; fi
	PATH=$(BINPATH)/:$(PATH) && . system/env/bin/activate && APTLY_VERSION=$(VERSION) $(PYTHON) system/run.py --long $(TESTS)
endif

test:
	go test -v ./... -gocheck.v=true -race -coverprofile=coverage.txt -covermode=atomic

bench:
	go test -v ./deb -run=nothing -bench=. -benchmem

mem.png: mem.dat mem.gp
	gnuplot mem.gp
	open mem.png

goxc: dev
	rm -rf root/
	mkdir -p root/usr/share/man/man1/ root/etc/bash_completion.d/ root/usr/share/zsh/vendor-completions/
	cp man/aptly.1 root/usr/share/man/man1
	cp completion.d/aptly root/etc/bash_completion.d/
	cp completion.d/_aptly root/usr/share/zsh/vendor-completions/
	gzip root/usr/share/man/man1/aptly.1
	goxc -pv=$(VERSION) -max-processors=2 $(GOXC_OPTS)

release: GOXC_OPTS=-tasks-=bintray,go-vet,go-test,rmbin
release: goxc
	rm -rf build/
	mkdir -p build/
	mv xc-out/$(VERSION)/aptly_$(VERSION)_* build/

man:
	make -C man

version:
	@echo $(VERSION)

.PHONY: man version release goxc
