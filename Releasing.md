# Creating a Release

- create branch release/1.x.y
- update debian/changelog
- create PR, merge when approved
- on updated master, git tag and push:
    ```
    version=$(dpkg-parsechangelog -S Version)
    git tag -a v$version -m 'aptly: release $version'
    git push aptly-dev v$version
    ```
- run swagger locally
- add generated swagger-1.x.y.json to www.aptly.info
- releae www.aptly.info
- create release announcement on https://github.com/aptly-dev/aptly/discussions
