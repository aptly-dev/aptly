FROM debian:bookworm-slim

RUN apt-get update -y && apt-get install -y --no-install-recommends curl gnupg && apt-get clean && rm -rf /var/lib/apt/lists/*

RUN echo deb http://deb.debian.org/debian bookworm-backports main > /etc/apt/sources.list.d/backports.list
RUN apt-get update && \
    apt-get install -y --no-install-recommends apg bzip2 xz-utils ca-certificates golang/bookworm-backports golang-go/bookworm-backports golang-doc/bookworm-backports golang-src/bookworm-backports make git python3 python3-venv && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

RUN useradd -m --shell /bin/sh --home-dir /var/lib/aptly aptly

RUN mkdir app
WORKDIR /app

RUN su - aptly -c "git clone https://github.com/aptly-dev/aptly-fixture-db.git"
RUN su - aptly -c "git clone https://github.com/aptly-dev/aptly-fixture-pool.git"

CMD /app/system/run-system-tests
