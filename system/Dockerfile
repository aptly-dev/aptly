FROM debian:bookworm-slim

RUN apt-get update -y && apt-get install -y --no-install-recommends curl gnupg apg bzip2 xz-utils ca-certificates golang golang-go golang-doc golang-src make git python3 python3-requests-unixsocket python3-termcolor python3-swiftclient python3-boto python3-azure-storage g++ python3-etcd3 python3-plyvel graphviz devscripts sudo dh-golang binutils-i686-linux-gnu binutils-aarch64-linux-gnu binutils-arm-linux-gnueabihf bash-completion && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

RUN useradd -m --shell /bin/sh --home-dir /var/lib/aptly aptly

RUN mkdir /work
WORKDIR /work
RUN chown aptly /work

RUN cd /var/lib/aptly; git clone https://github.com/aptly-dev/aptly-fixture-db.git
RUN cd /var/lib/aptly; git clone https://github.com/aptly-dev/aptly-fixture-pool.git
RUN cd /var/lib/aptly; curl -O http://repo.aptly.info/system-tests/etcd.db
RUN echo "aptly ALL=(ALL) NOPASSWD: ALL" > /etc/sudoers.d/aptly
ADD . /src
RUN chown aptly -R /src
RUN cd /src; su aptly -c "go mod download"
RUN /src/system/t13_etcd/install-etcd.sh
RUN rm -rf /src

CMD /app/system/run-system-tests
