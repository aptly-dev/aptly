# Creating a Release

- create branch release/1.x.y
- update debian/changelog
- create PR, merge when approved
- on updated master, create release:
```
version=$(dpkg-parsechangelog -S Version)
echo Releasing prod version $version
git tag -a v$version -m 'aptly: release $version'
git push origin v$version master
```
- run swagger locally (`make docker-serve`)
- copy generated docs/swagger.json to https://github.com/aptly-dev/www.aptly.info/tree/master/static/swagger/aptly_1.x.y.json
- releae www.aptly.info
- create release announcement on https://github.com/aptly-dev/aptly/discussions
