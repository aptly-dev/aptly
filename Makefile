GOVERSION=$(shell go version | awk '{print $$3;}')
GOPATH=$(shell go env GOPATH)
TAG="$(shell git describe --tags --always)"
VERSION=$(shell echo $(TAG) | sed 's@^v@@' | sed 's@-@+@g' | tr -d '\n')
PACKAGES=context database deb files gpg http query swift s3 utils
PYTHON?=python3
TESTS?=
BINPATH?=$(GOPATH)/bin
RUN_LONG_TESTS?=yes
GOLANGCI_LINT_VERSION=v1.56.2
COVERAGE_DIR?=$(shell mktemp -d)
# Uncomment to update test outputs
# CAPTURE := "--capture"

all: modules test bench check system-test

# Self-documenting Makefile
# https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
help:  ## Print this help
	@grep -E '^[a-zA-Z][a-zA-Z0-9_-]*:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

prepare:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin $(GOLANGCI_LINT_VERSION)

modules:
	go mod download
	go mod verify
	go mod tidy -v

dev:
	PATH=$(BINPATH)/:$(PATH)
	go get github.com/laher/goxc
	go install github.com/laher/goxc

check: system/env
ifeq ($(RUN_LONG_TESTS), yes)
	system/env/bin/flake8
endif

install:
	@echo Building aptly ...
	go generate
	@echo go install -v
	@out=`mktemp`; if ! go install -v > $$out 2>&1; then cat $$out; rm -f $$out; echo "\nBuild failed\n"; exit 1; else rm -f $$out; fi

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
	PATH=$(BINPATH)/:$(PATH) && . system/env/bin/activate && APTLY_VERSION=$(VERSION) FORCE_COLOR=1 $(PYTHON) system/run.py --long $(TESTS) --coverage-dir $(COVERAGE_DIR) $(CAPTURE)
endif

docker-test: install
	@echo Building aptly.test ...
	@rm -f aptly.test
	go test -v -coverpkg="./..." -c -tags testruncli
	@echo Running python tests ...
	export PATH=$(BINPATH)/:$(PATH); \
	export APTLY_VERSION=$(VERSION); \
	$(PYTHON) system/run.py --long $(TESTS) --coverage-dir $(COVERAGE_DIR) $(CAPTURE) $(TEST)

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

man:  ## Create man pages
	make -C man

version:  ## Print aptly version
	@echo $(VERSION)

docker-build-system-tests:  ## Build system-test docker image
	docker build -f system/Dockerfile . -t aptly-system-test

docker-unit-tests:  ## Run unit tests in docker container
	docker run -it --rm -v ${PWD}:/app aptly-system-test go test -v ./... -gocheck.v=true

docker-system-tests:  ## Run system tests in docker container (add TEST=t04_mirror to run only specific tests)
	docker run -it --rm -v ${PWD}:/app aptly-system-test /app/system/run-system-tests $(TEST)

golangci-lint:  ## Run golangci-line in docker container
	docker run -it --rm -v ~/.cache/golangci-lint/$(GOLANGCI_LINT_VERSION):/root/.cache -v ${PWD}:/app -w /app golangci/golangci-lint:$(GOLANGCI_LINT_VERSION) golangci-lint run

flake8:
	flake8 system

.PHONY: help man modules version release goxc docker-build docker-system-tests
