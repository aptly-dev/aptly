GOVERSION=$(shell go version | awk '{print $$3;}')
GOPATH=$(shell go env GOPATH)
TAG="$(shell git describe --tags --always)"
VERSION=$(shell echo $(TAG) | sed 's@^v@@' | sed 's@-@+@g' | tr -d '\n')
PACKAGES=context database deb files gpg http query swift s3 utils
PYTHON?=python3
TESTS?=
BINPATH?=$(GOPATH)/bin
RUN_LONG_TESTS?=yes
GOLANGCI_LINT_VERSION=v1.54.1  # version supporting go 1.19
COVERAGE_DIR?=$(shell mktemp -d)
# Uncomment to update test outputs
# CAPTURE := "--capture"

# Self-documenting Makefile
# https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
help:  ## Print this help
	@grep -E '^[a-zA-Z][a-zA-Z0-9_-]*:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

all: modules test bench check system-test

modules:  ## Install go module dependencies
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

system-test: install system/env  ## Run system tests in github CI
ifeq ($(RUN_LONG_TESTS), yes)
	go generate
	test -d /srv/etcd || system/t13_etcd/install-etcd.sh
	system/t13_etcd/start-etcd.sh &
	go test -v -coverpkg="./..." -c -tags testruncli
	kill `cat /tmp/etcd.pid`

	if [ ! -e ~/aptly-fixture-db ]; then git clone https://github.com/aptly-dev/aptly-fixture-db.git ~/aptly-fixture-db/; fi
	if [ ! -e ~/aptly-fixture-pool ]; then git clone https://github.com/aptly-dev/aptly-fixture-pool.git ~/aptly-fixture-pool/; fi
	cd /home/runner; curl -O http://repo.aptly.info/system-tests/etcd.db
	PATH=$(BINPATH)/:$(PATH) && . system/env/bin/activate && APTLY_VERSION=$(VERSION) FORCE_COLOR=1 $(PYTHON) system/run.py --long $(TESTS) --coverage-dir $(COVERAGE_DIR) $(CAPTURE)
endif

docker-test: install  ## Run system tests
	@echo Building aptly.test ...
	@rm -f aptly.test
	go test -v -coverpkg="./..." -c -tags testruncli
	@echo Running python tests ...
	@test -e aws.creds && . ./aws.creds; \
	export PATH=$(BINPATH)/:$(PATH); \
	export APTLY_VERSION=$(VERSION); \
	$(PYTHON) system/run.py --long $(TESTS) --coverage-dir $(COVERAGE_DIR) $(CAPTURE) $(TEST)

test:  ## Run unit tests
	@test -d /srv/etcd || system/t13_etcd/install-etcd.sh
	@system/t13_etcd/start-etcd.sh &
	@echo Running go test
	go test -v ./... -gocheck.v=true -coverprofile=unit.out
	@kill `cat /tmp/etcd.pid`

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

build:  ## Build aptly
	go mod tidy
	go generate
	go build -o build/aptly

docker-build-aptly-dev:  ## Build aptly-dev docker image
	@docker build -f system/Dockerfile . -t aptly-dev

docker-build:  ## Build aptly in docker container
	@docker run -it --rm -v ${PWD}:/app aptly-dev /app/system/run-aptly-cmd make build

docker-aptly:  ## Build and run aptly commands in docker container
	@docker run -it --rm -v ${PWD}:/app aptly-dev /app/system/run-aptly-cmd

docker-unit-tests:  ## Run unit tests in docker container
	@docker run -it --rm -v ${PWD}:/app aptly-dev /app/system/run-unit-tests

docker-system-tests:  ## Run system tests in docker container (add TEST=t04_mirror to run only specific tests)
	@docker run -it --rm -v ${PWD}:/app aptly-dev /app/system/run-system-tests $(TEST)

docker-lint:  ## Run golangci-lint in docker container
	@docker run -it --rm -v ${PWD}:/app -e GOLANGCI_LINT_VERSION=$(GOLANGCI_LINT_VERSION) aptly-dev /app/system/run-golangci-lint

flake8:  ## run flak8 on system tests
	flake8 system

clean:  ## remove local build and module cache
	test -f .go/ && chmod u+w -R .go/; rm -rf .go/
	rm -rf build/

.PHONY: help man modules version release goxc docker-build-aptly-dev docker-system-tests docker-unit-tests docker-lint docker-build build docker-aptly clean
