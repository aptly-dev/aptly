FROM aptly-dev

ADD --chown=aptly:aptly . /work/src/

# Pre-populate the Go module cache so go mod verify works offline
RUN chown aptly /work/src && mkdir -p /work/src/.go && chown aptly /work/src/.go && \
    cd /work/src && sudo -u aptly GOPATH=/work/src/.go GOCACHE=/work/src/.go/cache go mod download
