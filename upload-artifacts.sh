#!/bin/sh

set -e

builds=build/
packages=${builds}*.deb
folder=`mktemp -u tmp.XXXXXXXXXXXXXXX`
aptly_user="$APTLY_USER"
aptly_password="$APTLY_PASSWORD"
aptly_api="https://internal.aptly.info"
version=`make version`

for file in $packages; do
    echo "Uploading $file..."
    curl -fsS -X POST -F "file=@$file" -u $aptly_user:$aptly_password ${aptly_api}/api/files/$folder
    echo
done

if [[ "$1" = "nightly" ]]; then
    aptly_repository=aptly-nightly
    aptly_published=s3:repo.aptly.info:./nightly

    echo "Adding packages to $aptly_repository..."
    curl -fsS -X POST -u $aptly_user:$aptly_password ${aptly_api}/api/repos/$aptly_repository/file/$folder
    echo

    echo "Updating published repo..."
    curl -fsS -X PUT -H 'Content-Type: application/json' --data \
        '{"AcquireByHash": true, "Signing": {"Batch": true, "Keyring": "aptly.repo/aptly.pub",
                                             "secretKeyring": "aptly.repo/aptly.sec", "PassphraseFile": "aptly.repo/passphrase"}}' \
        -u $aptly_user:$aptly_password ${aptly_api}/api/publish/$aptly_published
    echo
fi

if [[ "$1" = "release" ]]; then
    aptly_repository=aptly-release
    aptly_snapshot=aptly-$version
    aptly_published=s3:repo.aptly.info:./squeeze

    echo "Adding packages to $aptly_repository..."
    curl -fsS -X POST -u $aptly_user:$aptly_password ${aptly_api}/api/repos/$aptly_repository/file/$folder
    echo

    echo "Creating snapshot $aptly_snapshot from repo $aptly_repository..."
    curl -fsS -X POST -u $aptly_user:$aptly_password -H 'Content-Type: application/json' --data \
        "{\"Name\":\"$aptly_snapshot\"}" ${aptly_api}/api/repos/$aptly_repository/snapshots
    echo

    echo "Switching published repo to use snapshot $aptly_snapshot..."
    curl -fsS -X PUT -H 'Content-Type: application/json' --data \
        "{\"AcquireByHash\": true, \"Snapshots\": [{\"Component\": \"main\", \"Name\": \"$aptly_snapshot\"}],
                                   \"Signing\": {\"Batch\": true, \"Keyring\": \"aptly.repo/aptly.pub\",
                                                 \"secretKeyring\": \"aptly.repo/aptly.sec\", \"PassphraseFile\": \"aptly.repo/passphrase\"}}" \
        -u $aptly_user:$aptly_password ${aptly_api}/api/publish/$aptly_published
    echo
fi

curl -fsS -X DELETE  -u $aptly_user:$aptly_password ${aptly_api}/api/files/$folder
echo
