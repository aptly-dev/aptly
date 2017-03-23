GOVERSION=$(shell go version | awk '{print $$3;}')
PACKAGES=context database deb files http query swift s3 utils
ALL_PACKAGES=api aptly context cmd console database deb files http query swift s3 utils
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

check:
	gometalinter --vendor --vendored-linters --config=linter.json ./...

install:
	go install -v

system-test: install
	if [ ! -e ~/aptly-fixture-db ]; then git clone https://github.com/aptly-dev/aptly-fixture-db.git ~/aptly-fixture-db/; fi
	if [ ! -e ~/aptly-fixture-pool ]; then git clone https://github.com/aptly-dev/aptly-fixture-pool.git ~/aptly-fixture-pool/; fi
	PATH=$(BINPATH)/:$(PATH) $(PYTHON) system/run.py --long $(TESTS)

travis: $(TRAVIS_TARGET) check system-test

test:
	go test -v `go list ./... | grep -v vendor/` -gocheck.v=true

coveralls: coverage.out
	$(BINPATH)/goveralls -service travis-ci.org -coverprofile=coverage.out -repotoken=$(COVERALLS_TOKEN)

mem.png: mem.dat mem.gp
	gnuplot mem.gp
	open mem.png

src-package:
	rm -rf aptly-$(VERSION)
	mkdir -p aptly-$(VERSION)/src/github.com/smira/aptly/
	cd aptly-$(VERSION)/src/github.com/smira/ && git clone https://github.com/smira/aptly && cd aptly && git checkout v$(VERSION)
	mkdir -p aptly-$(VERSION)/bash_completion.d
	(cd aptly-$(VERSION)/bash_completion.d && wget https://raw.github.com/aptly-dev/aptly-bash-completion/$(VERSION)/aptly)
	tar cyf aptly-$(VERSION)-src.tar.bz2 aptly-$(VERSION)
	rm -rf aptly-$(VERSION)
	curl -T aptly-$(VERSION)-src.tar.bz2 -usmira:$(BINTRAY_KEY) https://api.bintray.com/content/smira/aptly/aptly/$(VERSION)/$(VERSION)/aptly-$(VERSION)-src.tar.bz2

goxc:
	rm -rf root/
	mkdir -p root/usr/share/man/man1/ root/etc/bash_completion.d
	cp man/aptly.1 root/usr/share/man/man1
	(cd root/etc/bash_completion.d && wget https://raw.github.com/aptly-dev/aptly-bash-completion/master/aptly)
	gzip root/usr/share/man/man1/aptly.1
	goxc -pv=$(VERSION) -max-processors=4 $(GOXC_OPTS)

man:
	make -C man

.PHONY: coverage.out man
