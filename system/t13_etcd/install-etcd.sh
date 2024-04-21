#!/bin/sh

# etcd test env
ETCD_VER=v3.5.2
DOWNLOAD_URL=https://storage.googleapis.com/etcd

if [ ! -e /tmp/etcd-${ETCD_VER}-linux-amd64.tar.gz ]; then
    curl -L ${DOWNLOAD_URL}/${ETCD_VER}/etcd-${ETCD_VER}-linux-amd64.tar.gz -o /tmp/etcd-${ETCD_VER}-linux-amd64.tar.gz
fi

mkdir /srv/etcd
tar xf /tmp/etcd-${ETCD_VER}-linux-amd64.tar.gz -C /srv/etcd --strip-components=1
