# 13. Development Guide

## 13.1 Repository map

```
aptly/            interfaces (PackagePool, PublishedStorage, Downloader, Progress, …)
api/              REST handlers + router (gin) + swagger annotations
azure/ gcs/ s3/ swift/ jfrog/   cloud storage implementations
cmd/              CLI commands (commander)
console/          terminal progress
context/          AptlyContext
database/         KV interface + goleveldb + etcddb
deb/              Debian domain logic
docs/             swagger sources & generated docs package
files/            local pool + local published storage
http/             downloaders, compression fallback
pgp/              GPG (external / internal)
query/            query language parser
task/             task queue
utils/            config, checksums, compression, helpers
system/           Python system (functional) test suite
debian/ man/ completion.d/ _man/ systemd/   packaging, docs, completion, activation helper
```

Go version: see `go.mod` (`go 1.26`). Key dependencies: `gin-gonic/gin` (HTTP), `smira/commander` +
`smira/flag` (CLI), `syndtr/goleveldb` and `go.etcd.io/etcd/client/v3` (DB), `ugorji/go/codec` (msgpack),
`aws/aws-sdk-go-v2`, `cloud.google.com/go/storage`, `Azure/azure-sdk-for-go` (azblob), `ncw/swift`, `jfrog-client-go` (storage),
`klauspost/compress` + `pgzip` (compression), `rs/zerolog` (logs), `prometheus/client_golang`,
`swaggo/*` (OpenAPI), `awalterschulze/gographviz`, `cheggaaa/pb`, `google/uuid`, `saracen/walker`,
`gopkg.in/check.v1` (gocheck test framework), `gopkg.in/yaml.v3`.

## 13.2 Building

```bash
make prepare        # go mod verify/tidy + `go generate` (creates the VERSION file embedded in the binary)
make build          # swagger + `go build -o build/aptly`
make binaries       # cross-built static binaries
make dpkg           # Debian packages
make docker-build   # everything inside the dev container (system/Dockerfile)
make help           # all documented targets
```

`VERSION` comes from `make version`: the top `debian/changelog` version, plus a
`+<commit-date>.<hash>` suffix for non-tag (CI) builds (`releasetype`).

## 13.3 Tests

### Unit tests

```bash
make test                              # go test ./... with gocheck; runs under `faketime`
make test TEST=SnapshotSuite           # -check.f regex to select suites/tests
make bench                             # go test ./deb -run=nothing -bench=. -benchmem
```

* Written with **gocheck** (`. "gopkg.in/check.v1"`): a suite type + `Test…` methods, hooked with
  `func Test(t *testing.T) { TestingT(t) }` in each package's `*_test.go`.
* `faketime` pins the clock (`TEST_FAKETIME`) because some fixtures use expired certificates/keys.
* Fakes: `http.NewFakeDownloader`, `files/mocks.go`, temp directories, `goleveldb.NewOpenDB(tempdir)`,
  and a fake S3 (`s3/server_test.go`).
* Tests for `cmd`/`context` are minimal; behaviour is covered by the system tests below.

### System (functional) tests — `system/`

Python 3 test-runner exercising the **real binary** through the CLI and the REST API.

```bash
make system-test                                   # build test binary + run everything (long tests)
make docker-system-test TEST=t04_mirror            # one directory
make docker-system-test TEST=UpdateMirror26Test    # one test class
make docker-system-test CAPTURE=1 TEST=…           # regenerate gold files
COVERAGE_SKIP=1 make system-test                   # skip coverage plumbing
```

Structure:

* `system/tNN_name/` — a test *package* per feature: `t01_version`, `t02_config`, `t03_help`,
  `t04_mirror`, `t05_snapshot`, `t06_publish`, `t07_serve`, `t08_db`, `t09_repo`, `t10_task`,
  `t11_package`, `t12_api`, `t13_etcd`, `t14_graph`. Storage back-ends are exercised through helper base classes
  (`s3_lib.py`, `swift_lib.py`, `azure_lib.py`, `gcs_lib.py`, `jfrog_lib.py`, `fs_endpoint_lib.py`).
* Each `*.py` defines classes deriving from `BaseTest` (`system/lib.py`) or a specialised base
  (`APITest` in `api_lib.py`, `S3Test`, `SwiftTest`, `AzureTest`, `FileSystemEndpointTest`).
  Key attributes: `runCmd` (the command line), `expectedCode`, `fixtureDB`/`fixturePool`/`fixtureGpg`/`fixtureWebServer`,
  `configOverride`, `longTest`, `requiresGPG1/2`, `requiresDot`, `databaseType` (`leveldb`/`etcd`), `sortOutput`,
  `outputMatchPrepare` (regexp normalisers).
* **Gold files** `<TestName>_gold` hold expected stdout; `*_mirror_show`, etc. hold expected follow-up
  command outputs. `CAPTURE=1` rewrites them — review diffs carefully.
* Fixtures come from separate repos (`aptly-fixture-db`, `aptly-fixture-pool`) and `system/files` (test keys,
  packages). A local HTTP server (`fixtureWebServer`) hosts fake remote mirrors so no internet is needed.
* The runner builds a **coverage-instrumented test binary** using `main_test.go` (`//go:build testruncli`,
  `TestRunMain` calls `cmd.Run` and prints `EXIT: n`), enabling coverage of CLI paths.
* Lint for tests: `make flake8` (`.flake8`).

### CI (`.github/workflows/ci.yml`, `golangci-lint.yml`)

Jobs: **Unit Tests** (Docker image, coverage) → **System Test** (Ubuntu runner with
`graphviz gnupg2 gpgv2 faketime python3-…`, flake8, Azurite for Azure, benchmarks) → **Upload Coverage**
(merged, to Codecov via `codecov.yml`) → **Build** matrix producing packages for Debian 11/12/13 and Ubuntu 20.04–26.04
(artifact upload via `.github/workflows/scripts/upload-artifacts.sh`). `golangci-lint.yml` runs linting separately.

### Linting

`make lint` runs golangci-lint (version pinned in the Makefile). Code style: standard `gofmt`; exported
identifiers documented; `// nolint:` markers only where justified.

## 13.4 Generating API docs

Handlers carry swag annotations:

```go
// @Summary Create Repository
// @Description **Create a local repository** …
// @Tags Repos
// @Consume json
// @Param request body repoCreateParams true "Parameters"
// @Produce json
// @Success 201 {object} deb.LocalRepo
// @Failure 400 {object} Error "Bad Request"
// @Router /api/repos [post]
func apiReposCreate(c *gin.Context) { … }
```

`make swagger` runs `swag init --propertyStrategy pascalcase --parseDependency --parseInternal
--markdownFiles docs --generalInfo docs/swagger.conf` and regenerates `docs/docs.go`, `swagger.json/yaml`.
The `docs/*.md` files (`Repos.md`, `Snapshots.md`, …) provide the tag descriptions shown in the UI. `docs/index.go`
embeds `docs.html` served by the router. (This `documentation/` folder is intentionally separate and
is **not** consumed by swag.)

## 13.5 Recipes for common changes

### Add a new CLI command

1. Create `cmd/<group>_<verb>.go`:
   ```go
   func aptlyFooBar(cmd *commander.Command, args []string) error {
       if len(args) != 1 { cmd.Usage(); return commander.ErrCommandError }
       collectionFactory := context.NewCollectionFactory()
       … // use deb.* APIs
       return nil
   }
   func makeCmdFooBar() *commander.Command {
       c := &commander.Command{Run: aptlyFooBar, UsageLine: "bar <name>", Short: "…", Long: `…`,
           Flag: *flag.NewFlagSet("aptly-foo-bar", flag.ExitOnError)}
       c.Flag.Bool("dry-run", false, "…")
       return c
   }
   ```
2. Register it in the group's `Subcommands` (e.g. `cmd/snapshot.go`) or add a new group in `RootCommand`.
3. Read flags through `context.Flags().Lookup("dry-run").Value.Get().(bool)`; use `LookupOption` when a
   config default exists.
4. Update help golds (`system/t03_help`), man page (`make man`), completions (`completion.d/*`), and add
   system tests (+ gold files).

### Add a REST endpoint

1. Add params struct with `binding:"required"`/`json:` tags and swag comments in `api/<area>.go`.
2. Handler skeleton: bind → shallow validation/404 → `maybeRunTaskInBackground(c, name, resources, func(out, detail){ fresh
   factory; LoadComplete; mutate; Update; return &task.ProcessReturnValue{Code, Value}, err })`.
   **Choose resource keys** = keys of every object you mutate (`repo.Key()` …); use `task.AllResourcesKey` for
   global operations.
3. Register in `api/router.go` (respect canonical lock order in `api/api.go` comments).
4. `make swagger`; add tests in `api/*_test.go` (gin test router + fake context, see `api/api_test.go`) and system
   tests in `system/t12_api`.

### Add a published-storage backend

1. New package `foo/` with `PublishedStorage` implementing `aptly.PublishedStorage` (all 11 methods; make
   `MkDir` a no-op for flat namespaces). Cache remote listings (`pathCache`) if `LinkFromPool` needs MD5 lookups.
   Store MD5 as metadata if the store's ETag isn't MD5.
2. Config struct in `utils/config.go` (`FooPublishRoots map[string]FooPublishRoot` + json/yaml tags, initialise the
   map in `Config`), documented in `debian/aptly.conf`.
3. Wire in `context.GetPublishedStorage`: `strings.HasPrefix(name, "foo:")` → constructor.
4. Optional: an API listing endpoint (like `api/s3.go`), tests (`foo/public_test.go` with a fake/emulator), and a system
   test base class in `system/`.

### Add a package pool backend

Implement `aptly.PackagePool` (+ `LocalPackagePool` if files live on a local FS); extend `PackagePoolStorage` (custom
`UnmarshalJSON/YAML` discriminator on `type`) and `context.PackagePool()`.

### Add a database backend

Implement `database.Storage` (`Reader/Writer/PrefixReader/Batch/Transaction/CreateTemporary/CompactDB/Drop`), reuse the
contract tests pattern from `database/goleveldb/database_test.go`, and select it in `context._database()` by
`database_backend.type`. Keep keys **byte-ordered prefix scannable** — the entire data model depends on it.

### Extend the query language

`query/lex.go` (tokens), `query/syntax.go` (grammar → `deb.PackageQuery`), `deb/query.go` (new `PackageQuery` type
implementing `Matches/Fast/Query/String`). Add unit tests in `query/*_test.go` and `deb/query_test.go`; document in the
website + man page (`docs`).

### Add a package field to the model

Fields in `deb.Package` are msgpack-serialised: **adding** exported fields is backward compatible; renaming/removing is
not. Bump handling in `PackageCollection.ByKey` if the format truly changes (see the `0xC1 0x01` marker and `oldPackage`).
Remember `PackageFiles.Hash()` participates in the package key — changing what it hashes changes keys of *all* packages.

## 13.6 Debugging tips

* `-dep-verbose-resolve` for dependency resolution; `log_level: debug` for storage back-ends and API.
* Build with `aptly/version.go: EnableDebug = true` to get `-cpuprofile/-memprofile/-memstats` and gin debug mode.
* Inspect the DB: LevelDB keys are readable text prefixes; use a LevelDB tool or write a small Go program with
  `goleveldb.NewOpenDB(path)` and `ProcessByPrefix([]byte("S"), …)`. **Stop aptly first** (lock).
* `aptly package show -with-references -with-files 'Pamd64 nginx 1.24.0-1 8f3a91c2'` shows which mirrors/snapshots/repos
  reference a package and the pool file paths.
* `aptly graph -output=g.svg` visualises the DAG.
* Reproduce publish output deterministically with `SOURCE_DATE_EPOCH`.
* Use `aptly task run` in system tests when scripting multiple commands against one DB.

## 13.7 Releasing

See [`Releasing.md`](../Releasing.md): create `release/1.x.y`, update `debian/changelog`, merge, tag `v<version>` on master
(`git tag -a v$version …`), publish the generated `docs/swagger.json` to the website repository, update the download page and
announce. `make version` derives the version string embedded in the binary.

## 13.8 Contributing

Read [`CONTRIBUTING.md`](../CONTRIBUTING.md) and [`CODE_OF_CONDUCT.md`](../CODE_OF_CONDUCT.md). PRs should include unit
and/or system tests (a *failing* system test reproducing a bug is a welcome first PR), pass `make lint`, and update the
man page/docs when behaviour changes.
