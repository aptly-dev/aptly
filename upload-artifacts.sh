#!/bin/sh

set -e

builds=build/
packages=${builds}*.deb
folder=`mktemp -u tmp.XXXXXXXXXXXXXXX`
aptly_user="$APTLY_USER"
aptly_password="$APTLY_PASSWORD"
aptly_api="https://aptly-ops.aptly.info"
version=`make version`

action=$1
dist=$2

usage() {
    echo "Usage: $0 nighly jammy|focal|bookworm" >&2
    echo "       $0 release" >&2
}

if [ -z "$action" ]; then
    usage
    exit 1
fi

if [ "action" = "nightly" ] && [ -z "$dist" ]; then
    usage
    exit 1
fi

echo "Publishing version '$version' to $action for $dist...\n"

upload()
{
    echo "\nUploading files:"
    for file in $packages; do
        echo " - $file"
        curl -fsS -X POST -F "file=@$file" -u $aptly_user:$aptly_password ${aptly_api}/api/files/$folder
    done
    echo
}
cleanup() {
    echo "\nCleanup..."
    curl -fsS -X DELETE  -u $aptly_user:$aptly_password ${aptly_api}/api/files/$folder
    echo
}
trap cleanup EXIT

if [ "$action" = "nightly" ]; then
    if echo "$version" | grep -vq "+"; then
       # skip nightly when on release tag
       exit 0
    fi

    aptly_repository=aptly-nightly-$dist
    aptly_published=s3:repo.aptly.info:nightly-$dist

    upload

    echo "\nAdding packages to $aptly_repository ..."
    curl -fsS -X POST -u $aptly_user:$aptly_password ${aptly_api}/api/repos/$aptly_repository/file/$folder
    echo

    echo "\nUpdating published repo $aptly_published ..."
    curl -fsS -X PUT -H 'Content-Type: application/json' --data \
        '{"AcquireByHash": true,
          "Signing": {"Batch": true, "Keyring": "aptly.repo/aptly.pub", "secretKeyring": "aptly.repo/aptly.sec", "PassphraseFile": "aptly.repo/passphrase"}}' \
        -u $aptly_user:$aptly_password ${aptly_api}/api/publish/$aptly_published/$dist
    echo

    if [ $dist = "focal" ]; then
        echo "\nUpdating legacy nightly repo..."

        aptly_repository=aptly-nightly
        aptly_published=s3:repo.aptly.info:./nightly

        upload

        echo "\nAdding packages to $aptly_repository ..."
        curl -fsS -X POST -u $aptly_user:$aptly_password ${aptly_api}/api/repos/$aptly_repository/file/$folder
        echo

        echo "\nUpdating published repo $aptly_published ..."
        curl -fsS -X PUT -H 'Content-Type: application/json' --data \
            '{"AcquireByHash": true, "Signing": {"Batch": true, "Keyring": "aptly.repo/aptly.pub",
                                                 "secretKeyring": "aptly.repo/aptly.sec", "PassphraseFile": "aptly.repo/passphrase"}}' \
            -u $aptly_user:$aptly_password ${aptly_api}/api/publish/$aptly_published
        echo
    fi
fi

if [ "$1" = "release" ]; then
    aptly_repository=aptly-release
    aptly_snapshot=aptly-$version
    aptly_published=s3:repo.aptly.info:./squeeze

    echo "\nAdding packages to $aptly_repository..."
    curl -fsS -X POST -u $aptly_user:$aptly_password ${aptly_api}/api/repos/$aptly_repository/file/$folder
    echo

    echo "\nCreating snapshot $aptly_snapshot from repo $aptly_repository..."
    curl -fsS -X POST -u $aptly_user:$aptly_password -H 'Content-Type: application/json' --data \
        "{\"Name\":\"$aptly_snapshot\"}" ${aptly_api}/api/repos/$aptly_repository/snapshots
    echo

    echo "\nSwitching published repo $aptly_published to use snapshot $aptly_snapshot..."
    curl -fsS -X PUT -H 'Content-Type: application/json' --data \
        "{\"AcquireByHash\": true, \"Snapshots\": [{\"Component\": \"main\", \"Name\": \"$aptly_snapshot\"}],
                                   \"Signing\": {\"Batch\": true, \"Keyring\": \"aptly.repo/aptly.pub\",
                                                 \"secretKeyring\": \"aptly.repo/aptly.sec\", \"PassphraseFile\": \"aptly.repo/passphrase\"}}" \
        -u $aptly_user:$aptly_password ${aptly_api}/api/publish/$aptly_published
    echo
fi

