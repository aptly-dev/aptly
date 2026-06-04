FROM aptly-dev

RUN apt-get update -y && apt-get install -y --no-install-recommends dput-ng && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

ADD --chown=aptly:aptly . /work/src/

# Pre-populate the Go module cache so go mod verify works offline
RUN chown aptly /work/src && mkdir -p /work/src/.go && chown aptly /work/src/.go && \
    cd /work/src && sudo -u aptly GOPATH=/work/src/.go GOCACHE=/work/src/.go/cache go mod download
