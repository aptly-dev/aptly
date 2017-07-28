GOVERSION=$(shell go version | awk '{print $$3;}')
VERSION=$(shell git describe --tags | sed 's@^v@@' | sed 's@-@+@g')
PACKAGES=context database deb files gpg http query swift s3 utils
PYTHON?=python
TESTS?=
BINPATH?=$(GOPATH)/bin

ifeq ($(GOVERSION), devel)
TRAVIS_TARGET=coveralls
else
TRAVIS_TARGET=test
endif

all: test check system-test

prepare:
	go get -u github.com/mattn/goveralls
	go get -u github.com/axw/gocov/gocov
	go get -u golang.org/x/tools/cmd/cover
	go get -u github.com/alecthomas/gometalinter
	gometalinter --install

dev:
	go get -u github.com/golang/dep/...
	go get -u github.com/laher/goxc

coverage.out:
	rm -f coverage.*.out
	for i in $(PACKAGES); do go test -coverprofile=coverage.$$i.out -covermode=count ./$$i; done
	echo "mode: count" > coverage.out
	grep -v -h "mode: count" coverage.*.out >> coverage.out
	rm -f coverage.*.out

coverage: coverage.out
	go tool cover -html=coverage.out
	rm -f coverage.out

check: system/env
	if [ -x travis_wait ]; then \
		travis_wait gometalinter --config=linter.json ./...; \
	else \
		gometalinter --config=linter.json ./...; \
	fi
	. system/env/bin/activate && flake8 --max-line-length=200 --exclude=system/env/ system/

install:
	go install -v -ldflags "-X main.Version=$(VERSION)"

system/env: system/requirements.txt
	rm -rf system/env
	virtualenv system/env
	system/env/bin/pip install -r system/requirements.txt

system-test: install system/env
	if [ ! -e ~/aptly-fixture-db ]; then git clone https://github.com/aptly-dev/aptly-fixture-db.git ~/aptly-fixture-db/; fi
	if [ ! -e ~/aptly-fixture-pool ]; then git clone https://github.com/aptly-dev/aptly-fixture-pool.git ~/aptly-fixture-pool/; fi
	. system/env/bin/activate && APTLY_VERSION=$(VERSION) PATH=$(BINPATH)/:$(PATH) $(PYTHON) system/run.py --long $(TESTS)

travis: $(TRAVIS_TARGET) check system-test

test:
	go test -v `go list ./... | grep -v vendor/` -gocheck.v=true

coveralls: coverage.out
	$(BINPATH)/goveralls -service travis-ci.org -coverprofile=coverage.out -repotoken=$(COVERALLS_TOKEN)

mem.png: mem.dat mem.gp
	gnuplot mem.gp
	open mem.png

goxc:
	rm -rf root/
	mkdir -p root/usr/share/man/man1/ root/etc/bash_completion.d
	cp man/aptly.1 root/usr/share/man/man1
	cp bash_completion.d/aptly root/etc/bash_completion.d
	gzip root/usr/share/man/man1/aptly.1
	goxc -pv=$(VERSION) -max-processors=4 $(GOXC_OPTS)

man:
	make -C man

version:
	@echo $(VERSION)

.PHONY: coverage.out man version
