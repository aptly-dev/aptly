# 11. Queries, Dependencies & Version Comparison

## 11.1 Package query language

Used by `mirror -filter`, `snapshot filter/pull`, `repo copy/move/remove/import`, `package search`,
`*/search` commands, `uploaders.json` conditions and the API `?q=`. Parsed by
`query.Parse(string)` (`query/lex.go` → `query/syntax.go`) into a tree of `deb.PackageQuery`
(`deb/query.go`).

### Grammar (recursive descent, from `syntax.go`)

```
Query  := A     | A '|' Query            OR   (lowest precedence)
A      := B     | B ',' A                AND
B      := C     | '!' B                  NOT
C      := '(' Query ')' | D
D      := <field> [condition]  | <package-name> [condition] [arch] | <name>_<version>_<arch>
condition := '(' [operator] value ')'
operator  := '<<' | '<' | '<=' | '>' | '>>' | '>=' | '=' | '%' | '~'
arch      := '{' architecture '}'
```

Tokens: `( ) | , ! << <= < >> >= > = % ~ { }` and strings.
Note **precedence**: `,` (AND) binds tighter than `|` (OR); use parentheses.

### Term kinds

| Term | Example | Becomes | Meaning |
|------|---------|---------|---------|
| Package name | `nginx` | `DependencyQuery` | any version of package named `nginx` (any architecture) |
| Name + version condition | `nginx (>= 1.24)` | `DependencyQuery` | uses Debian dependency semantics (see below) |
| Name + arch | `nginx {amd64}` | `DependencyQuery` | limited to architecture |
| Exact package key | `nginx_1.24.0-1_amd64` | `PkgQuery` | matches name, version, arch exactly. Recognised when there is no condition and the string has two `_` |
| Field query | `Section (= web)` | `FieldQuery` | compare a control field. Field name must start with an **upper-case** letter (and contain no `_`) or with `$` |
| Regexp | `Name (~ ^lib.*-dev$)` / `nginx (~ ^1\.2)` | `FieldQuery`/`DependencyQuery` with `Regexp` | `~` |
| Pattern (glob) | `Name (% nginx-*)` | glob | `%` uses shell-style `filepath.Match` |
| Special fields | `$Version`, `$Architecture`, `$PackageType`, `$Source`, `$SourceVersion` | | see below |

**Operators.** Following Debian dependency syntax: `<<` strictly less, `>>` strictly greater; the
lexer maps single `<` to *less-or-equal* and `>` to *greater-or-equal* (`<=`, `>=` explicitly the
same) — the historical Debian meaning. `=` equal (default when no operator given inside
parentheses), `%` glob pattern, `~` regular expression (Go RE2).

### Field names available

`Package.GetField` (`deb/package.go`) answers, case-sensitively (for source packages
`Architecture` returns the real `Architecture:` of the source, not `source`):
`Name`, `Version`, `Architecture`, `Source`, `Depends`, `Pre-Depends`, `Suggests`, `Recommends`,
`Provides`, `Build-Depends`, `Build-Depends-Indep`, plus any other control field from the stanza
(`Section`, `Priority`, `Maintainer`, `Description`, `Filename` …), and specials:

| Special | Value |
|---------|-------|
| `$Version` | the package version, evaluated like a dependency on the package's own name (Debian version ordering) |
| `$Source` | source package name (own name if there is no `Source:`); **empty for source packages** |
| `$SourceVersion` | version from `Source: name (version)`, else the package's own version; empty for source packages |
| `$Architecture` | the architecture; an `=` comparison goes through `MatchesArchitecture`, so `$Architecture (= amd64)` also matches `Architecture: all` packages |
| `$PackageType` | `deb`, `udeb` or `source` (installer pseudo-packages report `deb`) |

> **Quirk:** every field comparison except `%` and `~` (and "field is non-empty" when no operator is
> given) is evaluated with `CompareVersions`, i.e. with *Debian version ordering*, even for non-version
> fields such as `Section (= web)`. Equality behaves as expected; `<`/`>` on textual fields follow version
> rules, which is rarely what you want.

### Examples

```
nginx                               all versions of package nginx
nginx (>= 1.20), !Name (nginx-extras)   nginx ≥ 1.20, and never the nginx-extras package
Priority (required) | Priority (important)
$PackageType (source)
Name (~ ^python3-.*), $Version (>= 3.0)
(Section (= web) | Section (= net)), !Depends (~ apache2)
Maintainer (% *@example.com)
libssl3_3.0.11-1_amd64              one exact binary package
```

Errors are reported by `Parse` as `parsing failed: …` (e.g. *unexpected token … expecting end of query*)
before any DB access; mirror filters are parsed at creation and at update.

### Evaluation strategy (`PackageList.Scan / Query`)

Each `PackageQuery` has `Matches(PackageLike)`, `Fast(catalog)` and `Query(catalog) *PackageList`:

* `DependencyQuery.Fast()` is true when the catalog supports searching by dependency — a
  `PackageList` with an index. Then `Query` uses **binary search on the name index** (`Search`) and,
  because `providesIndex` exists, also virtual packages (`Provides`).
* `AndQuery`: fast if **either** side is fast — evaluate the fast side, then `Scan` that (small)
  result with the other side.
* `OrQuery`: fast only if **both** sides are fast; results are unioned. Otherwise `Scan`.
* `NotQuery`, `FieldQuery`, `MatchAllQuery`: full `Scan` calling `Matches` per package.
* `PkgQuery` (`name_version_arch`) is a direct key lookup (`SearchByKey`).

## 11.2 Dependencies (`deb/package_deps.go`, `deb/list.go`)

### Parsing

`ParseDependency("libc6 (>= 2.34)")` → `Dependency{Pkg, Relation, Version, Architecture}`.
Alternatives `a | b` are split by `ParseDependencyVariants`. Architecture may be given as multiarch
`pkg:arch` (`pkg:any` = no restriction) or with aptly's own suffix `pkg (= 1.0) {arch}` (used by generated
source dependencies such as `foo (= 1.0) {source}`). Relations inside parentheses: `<<` (strictly less),
`<`/`<=` (both = less-or-equal), `=`, `>`/`>=` (both = greater-or-equal), `>>`; no operator = equal.
Relations: `VersionDontCare, Less, LessOrEqual, Equal, GreaterOrEqual, Greater, PatternMatch, Regexp`.

### Which fields are followed (`Package.GetDependencies(options)`)

Always `Depends` + `Pre-Depends`; optionally (bitmask `deb.DepFollow*`, set from flags/config):

| Option | Adds |
|--------|------|
| `DepFollowRecommends` | `Recommends` |
| `DepFollowSuggests` | `Suggests` |
| `DepFollowBuild` (internal) | `Build-Depends`, `Build-Depends-Indep` |
| `DepFollowSource` | a dependency on the package's own *source* package: `src (= version) {source}` |
| `DepFollowAllVariants` | for `a \| b` keep looking for every alternative even if one is already satisfied |
| `DepVerboseResolve` | log decisions |

### Verification & closure

`PackageList.VerifyDependencies(options, architectures, sources, progress)`:

```
for arch in architectures:
    cache := {}                          # dependency-hash → satisfied?
    for p in list where p.MatchesArchitecture(arch):
        for depString in p.GetDependencies(options):
            variants := ParseDependencyVariants(depString)   # alternatives
            for v in variants:                                # arch defaults to `arch`
                satisfied := sources.Search(v, allMatches=false, searchProvided=true) != nil   (cached)
                if satisfied and !FollowAllVariants: variantsMissing = nil ; break
                if !satisfied: variantsMissing += v
            missing += variantsMissing
missing = dedupe(missing)
```

`PackageList.Search(dep, allMatches, searchProvided)` binary-searches the name index and filters by
architecture (`MatchesDependency`: an `all` package satisfies any architecture but `source`), version
relation (`versionSatisfiesDependency` via `CompareVersions`) and, when `searchProvided`, virtual packages
from `Provides`. In `Provides`, only `=` versions are allowed (others are logged as a warning and ignored);
an **unversioned** `Provides: foo` is treated as providing `foo` at the *providing package's own version*.

`PackageList.Filter(FilterOptions)` implements **dependency closure**:

```
result := ⋃ query.Query(list) for each query
if WithSources: add matching source packages (via Source: field / SourceRegex)
if WithDependencies:
    depSource := Source ∪ result
    loop while something was added:
        missing := result.VerifyDependencies(opts, arch, depSource)
        for dep in missing:
            if !FollowAllVariants and result.Search(dep) != nil: continue     # satisfied now
            for p in list.Search(dep, allMatches=true, searchProvided=true):
                add p to result and depSource ; (unless FollowAllVariants → only first)
```

The **`Source`** in `FilterOptions` is the set that dependencies are considered *already satisfied*
by (e.g. `snapshot pull` passes the destination snapshot, `repo copy -with-deps` passes the target repo,
mirror filters pass an empty list so *everything* needed is pulled in).

### Practical consequences

* Filtering a mirror with `-filter-with-deps` will pull `Depends`/`Pre-Depends` closure only — use
  `-dep-follow-recommends` etc. globally if you want `apt install` with default recommends to work offline.
* Missing dependencies are **not an error** during filtering (they are logged only with
  `-dep-verbose-resolve`); use `aptly snapshot verify` to audit a snapshot afterwards.
* Virtual packages resolved through `Provides` may pick arbitrary providers; with
  `-dep-follow-all-variants` *all* providers/alternatives are included (larger result).

## 11.3 Debian version comparison (`deb/version.go`)

`CompareVersions(a, b)` → `-1, 0, 1` implements the Debian Policy algorithm
`[epoch:]upstream_version[-debian_revision]`:

1. Split epoch (text before the first `:`), then the revision — `parseVersion` splits at the **first**
   `-` (Debian Policy says the *last* hyphen; the two differ only for upstream versions that themselves
   contain hyphens) — leaving the upstream part.
2. Compare epoch (numeric; missing = 0), then upstream, then revision, each with `compareVersionPart`:
   alternate **non-digit run** (compared with `compareLexicographic`) and **digit run** (compared
   numerically), left to right.
3. `compareLexicographic`: letters sort before non-letters; `~` sorts before everything, even the end of
   the string (so `1.0~rc1 < 1.0`); otherwise ASCII order.

`PackageList` orders packages by name ascending then **version descending** (newest first), which is
why `snapshot pull` "first match wins" picks the highest version, and `FilterLatest`/
`FilterLatestRefs` keep only the maximum.

Note the distinction between **key order** (byte-wise on `P<arch> <name> <version> <hash>`, used inside
`PackageRefList`) and **version order** (`CompareVersions`, used when semantics matter).
