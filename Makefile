GOVERSION=$(shell go version | awk '{print $$3;}')
ifdef TRAVIS_TAG
	TAG=$(TRAVIS_TAG)
else
	TAG="$(shell git describe --tags --always)"
endif
VERSION=$(shell echo $(TAG) | sed 's@^v@@' | sed 's@-@+@g')
PACKAGES=context database deb files gpg http query swift s3 utils
PYTHON?=python
TESTS?=
BINPATH?=$(GOPATH)/bin
RUN_LONG_TESTS?=yes

# etcd test env
ETCD_VER=v3.5.2
DOWNLOAD_URL=https://storage.googleapis.com/etcd

all: modules test bench check system-test system-test-etcd

prepare:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.43.0
	# etcd test prepare
	rm -rf /tmp/etcd-download-test/test-data && mkdir -p /tmp/etcd-download-test/test-data
	if [ ! -e /tmp/etcd-${ETCD_VER}-linux-amd64.tar.gz ]; then curl -L ${DOWNLOAD_URL}/${ETCD_VER}/etcd-${ETCD_VER}-linux-amd64.tar.gz -o /tmp/etcd-${ETCD_VER}-linux-amd64.tar.gz; fi
	tar xzvf /tmp/etcd-${ETCD_VER}-linux-amd64.tar.gz -C /tmp/etcd-download-test --strip-components=1
	/tmp/etcd-download-test/etcd --max-request-bytes '33554432' --data-dir /tmp/etcd-download-test/test-data &

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

system-test-etcd: install system/env
ifeq ($(RUN_LONG_TESTS), yes)
	if [ ! -e ~/aptly-fixture-db ]; then git clone https://github.com/aptly-dev/aptly-fixture-db.git ~/aptly-fixture-db/; fi
	if [ ! -e ~/aptly-fixture-pool ]; then git clone https://github.com/aptly-dev/aptly-fixture-pool.git ~/aptly-fixture-pool/; fi
	# TODO: maybe we can skip imgrading levledb data to etcd
	PATH=$(BINPATH)/:$(PATH) && . system/env/bin/activate && $(PYTHON) system/leveldb2etcd.py --datadir ~/aptly-fixture-db
	PATH=$(BINPATH)/:$(PATH) && . system/env/bin/activate && APTLY_DATABASE_TYPE="etcd" APTLY_DATABASE_URL="127.0.0.1:2379" APTLY_VERSION=$(VERSION) $(PYTHON) system/run.py --long $(TESTS)
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

.PHONY: man modules version release goxc
