GOVERSION=$(shell go version | awk '{print $$3;}')
PACKAGES=database debian files http utils
ALL_PACKAGES=aptly cmd console database debian files http utils
BINPATH=$(abspath ./bin)
GOM_ENVIRONMENT=-test

ifeq ($(GOVERSION), go1.2)
TRAVIS_TARGET=coveralls
GOM_ENVIRONMENT+=-development
else
TRAVIS_TARGET=test
endif

ifeq ($(TRAVIS), true)
GOM=$(HOME)/gopath/bin/gom
else
GOM=gom
endif

all: test check system-test

prepare:
	mkdir -p $(BINPATH)
	go get github.com/mattn/gom
	$(GOM) $(GOM_ENVIRONMENT) install

coverage.out:
	rm -f coverage.*.out
	for i in $(PACKAGES); do $(GOM) test -coverprofile=coverage.$$i.out -covermode=count ./$$i; done
	echo "mode: count" > coverage.out
	grep -v -h "mode: count" coverage.*.out >> coverage.out
	rm -f coverage.*.out

coverage: coverage.out
	$(GOM) exec go tool cover -html=coverage.out
	rm -f coverage.out

check:
	$(GOM) exec go tool vet -all=true $(ALL_PACKAGES:%=./%)
	$(GOM) exec golint $(ALL_PACKAGES:%=./%)

system-test:
ifeq ($(GOVERSION),$(filter $(GOVERSION),go1.2 devel))
	if [ ! -e ~/aptly-fixture-db ]; then git clone https://github.com/aptly-dev/aptly-fixture-db.git ~/aptly-fixture-db/; fi
endif
	if [ ! -e ~/aptly-fixture-pool ]; then git clone https://github.com/aptly-dev/aptly-fixture-pool.git ~/aptly-fixture-pool/; fi
	$(GOM) build -o $(BINPATH)/aptly
	PATH=$(BINPATH)/:$(PATH) python system/run.py --long

travis: $(TRAVIS_TARGET) system-test

test:
	$(GOM) test -v $(PACKAGES:%=./%) -gocheck.v=true

coveralls: coverage.out
	$(GOM) build -o $(BINPATH)/goveralls github.com/mattn/goveralls
	$(GOM) exec $(BINPATH)/goveralls -service travis-ci.org -coverprofile=coverage.out -repotoken $(COVERALLS_TOKEN)

mem.png: mem.dat mem.gp
	gnuplot mem.gp
	open mem.png

.PHONY: coverage.out