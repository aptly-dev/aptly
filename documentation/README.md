# aptly Documentation

Complete, code-level documentation of **aptly**, the Debian repository management tool.
It explains what the tool does, how it is built, how every subsystem works internally,
and how to run, extend and test it.

> These documents describe the code in this repository. File and function names are given
> so you can jump straight to the source. Statements were verified against the code at the
> time of writing; when in doubt, the source is authoritative.

## Reading order

| # | Document | Read it to learn... |
|---|----------|---------------------|
| 1 | [Overview & Concepts](01-overview-and-concepts.md) | What aptly is, the object model (mirror / local repo / snapshot / published repo), typical workflows |
| 2 | [Architecture](02-architecture.md) | Layers, Go packages, dependency graph, runtime modes, key design decisions |
| 3 | [Data Model & Database](03-data-model-and-database.md) | Key/value layout, encodings, `PackageRefList`, collections, DB backends |
| 4 | [Package Pool & Published Storage](04-storage.md) | Content-addressed pool, publishing back-ends (filesystem, S3, GCS, Azure, Swift, JFrog) |
| 5 | [Core Workflows](05-workflows.md) | Step-by-step internals: mirror update, snapshot, merge/pull, add/include, cleanup |
| 6 | [Publishing Internals](06-publishing.md) | How `Packages`, `Release`, `Contents`, by-hash files and signatures are generated |
| 7 | [REST API & Task System](07-rest-api-and-tasks.md) | Router, sync/async execution, resource locking, endpoint catalogue |
| 8 | [CLI & Context](08-cli-and-context.md) | Command tree, `AptlyContext`, lifecycle of a command, DB open/close |
| 9 | [Configuration Reference](09-configuration.md) | Every config key, defaults, examples for all storage back-ends |
| 10 | [Signing & Verification](10-signing-and-verification.md) | GPG providers, Release signing, mirror signature verification |
| 11 | [Queries, Dependencies & Versions](11-queries-and-dependencies.md) | Query language grammar, dependency resolution, Debian version comparison |
| 12 | [Operations Guide](12-operations.md) | Deployment, locking, backups, recovery, performance, troubleshooting |
| 13 | [Development Guide](13-development.md) | Building, unit & system tests, CI, how to add commands / endpoints / back-ends |
| 14 | [Glossary](14-glossary.md) | Terms used throughout |

## One-paragraph summary

aptly keeps **metadata** (which packages exist, which repository/snapshot contains which
package) in an embedded key/value database and **package files** in a de-duplicated,
content-addressed *pool* on disk (or Azure Blob storage). Everything you can do — mirror a
remote archive, build a local repository from `.deb`/`.dsc`/`.changes` files, freeze state as
an immutable *snapshot*, filter/merge/pull between snapshots, and *publish* a snapshot as a
signed, apt-consumable repository to a directory, S3, GCS, Azure, Swift or Artifactory — is a
manipulation of lists of package *references* plus, at publish time, a generation of
`Packages`/`Release` index files and links/copies of pool files. It is driven either by the
`aptly` CLI or by the same logic exposed through a REST API served by `aptly api serve`.

## Other documentation in this repository

* [`docs/`](../docs) — API reference fragments used to generate the Swagger/OpenAPI spec
  (`make swagger`) and the embedded `/docs` page.
* [`files/README.md`](../files/README.md) — details of the on-disk pool layout.
* `man/aptly.1` — man page generated from the CLI help (`_man/gen.go`).
* Upstream user docs: <https://www.aptly.info/>.
