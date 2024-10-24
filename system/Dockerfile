FROM debian:bookworm-slim

ARG USER_ID
ARG GROUP_ID

RUN echo deb http://deb.debian.org/debian bookworm-backports main > /etc/apt/sources.list.d/backports.list

RUN apt-get update -y && apt-get install -y --no-install-recommends curl gnupg bzip2 xz-utils ca-certificates vim procps \
    golang/bookworm-backports golang-go/bookworm-backports golang-doc/bookworm-backports golang-src/bookworm-backports \
    make git python3 python3-requests-unixsocket python3-termcolor python3-swiftclient python3-boto python3-azure-storage \
    g++ python3-etcd3 python3-plyvel graphviz devscripts sudo dh-golang binutils-i686-linux-gnu binutils-aarch64-linux-gnu \
    binutils-arm-linux-gnueabihf bash-completion zip ruby3.1-dev && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

RUN useradd -m --shell /bin/bash --home-dir /var/lib/aptly -u $USER_ID -g $GROUP_ID aptly
RUN sed -i 's/#force_color_prompt=yes/force_color_prompt=yes/' /var/lib/aptly/.bashrc

RUN mkdir /work
WORKDIR /work/src
RUN chown aptly /work

ADD system/t13_etcd/install-etcd.sh /src/
RUN /src/install-etcd.sh
RUN rm -rf /src
RUN echo "aptly ALL=(ALL) NOPASSWD: ALL" > /etc/sudoers.d/aptly

USER aptly

RUN cd /var/lib/aptly; git clone https://github.com/aptly-dev/aptly-fixture-db.git
RUN cd /var/lib/aptly; git clone https://github.com/aptly-dev/aptly-fixture-pool.git
RUN cd /var/lib/aptly; curl -O http://repo.aptly.info/system-tests/etcd.db.xz && xz -d etcd.db.xz

ENTRYPOINT [ "/work/src/system/docker-wrapper" ]
