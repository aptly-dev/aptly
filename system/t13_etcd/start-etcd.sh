#!/bin/sh

if [ -e /tmp/etcd.pid ]; then
    echo etcd already running, killing..
    etcdpid=`cat /tmp/etcd.pid`
    kill $etcdpid
    sleep 2
fi

finish()
{
    if [ -n "$etcdpid" ]; then
        echo terminating etcd
        kill $etcdpid
    fi
}
trap finish INT

/srv/etcd/etcd --max-request-bytes '1073741824' --data-dir /tmp/etcd-data &
echo $! > /tmp/etcd.pid
etcdpid=`cat /tmp/etcd.pid`
wait $etcdpid
echo etcd terminated
