GOPATH=$(shell go env GOPATH)
VERSION=$(shell make -s version)
PYTHON?=python3
TESTS?=
BINPATH?=$(GOPATH)/bin
RUN_LONG_TESTS?=yes
GOLANGCI_LINT_VERSION=v1.54.1  # version supporting go 1.19
COVERAGE_DIR?=$(shell mktemp -d)
GOOS=$(shell go env GOHOSTOS)
GOARCH=$(shell go env GOHOSTARCH)
RELEASE=no
# Uncomment to update test outputs
# CAPTURE := "--capture"

# Self-documenting Makefile
# https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
help:  ## Print this help
	@grep -E '^[a-zA-Z][a-zA-Z0-9_-]*:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

all: prepare test bench check system-test

prepare:  ## Install go module dependencies
	go mod download
	go mod verify
	go mod tidy -v
	go generate

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
	cd /home/runner; curl -O http://repo.aptly.info/system-tests/etcd.db.xz; xz -d etcd.db.xz
	PATH=$(BINPATH)/:$(PATH) && . system/env/bin/activate && APTLY_VERSION=$(VERSION) FORCE_COLOR=1 $(PYTHON) system/run.py --long $(TESTS) --coverage-dir $(COVERAGE_DIR) $(CAPTURE)
endif

docker-test: ## Run system tests
	@echo Building aptly.test ...
	@rm -f aptly.test
	go generate
	go test -v -coverpkg="./..." -c -tags testruncli
	@echo Running python tests ...
	@test -e aws.creds && . ./aws.creds; \
	export PATH=$(BINPATH)/:$(PATH); \
	export APTLY_VERSION=$(VERSION); \
	$(PYTHON) system/run.py --long $(TESTS) --coverage-dir $(COVERAGE_DIR) $(CAPTURE) $(TEST)

test: prepare  ## Run unit tests
	@test -d /srv/etcd || system/t13_etcd/install-etcd.sh
	@echo "\nStarting etcd ..."
	@mkdir -p /tmp/etcd-data; system/t13_etcd/start-etcd.sh > /tmp/etcd-data/etcd.log 2>&1 &
	@echo "\nRunning go test ..."
	go test -v ./... -gocheck.v=true -coverprofile=unit.out; echo $$? > .unit-test.ret
	@echo "\nStopping etcd ..."
	@pid=`cat /tmp/etcd.pid`; kill $$pid
	@rm -f /tmp/etcd-data/etcd.log
	@ret=`cat .unit-test.ret`; if [ "$$ret" = "0" ]; then echo "\n\e[32m\e[1mUnit Tests SUCCESSFUL\e[0m"; else echo "\n\e[31m\e[1mUnit Tests FAILED\e[0m"; fi; rm -f .unit-test.ret; exit $$ret

bench:
	go test -v ./deb -run=nothing -bench=. -benchmem

mem.png: mem.dat mem.gp
	gnuplot mem.gp
	open mem.png

man:  ## Create man pages
	make -C man

version:  ## Print aptly version
	@ci="" ; \
	if [ "`make -s releasetype`" = "ci" ]; then \
		ci=`TZ=UTC git show -s --format='+%cd.%h' --date=format-local:'%Y%m%d%H%M%S'`; \
	fi ; \
	if which dpkg-parsechangelog > /dev/null 2>&1; then \
		echo `dpkg-parsechangelog -S Version`$$ci; \
	else \
		echo `grep ^aptly -m1  debian/changelog | sed 's/.*(\([^)]\+\)).*/\1/'`$$ci ; \
	fi

releasetype:  # Print release type (ci/release)
	@reltype=ci ; \
	gitbranch=`git rev-parse --abbrev-ref HEAD` ; \
	if [ "$$gitbranch" = "HEAD" ] && [ "$$FORCE_CI" != "true" ]; then \
		gittag=`git describe --tags --exact-match` ;\
		if echo "$$gittag" | grep -q '^v[0-9]'; then \
			reltype=release ; \
		fi ; \
	fi ; \
	echo $$reltype

build:  ## Build aptly
	go mod tidy
	go generate
	go build -o build/aptly

dpkg:  ## Build debian packages
	@test -n "$(DEBARCH)" || (echo "please define DEBARCH"; exit 1)
	@if [ "`make -s releasetype`" = "ci" ]; then  \
		echo CI Build, setting version... ; \
		cp debian/changelog debian/changelog.dpkg-bak ; \
		DEBEMAIL="CI <ci@aptly>" dch -v `make -s version` "CI build" ; \
	fi
	buildtype="any" ; \
	if [ "$(DEBARCH)" = "amd64" ]; then  \
	  buildtype="any,all" ; \
	fi ; \
	echo Building: $$buildtype ; \
	dpkg-buildpackage -us -uc --build=$$buildtype -d --host-arch=$(DEBARCH)
	@test -f debian/changelog.dpkg-bak && mv debian/changelog.dpkg-bak debian/changelog || true ; \
	mkdir -p build && mv ../*.deb build/ ; \
	cd build && ls -l *.deb

binaries:  ## Build binary releases (FreeBSD, MacOS, Linux tar)
	@mkdir -p build/tmp/man build/tmp/completion/bash_completion.d build/tmp/completion/zsh/vendor-completions
	@make version > VERSION
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o build/tmp/aptly -ldflags='-extldflags=-static'
	@cp man/aptly.1 build/tmp/man/
	@cp completion.d/aptly build/tmp/completion/bash_completion.d/
	@cp completion.d/_aptly build/tmp/completion/zsh/vendor-completions/
	@cp README.rst LICENSE AUTHORS build/tmp/
	@gzip -f build/tmp/man/aptly.1
	@path="aptly_$(VERSION)_$(GOOS)_$(GOARCH)"; \
	rm -rf "build/$$path"; \
	mv build/tmp build/"$$path"; \
	rm -rf build/tmp; \
	cd build; \
	zip -r "$$path".zip "$$path" > /dev/null \
		&& echo "Built build/$${path}.zip"; \
	rm -rf "$$path"

docker-image:  ## Build aptly-dev docker image
	@docker build -f system/Dockerfile . -t aptly-dev

docker-build:  ## Build aptly in docker container
	@docker run -it --rm -v ${PWD}:/work/src aptly-dev /work/src/system/run-aptly-cmd make build

docker-aptly:  ## Build and run aptly commands in docker container
	@docker run -it --rm -v ${PWD}:/work/src aptly-dev /work/src/system/run-aptly-cmd

docker-deb:  ## Build debian packages in docker container
	@docker run -it --rm -v ${PWD}:/work/src aptly-dev /work/src/system/build-deb

docker-unit-tests:  ## Run unit tests in docker container
	@docker run -it --rm -v ${PWD}:/app aptly-dev /app/system/run-unit-tests

docker-system-tests:  ## Run system tests in docker container (add TEST=t04_mirror to run only specific tests)
	@docker run -it --rm -v ${PWD}:/app aptly-dev /app/system/run-system-tests $(TEST)

docker-lint:  ## Run golangci-lint in docker container
	@docker run -it --rm -v ${PWD}:/app -e GOLANGCI_LINT_VERSION=$(GOLANGCI_LINT_VERSION) aptly-dev /app/system/run-golangci-lint

docker-binaries:  ## Build binary releases (FreeBSD, MacOS, Linux tar) in docker container
	@docker run -it --rm -v ${PWD}:/app aptly-dev /app/system/build-binaries

flake8:  ## run flake8 on system tests
	flake8 system

clean:  ## remove local build and module cache
	test -d .go/ && chmod u+w -R .go/ && rm -rf .go/ || true
	rm -rf build/ docs/ obj-*-linux-gnu*

.PHONY: help man prepare version binaries docker-release docker-system-tests docker-unit-tests docker-lint docker-build docker-image build docker-aptly clean releasetype dpkg
