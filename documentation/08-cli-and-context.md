# 8. CLI & Application Context

## 8.1 Entry point and command dispatch

```go
// main.go
//go:embed VERSION            var Version string
//go:embed debian/aptly.conf  var AptlyConf []byte
func main() {
    aptly.Version = Version ; aptly.AptlyConf = AptlyConf
    os.Exit(cmd.Run(cmd.RootCommand(), os.Args[1:], true))
}
```

The CLI framework is [`smira/commander`](https://github.com/smira/commander) (a fork of
`gonuts/commander`) with [`smira/flag`](https://github.com/smira/flag) (a fork of the standard
`flag` package that additionally supports `IsSet`, so "flag not given" can be distinguished from
"flag given with the default value" — vital for *config default vs. explicit flag* logic).

`cmd.Run(cmd, args, initContext)` (`cmd/run.go`):

```
defer recover(): a *context.FatalError panic → print "ERROR: msg" to stderr, return its exit code
flags, args, err := cmd.ParseFlags(cmdArgs)        // global + subcommand flags
if initContext: InitContext(flags)  (creates AptlyContext)  ; defer ShutdownContext()
context.UpdateFlags(flags)                          // context.flags = flags of the leaf command
err = cmd.Dispatch(args)                            // walks the command tree, calls Run(cmd,args)
if err != nil: Fatal(err)                           // exit 1  (2 for usage / flag errors)
```

Exit codes: `0` success, `1` runtime error, `2` command/flag usage error
(`commander.ErrCommandError`, `ErrFlagError`).

### Command tree (`cmd/*.go`; every command is built by a `makeCmd…()` function)

```
aptly
├── api        serve
├── config     show
├── db         cleanup | recover
├── graph
├── mirror     create | list | show | drop | update | rename | edit | search
├── package    search | show
├── publish    snapshot | repo | list | show | drop | update | switch
│              └─ source  add | drop | list | remove | replace | update
├── repo       create | add | copy | drop | edit | import | list | move | remove
│              | show | rename | search | include
├── serve
├── snapshot   create | list | show | verify | pull | diff | merge | drop | rename
│              | search | filter
├── task       run
└── version
```

Each `makeCmd…` returns a `*commander.Command{Run, UsageLine, Short, Long, Flag}`; help text is
what `aptly help <cmd>` prints, and `man/aptly.1` is generated from the same strings
(`_man/gen.go` + `man/aptly.1.ronn.tmpl`). Bash/zsh completion scripts live in `completion.d/`.

### Global flags (`cmd.RootCommand`)

| Flag | Meaning |
|------|---------|
| `-config=path` | Config file (else `~/.aptly.conf`, `/usr/local/etc/aptly.conf`, `/etc/aptly.conf`) |
| `-architectures=a,b` | Override configured default architectures |
| `-dep-follow-suggests`, `-dep-follow-recommends`, `-dep-follow-all-variants`, `-dep-follow-source`, `-dep-verbose-resolve` | Dependency-resolution switches (override config) |
| `-db-open-attempts=N` | Retries while the DB is locked by another process (default 10) |
| `-gpg-provider=gpg\|gpg1\|gpg2\|internal` | PGP implementation |
| `-cpuprofile -memprofile -memstats -meminterval` | Only when built with `EnableDebug` |

Precedence rule used everywhere: **explicit flag → config value → built-in default**
(`LookupOption(default, flags, name)` returns the flag value only if `flags.IsSet(name)`).

### Reading queries from files

Package queries and mirror filters may be given as `@file` or `@-` (stdin) via
`GetStringOrFileContent` / `AddStringOrFileFlag` (`cmd/string_or_file_flag.go`).

### Output conventions

Human-readable text goes through `context.Progress()` (`Printf`, `ColoredPrintf`), while
`-json` / `-raw` flags on `list/show` commands print machine-readable output directly. `ColoredPrintf`
understands `@r @g @y @!` colour markup (see `console/progress.go`).

## 8.2 `AptlyContext` — the composition root (`context/context.go`)

`AptlyContext` owns every shared resource and creates each **lazily on first use**, guarded by a
single mutex (a `Lock()`ed public method calls an unlocked `_private` twin so internal callers
don't deadlock). It embeds a `context.Context` so it can be passed straight to `Download(ctx,…)`
and be cancelled by signals.

| Accessor | Lazily creates | Notes |
|----------|----------------|-------|
| `Config()` | `utils.Config` | see §8.3 |
| `Progress()` | `console.Progress` | started on creation; shut down in `Shutdown()` |
| `Downloader()` / `NewDownloader(p)` | `http.NewDownloader` or `NewGrabDownloader` | choice from `-downloader` flag, else `downloader` config; limit from `-download-limit` (KiB/s) else `download_limit`; tries = `-max-tries` else `download_retries + 1` |
| `Database()` | LevelDB or etcd `database.Storage` | `Open()` with retry on `resource temporarily unavailable`: up to `-db-open-attempts` (or config `database_open_attempts` unless it is `-1`) with `10s ± 1s` Gaussian jitter |
| `NewCollectionFactory()` | `deb.CollectionFactory` over `Database()` | one per command/request; `Fatal`s if DB can't open |
| `PackagePool()` | local or Azure pool | path from `packagepool_storage.path` else `rootDir/pool` |
| `GetPublishedStorage(name)` | one per configured endpoint | `""` → `rootDir/public` hardlink; `filesystem:x`, `s3:x`, `gcs:x`, `swift:x`, `azure:x`, `jfrog:x`; cached in a map |
| `GetSigner()`, `GetVerifier()` | `pgp.GoSigner`/`GoVerifier` or `GpgSigner`/`GpgVerifier` | provider from `-gpg-provider` else `gpg_provider` (`gpg`, `gpg1`, `gpg2`, `internal`); unknown ⇒ Fatal |
| `TaskList()` | `task.NewList()` | starts the consumer goroutine |
| `DependencyOptions()` | bitmask | computed once from flags/config (`deb.DepFollow*`) |
| `ArchitecturesList()` | `[]string` | `-architectures` else config |
| `UploadPath()`, `SkelPath()`, `DBPath()` | paths | `rootDir/upload`, `rootDir/skel`, `rootDir/db` |
| `CloseDatabase()` / `ReOpenDatabase()` | | temporarily release / reacquire the DB lock |
| `GoContextHandleSignals()` | wraps the embedded `Context` in `WithCancel`; first SIGINT/SIGTERM cancels | second signal kills default handling (`signal.Stop`) |
| `Shutdown()` | | stops task list, closes DB, drops downloader, shuts down progress, writes debug profiles |
| `Cleanup()` | | partial shutdown (downloader + progress) used between `task run` commands |

`FatalError`/`Fatal(err)` panic with a struct carrying exit code and message; `cmd.Run` (or any
`recover`) turns it into `ERROR: …` + exit status.

Note the `api` package keeps its own package-level `context` variable pointing to the same
`AptlyContext` (set in `Router(c)`), and `cmd` keeps one in `cmd/context.go`.

## 8.3 Configuration loading

`context.config()` (once, memoised by `configLoaded`):

1. If `-config` given → `utils.LoadConfig(path, &utils.Config)`; any error is fatal.
2. Else try in order `~/.aptly.conf`, `/usr/local/etc/aptly.conf`, `/etc/aptly.conf`:
   skip *not exist / permission denied*; stop at first success; any other error is fatal.
3. If nothing was found: print *"Config file not found, creating default config at ~/.aptly.conf"* and
   write the embedded `debian/aptly.conf` (or `root_dir: ""` if not embedded), then load it.
4. `utils.LoadConfig` first tries **JSON** (with comments/trailing commas tolerated by
   `JsonConfigReader`), and on failure re-reads as **YAML**; if both fail the error mentions both.
   `utils.Config` starts as a struct of defaults (see `utils/config.go`), so keys omitted in the file keep
   defaults.
5. Logging is initialised from `log_level`/`log_format` (`utils.SetupJSONLogger` or `SetupDefaultLogger`),
   and `context.StructuredLogging(true)` disables terminal progress bars.

`aptly config show [-yaml]` prints the effective config (default JSON; converting old JSON configs to
YAML is `aptly config show -yaml > aptly.yaml`). `ConfigStructure.GetRootDir()` expands a leading `~`.

Full key reference: [chapter 9](09-configuration.md).

## 8.4 Lifecycle of a typical command

```
aptly snapshot create s1 from mirror m1
  main → cmd.Run
    ParseFlags                          (global flags + "snapshot create" flags)
    InitContext                         NewContext(flags)   — nothing opened yet
    UpdateFlags(leaf flags)
    Dispatch → aptlySnapshotCreate
        context.NewCollectionFactory()  → Config() loads/creates config
                                        → Database() opens LevelDB (lock; retry if busy)
        RemoteRepoCollection.ByName("m1")   ← reads R… keys
        repo.CheckLock ; LoadComplete   ← reads E<uuid>
        deb.NewSnapshotFromRepository
        SnapshotCollection.Add          → batch: Put S<uuid>, Put E<uuid>
        fmt.Printf("Snapshot … created")
    ShutdownContext → Database().Close(), Progress.Shutdown()
  os.Exit(0)
```

## 8.5 The `console` package (`console/progress.go`)

`console.Progress` implements `aptly.Progress` with **one worker goroutine** draining a buffered
channel (capacity 100) of print tasks (`codePrint`, `codePrintStdErr`, `codeProgress`,
`codeHideProgress`, `codeFlush`, `codeStop`, `codeBarEnabled/Disabled`). Concurrent goroutines
(e.g. the download workers) therefore never interleave output. A `progressWorkerFactroy` picks a
terminal worker or a plain/structured-logging worker depending on `structuredLogging`.

* Progress bars use `cheggaaa/pb` and are created **only when stdout is a terminal**
  (`utils.RunningOnTerminal()`); otherwise `InitBar` is a no-op and only text is printed.
  `Write(p)` (the `io.Writer` side) advances the bar by bytes, which is how the downloader feeds it.
* `Printf` while a bar is displayed hides the bar, prints, then redraws it.
* `ColoredPrintf` handles `@r @g @y @b @!` … `@|` colour markup through `wsxiaoys/terminal/color`.
* `Flush()` blocks until all queued messages are written — called before spawning `gpg`, which may
  prompt on the terminal.

The REST counterpart is `task.Output` / `task.PublishOutput` (`task/output.go`): a mutex-protected
text buffer whose bar methods are no-ops, except that `PublishOutput` counts down
`RemainingNumberOfPackages` for the `BarPublishGeneratePackageFiles` bar so API clients can poll
publish progress from `/api/tasks/:id/detail`. `ColoredPrintf` there appends the text plus a newline
without colours.
