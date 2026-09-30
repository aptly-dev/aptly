# 7. REST API & Task System

`aptly api serve` exposes nearly every CLI capability over HTTP/JSON. Handlers live in
`api/`; the router is `api.Router(ctx)` (`api/router.go`), built on
[gin](https://github.com/gin-gonic/gin). This chapter explains request handling, the task
system that serialises mutations, and gives an endpoint catalogue.

> The generated OpenAPI/Swagger spec is produced from the `@Summary/@Router/@Param` comments
> above every handler (`make swagger`, output in `docs/`). Enable the live UI with
> `enable_swagger_endpoint: true` (served at `/docs.html` and `/docs/index.html`).

## 7.1 Starting the server

```
aptly api serve [-listen=:8080 | -listen=unix:///run/aptly.sock] [-no-lock]
```

* Verifies `rootDir` is accessible (`utils.DirIsAccessible`).
* If systemd passed exactly one listener fd (`systemd/activation`), it is used; more than one
  panics ("not supported").
* Otherwise listens on TCP or a Unix socket (the socket file is removed and recreated).
* On `SIGINT/SIGTERM`: prints *"Shutdown signal received, waiting for background tasks…"*,
  `TaskList().Wait()`, then `server.Shutdown`.
* `-no-lock`: the server does **not** hold the LevelDB lock while idle (see §7.5).

> **Security:** the API has **no authentication or authorisation** of its own. Anyone who can
> reach the port can create, delete and publish repositories, upload files and add GPG keys.
> Bind to localhost / a Unix socket, or put it behind a reverse proxy providing TLS + auth (see
> [Operations](12-operations.md#124-securing-the-api)). Uploaded files and paths are
> sanitised (`verifyPath` rejects `.`/`..`, `utils.SanitizePath`), but the trust boundary is the
> network.

## 7.2 Request pipeline

```
HTTP request
  → gin.Logger (or JSONLogger when log_format=json) → gin.Recovery → gin.ErrorLogger
  → [swagger redirect]           (/docs → /docs.html, only if enabled)
  → [metrics collectors]         (only if enable_metrics_endpoint)
  → /api group
       → [-no-lock DB acquire/release middleware]
       → handler
             1. bind + validate JSON (c.Bind), path/query params
             2. cheap validation & 404 checks using a *fresh CollectionFactory*
             3. maybeRunTaskInBackground(name, resources, func(out, detail){ … })
                 - inside the closure: NEW collection factory, re-load objects, mutate, save
             4. respond: 200/201 + JSON  (sync)   or   202 + Task JSON (async)
```

Key conventions:

* Handlers get a **shallow** view for routing/404 decisions, then **re-load inside the task**
  after resource locks are acquired. This avoids TOCTOU races between concurrent requests.
* Errors use `AbortWithJSONError(c, status, err)` → JSON `{"error": "message"}`.
* `router.UseRawPath = true`: percent-encoded slashes in path parameters are preserved.
  For publish endpoints the *prefix* parameter uses an escaping scheme (`slashEscape`):
  `_` → `/` and `__` → `_`; prefix `.` denotes the root. e.g.
  `/api/publish/:./stable` (root prefix), `/api/publish/debian_pool/bookworm` for prefix
  `debian/pool`. Use `storage:prefix` form (e.g. `s3:mybucket:prod`) for non-default storage.
* `?_async=true` (`1`, `yes`, … — anything that is not `n/no/f/false/0/off`) runs the
  operation in the background. The obsolete config key `async_api` sets the default.

## 7.3 The task system

Package `task` (`list.go`, `task.go`, `resources.go`, `output.go`).

### Why tasks exist

Mutating operations touch shared state (a local repo's ref list, a mirror's contents, the
published tree, the package pool). Two concurrent `repo add` calls on the same repo would lose
updates. The task list **serialises conflicting operations** while allowing unrelated ones to
run in parallel.

### Data model

```go
type Task struct {
    Name string ; ID int ; State State  // IDLE | RUNNING | SUCCEEDED | FAILED
    Resources []string                    // exclusive resource keys
    output *Output                        // thread-safe text buffer (aptly.Progress impl)
    detail *Detail                        // atomic.Value with structured progress
    processReturnValue *ProcessReturnValue // {Code int, Value interface{}}
    err error
}
type Process func(out aptly.Progress, detail *Detail) (*ProcessReturnValue, error)
```

* `Output` implements `aptly.Progress`, so the same `deb.*` code that draws progress bars in
  the terminal writes plain text into a buffer retrievable at `/api/tasks/:id/output`
  (bars are no-ops; `Printf` appends lines).
* `Detail` carries machine-readable progress (e.g. remaining packages during cleanup, publish
  counters) at `/api/tasks/:id/detail`.

### Scheduling algorithm

```
RunTaskInBackground(name, resources, process):
    lock list
    task := new Task{ID: ++counter, State: IDLE}
    append to list
    if no running task uses any of `resources` (ResourcesSet.UsedBy == ∅):
          mark resources in use by task ; send task to the `queue` channel
    else:
          leave IDLE (waits)
    return copy of task
consumer goroutine:
    for task := range queue: set RUNNING ; go run process()
      on completion: set SUCCEEDED/FAILED, append "Task succeeded/failed…" to output,
                     free its resources, Done() wait-groups,
                     then scan IDLE tasks in order — the first one whose resources are now
                     free is marked in use and enqueued
```

Consequences:

* **Conflicting requests wait; they are not rejected.** (`runTaskInBackground` has a
  `ResourceConflictError` return path that the current `List` never uses; a 409 is therefore
  not produced in practice.) A synchronous HTTP request simply blocks until its turn.
* Scheduling is FIFO among *runnable* tasks with a "first-fit" scan on each completion. Only one
  waiting task is released per completion event, so long queues of independent tasks progress as
  completions occur.
* IDs are process-local counters (reset on restart); tasks are kept in memory until
  `DELETE /api/tasks/:id` or `POST /api/tasks-clear`. Synchronous requests delete their own task
  after returning the result.

### Resource keys

| Key | Meaning | Used by |
|-----|---------|---------|
| `R<uuid>` (`RemoteRepo.Key()`) | one mirror | mirror update/edit/drop |
| `L<uuid>` (`LocalRepo.Key()`) | one local repo | repo add/remove/copy/edit/drop/include |
| `S<uuid>` | one snapshot | snapshot ops |
| `U<storage:prefix>>><dist>` (`PublishedRepo.Key()`) | one publication | publish ops |
| filesystem paths of the upload dir being consumed | uploads | `repos/:name/file/...` includes the source path as a resource, so the same upload dir can't be imported twice concurrently |
| `__alllocalrepos__` | every local repo | `include` with a templated repo name |
| `__all__` | everything | `db cleanup` |

`ResourcesSet.UsedBy` implements the wildcards: requesting `__all__` conflicts with any used
resource; requesting `__alllocalrepos__` conflicts with any key beginning with `L`; and if
someone already holds a wildcard, everything conflicts with it.

### Sync vs async responses

| Mode | Response | How |
|------|----------|-----|
| **Synchronous** (default) | Final status/body of the operation (`200`, `201`, `4xx/5xx`) | `maybeRunTaskInBackground` schedules the task, `WaitForTaskByID`, reads `ProcessReturnValue{Code,Value}` and the error, deletes the task, replies |
| **Asynchronous** (`_async=true`) | `202 Accepted` + JSON of the `Task` (`ID`, `Name`, `State`, `Resources`) | client polls `/api/tasks/:id`, or blocks on `/api/tasks/:id/wait`, then reads `/output`, `/return_value` |

Example:

```bash
# start a long mirror update without holding the connection
TASK=$(curl -s -X PUT -H 'Content-Type: application/json' -d '{}' \
       'http://localhost:8080/api/mirrors/bookworm-main?_async=true' | jq .ID)
curl -s http://localhost:8080/api/tasks/$TASK/wait          # blocks until done → task JSON
curl -s http://localhost:8080/api/tasks/$TASK/output        # text log
curl -s http://localhost:8080/api/tasks/$TASK/return_value  # {"Code":..,"Value":..}
curl -s -X DELETE http://localhost:8080/api/tasks/$TASK
```

## 7.4 Uploading files

The API cannot read your disk, so package files are first **uploaded** into
`rootDir/upload/<dir>/`:

```bash
curl -X POST -F file=@pkg_1.0_amd64.deb http://localhost:8080/api/files/mydir
curl http://localhost:8080/api/files/mydir            # list files
curl -X DELETE http://localhost:8080/api/files/mydir/pkg_1.0_amd64.deb
```

Then ingest with `POST /api/repos/:name/file/:dir[/:file]` (like `repo add`; removes the
files afterwards unless `?noRemove=1`) or `POST /api/repos/:name/include/:dir[/:file]` (like
`repo include`, for `.changes`; `noRemoveFiles`, `forceReplace`, `acceptUnsigned`,
`ignoreSignature`). `PUT /api/files/:dir/:file` uploads a single file by name.
`apiFilesUpload` streams each part to disk and `fsync`s it (`syncFile`) so `ENOSPC` is reported.
The Prometheus counter `aptly_api_files_uploaded_total{directory=…}` tracks uploads.

## 7.5 Database locking in API mode

LevelDB allows one opener per directory. Modes:

| Mode | Behaviour |
|------|-----------|
| default | The server opens the DB at first use and keeps it open until exit. No other aptly process can use the same `rootDir` concurrently (they retry for `db-open-attempts` × ~10 s) |
| `-no-lock` | A middleware wraps every `/api` request: `acquiredb` opens the DB if there are 0 clients, `releasedb` closes it when the last in-flight client leaves (`acquireDatabase` goroutine with a `dbRequests` channel). Tasks additionally acquire/release a DB reference (`runTaskInBackground`). This lets you run CLI commands *between* API bursts, at the cost of re-opening LevelDB often |
| etcd backend | No exclusive lock; multiple servers/CLIs can share state — resource locks are still **per process** (tasks in different processes don't see each other), so coordinate externally |

## 7.6 Endpoint catalogue

Base path `/api`. `{name}` = object name; JSON bodies use PascalCase keys. All mutating
endpoints accept `?_async=true`.

### Status & meta

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/api/version` | `{"Version": "…"}` |
| GET | `/api/ready` | 200 `Aptly is ready` once the router is built, else 503 |
| GET | `/api/healthy` | 200 always (liveness) |
| GET | `/api/storage` | Disk usage of `rootDir` filesystem: `Total`, `Free` (MiB), `PercentFull` |
| GET | `/api/metrics` | Prometheus metrics (if enabled); refreshes per-repo package counts first |
| GET | `/docs.html`, `/docs/*` | Swagger UI (if enabled) |

### Local repositories

| Method | Path | Notes |
|--------|------|-------|
| GET | `/api/repos` | list |
| POST | `/api/repos` | `{"Name","Comment","DefaultDistribution","DefaultComponent","FromSnapshot"}` — `FromSnapshot` seeds from a snapshot |
| GET/PUT/DELETE | `/api/repos/{name}` | show / edit (`Name`,`Comment`, defaults — pointer fields, only provided ones change) / drop (`?force=1` when used as a source; refused if published) |
| GET | `/api/repos/{name}/packages` | `?q=<query>` `?withDeps=1` `?format=details` `?maximumVersion=1` |
| POST/DELETE | `/api/repos/{name}/packages` | body `{"PackageRefs":[keys…]}` add/remove by key |
| POST | `/api/repos/{name}/file/{dir}[/{file}]` | import uploaded `.deb/.dsc` (`noRemove`, `forceReplace`) |
| POST | `/api/repos/{name}/include/{dir}[/{file}]` | import via `.changes` |
| POST | `/api/repos/{name}/copy/{src}/{file}` | copy package(s) matching query `file` from repo `src` (`{"with-deps":bool,"dry-run":bool}`) |
| POST | `/api/repos/{name}/snapshots` | snapshot of the repo |

### Mirrors

| Method | Path | Notes |
|--------|------|-------|
| GET/POST | `/api/mirrors` | list / create (`Name, ArchiveURL, Distribution, Components, Architectures, Filter, FilterWithDeps, DownloadSources/Udebs/Installer/AppStream, Keyrings, SkipComponentCheck, SkipArchitectureCheck, IgnoreSignatures`, `ppa:` URLs allowed) |
| GET | `/api/mirrors/{name}` | show |
| POST | `/api/mirrors/{name}` | **edit** (POST!) |
| PUT | `/api/mirrors/{name}` | **update** = download (long-running; use `_async`). Body (all optional): `Name` (rename), `Keyrings`, `IgnoreChecksums`, `IgnoreSignatures`, `ForceUpdate`, `SkipExistingPackages`, `LatestOnly` — the same switches as `mirror update` |
| DELETE | `/api/mirrors/{name}` | drop (`?force=1`) |
| GET | `/api/mirrors/{name}/packages` | like repo packages |
| POST | `/api/mirrors/{name}/snapshots` | snapshot of the mirror |

### Snapshots

| Method | Path | Notes |
|--------|------|-------|
| GET | `/api/snapshots` | list (`?sort=name` (default) or `?sort=time`) |
| POST | `/api/snapshots` | `{"Name","Description","SourceSnapshots":[…],"PackageRefs":[…]}` — merge sources and/or add explicit package refs |
| GET/PUT/DELETE | `/api/snapshots/{name}` | show / rename & describe / drop (`?force=1`) |
| GET | `/api/snapshots/{name}/packages` | search/list |
| GET | `/api/snapshots/{name}/diff/{withSnapshot}` | `[{"Left":key\|null,"Right":key\|null}]`; `?onlyMatching=1` |
| POST | `/api/snapshots/{name}/merge` | body `{"Sources":[…]}` — the URL snapshot is the **destination name**; options are **query parameters**: `?latest=1`, `?no-remove=1` |
| POST | `/api/snapshots/{name}/pull` | body `{"Source","Destination","Queries":[…],"Architectures":[…]}`; options are **query parameters**: `?no-deps=1`, `?no-remove=1`, `?all-matches=1`, `?dry-run=1` |

### Publishing

| Method | Path | Notes |
|--------|------|-------|
| GET | `/api/publish` | list all publications |
| GET | `/api/publish/{prefix}/{distribution}` | show |
| POST | `/api/publish[/{prefix}]` | create: `{"SourceKind":"snapshot\|local","Sources":[{"Component","Name"}],"Distribution","Architectures","Label","Origin","Signing":{…},"SkipContents","SkipBz2","SkipCleanup","AcquireByHash","SignedBy","MultiDist","Version","NotAutomatic","ButAutomaticUpgrades","ForceOverwrite"}`; 201 |
| PUT | `/api/publish/{prefix}/{distribution}` | switch (snapshots: `{"Snapshots":[{"Component","Name"}]}`) or refresh local repo; same flags |
| DELETE | `/api/publish/{prefix}/{distribution}` | drop (`?force=1`, `?SkipCleanup=1`) |
| POST/GET/PUT/DELETE | `/api/publish/{prefix}/{distribution}/sources` | staged sources: add one / list pending changes / replace all / discard pending |
| PUT/DELETE | `/api/publish/{prefix}/{distribution}/sources/{component}` | update / remove one staged component |
| POST | `/api/publish/{prefix}/{distribution}/update` | apply staged revision and publish |

The `Signing` object: `{"Skip":bool,"GpgKey":"ID1, ID2","Keyring":"","SecretKeyring":"",
"Passphrase":"","PassphraseFile":""}`. The API **always** runs GPG in batch mode (no prompt),
so an agent or `PassphraseFile`/`Passphrase` is required for protected keys.

### Files, packages, GPG, cloud listing, graph, cleanup

| Method | Path | Notes |
|--------|------|-------|
| GET | `/api/files` | list upload directories |
| POST/GET/DELETE | `/api/files/{dir}` | upload (multipart `file=`) / list files / delete dir |
| PUT/DELETE | `/api/files/{dir}/{file}` | upload one file / delete one |
| GET | `/api/packages` | search across the whole DB (`?q=`, `?format=details`) |
| GET | `/api/packages/{key}` | full detail of one package by key (URL-escaped) |
| GET | `/api/gpg/keys` | list keys in a keyring (`?keyring=`, default `trustedkeys.gpg`) |
| POST/DELETE | `/api/gpg/key` | body `{"Keyring", "GpgKeyArmor" \| "Keyserver"+"GpgKeyID"}` to import; delete takes `Keyring` + key IDs — used to trust mirror signing keys |
| GET | `/api/s3`, `/api/gcs`, `/api/jfrog` | list configured endpoints |
| GET | `/api/graph.{ext}` | Graphviz render of the object graph by shelling out to `dot -T<ext>` (`ext` = `png`, `svg`, …; `dot`/`gv` return the source); `?layout=horizontal\|vertical` |
| POST | `/api/db/cleanup` | GC (see §5.6); takes resource `__all__` |

### Tasks

| Method | Path | Notes |
|--------|------|-------|
| GET | `/api/tasks` | all tasks |
| POST | `/api/tasks-clear` | drop finished ones |
| GET | `/api/tasks-wait` | block until *all* tasks complete |
| GET | `/api/tasks/{id}` | task JSON |
| GET | `/api/tasks/{id}/wait` | block until it completes |
| GET | `/api/tasks/{id}/output` | plain text log |
| GET | `/api/tasks/{id}/detail` | structured progress |
| GET | `/api/tasks/{id}/return_value` | `{Code, Value}` |
| DELETE | `/api/tasks/{id}` | only finished tasks |

### Built-in file serving (`serve_in_api_mode: true`)

Outside `/api`: `GET /repos/` lists configured *filesystem* endpoints as HTML and
`GET /repos/{storage}/{path…}` serves published files from them, so a single process can be
both API and (basic) repository server. (`aptly serve`, in contrast, exposes only the default
`public/` tree.)

## 7.7 Metrics (`api/metrics.go`)

Enabled with `enable_metrics_endpoint: true`; scraped at `/api/metrics` and registered through
`MetricsCollectorRegistrar`:

| Metric | Type | Labels |
|--------|------|--------|
| `aptly_api_http_requests_in_flight` | gauge | method, path |
| `aptly_api_http_requests_total` | counter | code, method, path |
| `aptly_api_http_request_size_bytes` / `response_size_bytes` | summary | code, method, path |
| `aptly_api_http_request_duration_seconds` | histogram | code, method, path |
| `aptly_build_info` | gauge (=1) | version, goversion |
| `aptly_api_files_uploaded_total` | counter | directory |
| `aptly_repos_package_count` | gauge | source, distribution, component (published packages) |

The `path` label uses only the first two path segments (`/api/repos`) to bound cardinality.

## 7.8 Client recipes

**Publish new package end-to-end (synchronous):**

```bash
H='Content-Type: application/json'
curl -X POST -F file=@app_1.2_amd64.deb   localhost:8080/api/files/incoming
curl -X POST                                localhost:8080/api/repos/app/file/incoming
curl -X POST -H "$H" -d '{"Name":"app-1.2"}' localhost:8080/api/repos/app/snapshots
curl -X PUT  -H "$H" -d '{"Snapshots":[{"Component":"main","Name":"app-1.2"}],"Signing":{"GpgKey":"ABCD1234","PassphraseFile":"/etc/aptly.pass"}}' \
     localhost:8080/api/publish/prod/stable
```

**Poll a long mirror update:**

```bash
curl -X PUT -H "$H" -d '{}' 'localhost:8080/api/mirrors/deb?_async=true'
# → {"Name":"Update mirror deb","ID":7,"State":0,"Resources":["Rabc…"]}
until [ "$(curl -s localhost:8080/api/tasks/7 | jq .State)" -ge 2 ]; do sleep 5; done
```

`State`: `0` idle/queued, `1` running, `2` succeeded, `3` failed.
