GOPATH=$(shell go env GOPATH)
VERSION=$(shell make -s version)
PYTHON?=python3
BINPATH?=$(GOPATH)/bin
GOLANGCI_LINT_VERSION=v1.54.1  # version supporting go 1.19
COVERAGE_DIR?=$(shell mktemp -d)
GOOS=$(shell go env GOHOSTOS)
GOARCH=$(shell go env GOHOSTARCH)

# Uncomment to update test outputs
# CAPTURE := "--capture"

help:  ## Print this help
	@grep -E '^[a-zA-Z][a-zA-Z0-9_-]*:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

prepare:  ## Install go module dependencies
	# Prepare go modules
	go mod verify
	go mod tidy -v
	# Generate VERSION file
	go generate

releasetype:  # Print release type: ci (on any branch/commit), release (on a tag)
	@reltype=ci ; \
	gitbranch=`git rev-parse --abbrev-ref HEAD` ; \
	if [ "$$gitbranch" = "HEAD" ] && [ "$$FORCE_CI" != "true" ]; then \
		gittag=`git describe --tags --exact-match 2>/dev/null` ;\
		if echo "$$gittag" | grep -q '^v[0-9]'; then \
			reltype=release ; \
		fi ; \
	fi ; \
	echo $$reltype

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

swagger-install:
	# Install swag
	@test -f $(BINPATH)/swag || GOOS=linux GOARCH=amd64 go install github.com/swaggo/swag/cmd/swag@latest

swagger: swagger-install
	# Generate swagger docs
	@PATH=$(BINPATH)/:$(PATH) swag init --markdownFiles docs

etcd-install:
	# Install etcd
	test -d /srv/etcd || system/t13_etcd/install-etcd.sh

flake8:  ## run flake8 on system test python files
	flake8 system/

lint:
	# Install golangci-lint
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	# Running lint
	@PATH=$(BINPATH)/:$(PATH) golangci-lint run


build: prepare swagger  ## Build aptly
	go build -o build/aptly

install:
	@echo "\e[33m\e[1mBuilding aptly ...\e[0m"
	go generate
	@out=`mktemp`; if ! go install -v > $$out 2>&1; then cat $$out; rm -f $$out; echo "\nBuild failed\n"; exit 1; else rm -f $$out; fi

test: prepare swagger etcd-install  ## Run unit tests
	@echo "\e[33m\e[1mStarting etcd ...\e[0m"
	@mkdir -p /tmp/etcd-data; system/t13_etcd/start-etcd.sh > /tmp/etcd-data/etcd.log 2>&1 &
	@echo "\e[33m\e[1mRunning go test ...\e[0m"
	go test -v ./... -gocheck.v=true -coverprofile=unit.out; echo $$? > .unit-test.ret
	@echo "\e[33m\e[1mStopping etcd ...\e[0m"
	@pid=`cat /tmp/etcd.pid`; kill $$pid
	@rm -f /tmp/etcd-data/etcd.log
	@ret=`cat .unit-test.ret`; if [ "$$ret" = "0" ]; then echo "\n\e[32m\e[1mUnit Tests SUCCESSFUL\e[0m"; else echo "\n\e[31m\e[1mUnit Tests FAILED\e[0m"; fi; rm -f .unit-test.ret; exit $$ret

system-test: prepare swagger etcd-install  ## Run system tests
	# build coverage binary
	go test -v -coverpkg="./..." -c -tags testruncli
	# Download fixture-db, fixture-pool, etcd.db
	if [ ! -e ~/aptly-fixture-db ]; then git clone https://github.com/aptly-dev/aptly-fixture-db.git ~/aptly-fixture-db/; fi
	if [ ! -e ~/aptly-fixture-pool ]; then git clone https://github.com/aptly-dev/aptly-fixture-pool.git ~/aptly-fixture-pool/; fi
	test -f ~/etcd.db || (curl -o ~/etcd.db.xz http://repo.aptly.info/system-tests/etcd.db.xz && xz -d ~/etcd.db.xz)
	# Run system tests
	PATH=$(BINPATH)/:$(PATH) && FORCE_COLOR=1 $(PYTHON) system/run.py --long --coverage-dir $(COVERAGE_DIR) $(CAPTURE) $(TEST)

bench:
	@echo "\e[33m\e[1mRunning benchmark ...\e[0m"
	go test -v ./deb -run=nothing -bench=. -benchmem

serve: prepare swagger-install  ## Run development server (auto recompiling)
	test -f $(BINPATH)/air || go install github.com/air-verse/air@v1.52.3
	cp debian/aptly.conf ~/.aptly.conf
	sed -i /enableSwaggerEndpoint/s/false/true/ ~/.aptly.conf
	PATH=$(BINPATH):$$PATH air -build.pre_cmd 'swag init -q --markdownFiles docs' -build.exclude_dir docs,system,debian,pgp/keyrings,pgp/test-bins,completion.d,man,deb/testdata,console,_man,cmd,systemd -- api serve -listen 0.0.0.0:3142

dpkg: prepare swagger  ## Build debian packages
	@test -n "$(DEBARCH)" || (echo "please define DEBARCH"; exit 1)
	# set debian version
	@if [ "`make -s releasetype`" = "ci" ]; then  \
		echo CI Build, setting version... ; \
		cp debian/changelog debian/changelog.dpkg-bak ; \
		DEBEMAIL="CI <ci@aptly>" dch -v `make -s version` "CI build" ; \
	fi
	# Run dpkg-buildpackage
	buildtype="any" ; \
	if [ "$(DEBARCH)" = "amd64" ]; then  \
	  buildtype="any,all" ; \
	fi ; \
	echo "\e[33m\e[1mBuilding: $$buildtype\e[0m" ; \
	dpkg-buildpackage -us -uc --build=$$buildtype -d --host-arch=$(DEBARCH)
	# cleanup
	@test -f debian/changelog.dpkg-bak && mv debian/changelog.dpkg-bak debian/changelog || true ; \
	mkdir -p build && mv ../*.deb build/ ; \
	cd build && ls -l *.deb

binaries: prepare swagger  ## Build binary releases (FreeBSD, MacOS, Linux tar)
	# build aptly
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o build/tmp/aptly -ldflags='-extldflags=-static'
	# install
	@mkdir -p build/tmp/man build/tmp/completion/bash_completion.d build/tmp/completion/zsh/vendor-completions
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
	@docker run -it --rm -v ${PWD}:/work/src aptly-dev /work/src/system/docker-wrapper build

docker-shell:  ## Run aptly and other commands in docker container
	@docker run -it --rm -v ${PWD}:/work/src aptly-dev /work/src/system/docker-wrapper || true

docker-deb:  ## Build debian packages in docker container
	@docker run -it --rm -v ${PWD}:/work/src aptly-dev /work/src/system/docker-wrapper dpkg DEBARCH=amd64

docker-unit-test:  ## Run unit tests in docker container
	@docker run -it --rm -v ${PWD}:/work/src aptly-dev /work/src/system/docker-wrapper test

docker-system-test:  ## Run system tests in docker container (add TEST=t04_mirror or TEST=UpdateMirror26Test to run only specific tests)
	@docker run -it --rm -v ${PWD}:/work/src aptly-dev /work/src/system/docker-wrapper system-test TEST=$(TEST)

docker-serve:  ## Run development server (auto recompiling) on http://localhost:3142
	@docker run -it --rm -p 3142:3142 -v ${PWD}:/work/src aptly-dev /work/src/system/docker-wrapper serve || true

docker-lint:  ## Run golangci-lint in docker container
	@docker run -it --rm -v ${PWD}:/work/src aptly-dev /work/src/system/docker-wrapper lint

docker-binaries:  ## Build binary releases (FreeBSD, MacOS, Linux tar) in docker container
	@docker run -it --rm -v ${PWD}:/work/src aptly-dev /work/src/system/docker-wrapper binaries

docker-man:  ## Create man page in docker container
	@docker run -it --rm -v ${PWD}:/work/src aptly-dev /work/src/system/docker-wrapper man

mem.png: mem.dat mem.gp
	gnuplot mem.gp
	open mem.png

man:  ## Create man pages
	make -C man

clean:  ## remove local build and module cache
	# Clean all generated and build files
	test -d .go/ && chmod u+w -R .go/ && rm -rf .go/ || true
	rm -rf build/ obj-*-linux-gnu* tmp/
	rm -f unit.out aptly.test VERSION docs/docs.go docs/swagger.json docs/swagger.yaml docs/swagger.conf
	find system/ -type d -name __pycache__ -exec rm -rf {} \; 2>/dev/null || true

.PHONY: help man prepare swagger version binaries docker-release docker-system-test docker-unit-test docker-lint docker-build docker-image build docker-shell clean releasetype dpkg serve docker-serve flake8
