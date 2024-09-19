FROM debian:bookworm-slim

RUN echo deb http://deb.debian.org/debian bookworm-backports main > /etc/apt/sources.list.d/backports.list

RUN apt-get update -y && apt-get install -y --no-install-recommends curl gnupg apg bzip2 xz-utils ca-certificates \
        golang/bookworm-backports golang-go/bookworm-backports golang-doc/bookworm-backports golang-src/bookworm-backports \
        make git python3 python3-requests-unixsocket python3-termcolor python3-swiftclient python3-boto python3-azure-storage g++ python3-etcd3 python3-plyvel graphviz devscripts sudo dh-golang binutils-i686-linux-gnu binutils-aarch64-linux-gnu binutils-arm-linux-gnueabihf bash-completion zip && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

RUN useradd -m --shell /bin/sh --home-dir /var/lib/aptly aptly

RUN mkdir /work
WORKDIR /work
RUN chown aptly /work

RUN cd /var/lib/aptly; git clone https://github.com/aptly-dev/aptly-fixture-db.git
RUN cd /var/lib/aptly; git clone https://github.com/aptly-dev/aptly-fixture-pool.git
RUN cd /var/lib/aptly; curl -O http://repo.aptly.info/system-tests/etcd.db
RUN echo "aptly ALL=(ALL) NOPASSWD: ALL" > /etc/sudoers.d/aptly
ADD system/t13_etcd/install-etcd.sh /src/
RUN /src/install-etcd.sh
RUN rm -rf /src

CMD /app/system/run-system-tests
