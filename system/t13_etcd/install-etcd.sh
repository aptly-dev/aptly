#!/bin/sh

# etcd test env
ETCD_VER=v3.5.2
DOWNLOAD_URL=https://storage.googleapis.com/etcd

ARCH=""
OS=""
case $(uname -s) in
  Linux)   OS="linux" ;;
  Darwin)  OS="darwin" ;;
  *)  echo "unsupported OS"; exit 1 ;;
esac

case $(uname -m) in
  x86_64)  ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  arm64)   ARCH="arm64" ;; # macOS M1/M2
  *)  echo "unsupported cpu arch"; exit 1 ;;
esac

TARBALL="etcd-${ETCD_VER}-${OS}-$ARCH.tar.gz"
if [ ! -e /tmp/$TARBALL ]; then
    curl -L ${DOWNLOAD_URL}/${ETCD_VER}/$TARBALL -o /tmp/$TARBALL
fi

mkdir -p /tmp/aptly-etcd
tar xf /tmp/$TARBALL -C /tmp/aptly-etcd --strip-components=1
