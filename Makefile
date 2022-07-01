GOVERSION=$(shell go version | awk '{print $$3;}')
TAG="$(shell git describe --tags --always)"
VERSION=$(shell echo $(TAG) | sed 's@^v@@' | sed 's@-@+@g' | tr -d '\n')
PACKAGES=context database deb files gpg http query swift s3 utils
PYTHON?=python3
TESTS?=
BINPATH?=$(GOPATH)/bin
RUN_LONG_TESTS?=yes
COVERAGE_DIR?=$(shell mktemp -d)

all: modules test bench check system-test

prepare:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.50.1

modules:
	go mod download
	go mod verify
	go mod tidy -v

dev:
	go get -u github.com/laher/goxc

check: system/env
ifeq ($(RUN_LONG_TESTS), yes)
	golangci-lint run
	system/env/bin/flake8
endif

install:
	go generate
	go install -v

system/env: system/requirements.txt
ifeq ($(RUN_LONG_TESTS), yes)
	rm -rf system/env
	$(PYTHON) -m venv system/env
	system/env/bin/pip install -r system/requirements.txt
endif

system-test: install system/env
ifeq ($(RUN_LONG_TESTS), yes)
	go generate
	go test -v -coverpkg="./..." -c -tags testruncli
	if [ ! -e ~/aptly-fixture-db ]; then git clone https://github.com/aptly-dev/aptly-fixture-db.git ~/aptly-fixture-db/; fi
	if [ ! -e ~/aptly-fixture-pool ]; then git clone https://github.com/aptly-dev/aptly-fixture-pool.git ~/aptly-fixture-pool/; fi
	PATH=$(BINPATH)/:$(PATH) && . system/env/bin/activate && APTLY_VERSION=$(VERSION) $(PYTHON) system/run.py --long $(TESTS) --coverage-dir $(COVERAGE_DIR)
endif

test:
	go test -v ./... -gocheck.v=true -coverprofile=unit.out

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
	go generate
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

.PHONY: man modules version release goxc
