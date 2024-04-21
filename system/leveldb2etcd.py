#!/usr/bin/env python

import plyvel
import etcd3
import argparse
from termcolor import cprint

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--datadir", required=True, help="leveldb data dir")
    parser.add_argument("--etcdaddr", default="127.0.0.1", help="etcd server address")
    parser.add_argument("--etcdport", default="2379", help="etcd server address")

    args = parser.parse_args()

    ldb = plyvel.DB(args.datadir)
    etcd = etcd3.client(args.etcdaddr, args.etcdport)

    for key, value in ldb:
        try:
            keystr = key
            valuestr = value
            etcd.put(keystr, valuestr)
            # cprint("key: "+keystr+", value: "+valuestr+"put success!\n", 'green')
        except Exception as e:
            cprint("key: " + str(keystr) + ", value: " + str(valuestr) + "put err: " + str(e) + "\n", 'red')
            exit(1)
