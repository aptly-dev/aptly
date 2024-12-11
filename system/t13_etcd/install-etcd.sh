#!/bin/sh

# etcd test env
ETCD_VER=v3.5.2
DOWNLOAD_URL=https://storage.googleapis.com/etcd

ARCH=""
case $(uname -m) in
  x86_64)  ARCH="amd64" ;;
  aarch64)  ARCH="arm64" ;;
  *)  echo "unsupported cpu arch"; exit 1 ;;
esac

if [ ! -e /tmp/etcd-${ETCD_VER}-linux-$ARCH.tar.gz ]; then
    curl -L ${DOWNLOAD_URL}/${ETCD_VER}/etcd-${ETCD_VER}-linux-$ARCH.tar.gz -o /tmp/etcd-${ETCD_VER}-linux-$ARCH.tar.gz
fi

mkdir /tmp/aptly-etcd
tar xf /tmp/etcd-${ETCD_VER}-linux-$ARCH.tar.gz -C /tmp/aptly-etcd --strip-components=1
