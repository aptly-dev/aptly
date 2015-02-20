GOVERSION=$(shell go version | awk '{print $$3;}')
PACKAGES=context database deb files http query swift s3 utils
ALL_PACKAGES=api aptly context cmd console database deb files http query swift s3 utils
BINPATH=$(abspath ./_vendor/bin)
GOM_ENVIRONMENT=-test
PYTHON?=python

ifeq ($(GOVERSION), devel)
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
	go get -u github.com/mattn/gom
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
	$(GOM) exec go tool vet -all=true -shadow=true $(ALL_PACKAGES:%=./%)
	$(GOM) exec golint $(ALL_PACKAGES:%=./%)

install:
	$(GOM) build -o $(BINPATH)/aptly

system-test: install
	if [ ! -e ~/aptly-fixture-db ]; then git clone https://github.com/aptly-dev/aptly-fixture-db.git ~/aptly-fixture-db/; fi
	if [ ! -e ~/aptly-fixture-pool ]; then git clone https://github.com/aptly-dev/aptly-fixture-pool.git ~/aptly-fixture-pool/; fi
	PATH=$(BINPATH)/:$(PATH) $(PYTHON) system/run.py --long

travis: $(TRAVIS_TARGET) system-test

test:
	$(GOM) test -v ./... -gocheck.v=true

coveralls: coverage.out
	$(GOM) exec $(BINPATH)/goveralls -service travis-ci.org -coverprofile=coverage.out -repotoken=$(COVERALLS_TOKEN)

mem.png: mem.dat mem.gp
	gnuplot mem.gp
	open mem.png

package:
	rm -rf root/
	mkdir -p root/usr/bin/ root/usr/share/man/man1/ root/etc/bash_completion.d
	cp $(BINPATH)/aptly root/usr/bin
	cp man/aptly.1 root/usr/share/man/man1
	(cd root/etc/bash_completion.d && wget https://raw.github.com/aptly-dev/aptly-bash-completion/master/aptly)
	gzip root/usr/share/man/man1/aptly.1
	fpm -s dir -t deb -n aptly -v $(VERSION) --url=http://www.aptly.info/ --license=MIT --vendor="Andrey Smirnov <me@smira.ru>" \
	   -f -m "Andrey Smirnov <me@smira.ru>" --description="Debian repository management tool" --deb-recommends bzip2 -C root/ .
	mv aptly_$(VERSION)_*.deb ~

src-package:
	rm -rf aptly-$(VERSION)
	mkdir -p aptly-$(VERSION)/src/github.com/smira/aptly/
	cd aptly-$(VERSION)/src/github.com/smira/ && git clone https://github.com/smira/aptly && cd aptly && git checkout v$(VERSION)
	cd aptly-$(VERSION)/src/github.com/smira/aptly && gom -production install
	cd aptly-$(VERSION)/src/github.com/smira/aptly && find . \( -name .git -o -name .bzr -o -name .hg \) -print | xargs rm -rf
	rm -rf aptly-$(VERSION)/src/github.com/smira/aptly/_vendor/{pkg,bin}
	mkdir -p aptly-$(VERSION)/bash_completion.d
	(cd aptly-$(VERSION)/bash_completion.d && wget https://raw.github.com/aptly-dev/aptly-bash-completion/$(VERSION)/aptly)
	tar cyf aptly-$(VERSION)-src.tar.bz2 aptly-$(VERSION)
	rm -rf aptly-$(VERSION)
	curl -T aptly-$(VERSION)-src.tar.bz2 -usmira:$(BINTRAY_KEY) https://api.bintray.com/content/smira/aptly/aptly/$(VERSION)/$(VERSION)/aptly-$(VERSION)-src.tar.bz2

.PHONY: coverage.out
