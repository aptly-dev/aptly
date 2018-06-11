#!/bin/sh

set -e

builds=build/
packages=${builds}*.deb
folder=`mktemp -u tmp.XXXXXXXXXXXXXXX`
aptly_user="$APTLY_USER"
aptly_password="$APTLY_PASSWORD"
aptly_api="https://internal.aptly.info"

for file in $packages; do
    echo "Uploading $file..."
    curl -sS -X POST -F "file=@$file" -u $aptly_user:$aptly_password ${aptly_api}/api/files/$folder
    echo
done

if [[ "$1" = "nightly" ]]; then
    aptly_repository=aptly-nightly
    aptly_published=s3:repo.aptly.info:./nightly

    echo "Adding packages to $aptly_repository..."
    curl -sS -X POST -u $aptly_user:$aptly_password ${aptly_api}/api/repos/$aptly_repository/file/$folder
    echo

    echo "Updating published repo..."
    curl -sS -X PUT -H 'Content-Type: application/json' --data \
        '{"AcquireByHash": true, "Signing": {"Batch": true, "Keyring": "aptly.repo/aptly.pub",
                                             "secretKeyring": "aptly.repo/aptly.sec", "PassphraseFile": "aptly.repo/passphrase"}}' \
        -u $aptly_user:$aptly_password ${aptly_api}/api/publish/$aptly_published
    echo
fi

curl -sS -X DELETE  -u $aptly_user:$aptly_password ${aptly_api}/api/files/$folder
echo
