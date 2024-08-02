FROM debian:bookworm-slim

RUN apt-get update -y && apt-get install -y --no-install-recommends curl gnupg apg bzip2 xz-utils ca-certificates golang golang-go golang-doc golang-src make git python3 python3-requests-unixsocket python3-termcolor python3-swiftclient python3-boto python3-azure-storage g++ python3-etcd3 python3-plyvel graphviz && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

RUN useradd -m --shell /bin/sh --home-dir /var/lib/aptly aptly

RUN mkdir app
WORKDIR /app

RUN mkdir /home/runner
RUN chown aptly /home/runner
RUN cd /home/runner; git clone https://github.com/aptly-dev/aptly-fixture-db.git
RUN cd /home/runner; git clone https://github.com/aptly-dev/aptly-fixture-pool.git
RUN cd /home/runner; curl -O http://repo.aptly.info/system-tests/etcd.db
ADD . /src
RUN chown aptly -R /src
RUN cd /src; su aptly -c "HOME=/home/runner go mod tidy"
RUN /src/system/t13_etcd/install-etcd.sh
RUN rm -rf /src

CMD /app/system/run-system-tests
