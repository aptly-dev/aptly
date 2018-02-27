# Contributing to aptly

:+1::tada: First off, thanks for taking the time to contribute! :tada::+1:

The following is a set of guidelines for contributing to [aptly](https://github.com/smira/aplty) and related repositories, which are hosted in the [aptly-dev Organization](https://github.com/aptly-dev) on GitHub.
These are just guidelines, not rules. Use your best judgment, and feel free to propose changes to this document in a pull request.

## What should I know before I get started?

### Code of Conduct

This project adheres to the Contributor Covenant [code of conduct](CODE_OF_CONDUCT.md).
By participating, you are expected to uphold this code.
Please report unacceptable behavior to [team@aptly.info](mailto:team@aptly.info).

### List of Repositories

* [smira/aptly](https://github.com/smira/aptly) - aptly source code, functional tests, man page
* [apty-dev/aptly-dev.github.io](https://github.com/aptly-dev/aptly-dev.github.io) - aptly website (https://www.aptly.info/)
* [aptly-dev/aptly-fixture-db](https://github.com/aptly-dev/aptly-fixture-db) & [aptly-dev/aptly-fixture-pool](https://github.com/aptly-dev/aptly-fixture-pool) provide
  fixtures for aptly functional tests

## How Can I Contribute?

### Reporting Bugs

1. Please search for similar bug report in [issue tracker](https://github.com/smira/aptly/issues)
2. Please verify that bug is not fixed in latest aptly nightly ([download information](https://www.aptly.info/download/))
3. Steps to reproduce increases chances for bug to be fixed quickly. If possible, submit PR with new functional test which fails.
4. If bug is reproducible with specific package, please provide link to package file.
5. Open issue at [GitHub](https://github.com/smira/aptly/issues)

### Suggesting Enhancements

1. Please search [issue tracker](https://github.com/smira/aptly/issues) for similar feature requests.
2. Describe why enhancement is important to you.
3. Include any additional details or implementation details.

### Improving Documentation

There are two kinds of documentation:

* [aptly website](https://www.aptly/info)
* aptly `man` page

Core content is mostly the same, but website contains more information, tutorials, examples.

If you want to update `man` page, please open PR to [main aptly repo](https://github.com/smira/aptly),
details in [man page](#man-page) section.

If you want to update website, please follow steps below:

1. Install [hugo](http://gohugo.io/)
2. Fork [website source](https://github.com/aptly-dev/aptly-dev.github.io) and clone it
3. Launch hugo in development mode: `hugo -w server`
4. Navigate to `http://localhost:1313/`: you should see aptly website
5. Update documentation, most of the time editing Markdown is all you need.
6. Page in browser should reload automatically as you make changes to source files.

We're always looking for new contributions to [FAQ](https://www.aptly.info/doc/faq/), [tutorials](https://www.aptly.info/tutorial/),
general fixes, clarifications, misspellings, grammar mistakes!

### Your Fist Code Contribution

Please follow [next section](#development-setup) on development process. When change is ready, please submit PR
following [PR template](.github/PULL_REQUEST_TEMPLATE.md).

Make sure that purpose of your change is clear, all the tests and checks pass, and all new code is covered with tests
if that is possible.

## Development Setup

This section describes local setup to start contributing to aptly source.

### Go & Python

You would need `Go` (latest version is recommended) and `Python` 2.7.x (3.x is not supported yet).

If you're new to Go, follow [getting started guide](https://golang.org/doc/install) to install it and perform
initial setup. With Go 1.8+, default `$GOPATH` is `$HOME/go`, so rest of this document assumes that.

Usually `$GOPATH/bin` is appended to your `$PATH` to make it easier to run built binaries, but you might choose
to prepend it or to skip this test if you're security conscious.  

### Forking and Cloning

As Go is using repository path in import paths, it's better to clone aptly repo (not your fork) at default location:

    mkdir -p ~/go/src/github.com/smira
    cd ~/go/src/github.com/smira
    git clone git@github.com:smira/aptly.git
    cd aptly

For main repo under your GitHub user and add it as another Git remote:

    git remote add <user> git@github.com:<user>/aptly.git

That way you can continue to build project as is (you don't need to adjust import paths), but you would need
to specify your remote name when pushing branches:

    git push <user> <your-branch>

### Dependencies

You would need some additional tools and Python virtual environment to run tests and checks, install them with:

    make prepare dev system/env

This is usually one-time action.

### Building

If you want to build aptly binary from your current source tree, run:

    make install

This would build `aptly` in `$GOPATH/bin`, so depending on your `$PATH`, you should be able to run it immediately with:

    aptly

Or, if it's not on your path:

    ~/go/bin/aptly

### Unit-tests

aptly has two kinds of tests: unit-tests and functional (system) tests. Functional tests are preferred way to test any
feature, but some features are much easier to test with unit-tests (e.g. algorithms, failure scenarios, ...)

aptly is using standard Go unit-test infrastructure plus [gocheck](http://labix.org/gocheck). Run the unit-tests with:

    make test

### Functional Tests

Functional tests are implemented in Python, and they use custom test runner which is similar to Python unit-test
runner. Most of the tests start with clean aptly state, run some aptly commands to prepare environment, and finally
run some aptly commands capturing output, exit code, checking any additional files being created and so on. API tests
are a bit different, as they re-use same aptly process serving API requests.

The easiest way to run functional tests is to use `make`:

    make system-test

This would check all the dependencies and run all the tests. Some tests (S3, Swift) require access credentials to
be set up in the environment. For example, it needs AWS credentials to run S3 tests (they would be used to publish to S3).
If credentials are missing, tests would be skipped.

You can also run subset of tests manually:

    system/run.py t04_mirror

This would run all the mirroring tests under `system/t04_mirror` folder.

Or you can run tests by test name mask:

    system/run.py UpdateMirror*

Or, you can run specific test by name:

    system/run.py UpdateMirror7Test

Test runner can update expected output instead of failing on mismatch (this is especially useful while
working on new tests):

    system/run.py --capture <test>

Output for some tests might contain environment-specific things, e.g. your home directory. In that case
you can use `${HOME}` and similar variable expansion in expected output files.

Some tests depend on fixtures, for example pre-populated GPG trusted keys. There are also test fixtures
captured after mirror update which contain pre-build aptly database and pool contents. They're useful if you
don't want to waste time in the test on populating aptly database while you need some packages to work with.
There are some packages available under `system/files/` directory which are used to build contents of local repos.

*WARNING*: tests are running under current `$HOME` directory with aptly default settings, so they clear completely
`~/.aptly.conf` and `~/.aptly` subdirectory between the runs. So it's not wise to have non-dev aptly being used with
this default location. You can run aptly under different user or by using non-default config location with non-default
aptly root directory.

### Style Checks

Style checks could be run with:

    make check

aptly is using [gometalinter](https://github.com/alecthomas/gometalinter) to run style checks on Go code. Configuration
for the linter could be found in [linter.json](linter.json) file. Running linters might take considerable amount of time
unfortunately, but usually warning reported by linters hint at real code issues.

Python code (system tests) are linted with [flake8 tool](https://pypi.python.org/pypi/flake8).

### Vendored Code

aptly is using Go vendoring for all the libraries aptly depends upon. `vendor/` directory is checked into the source
repository to avoid any problems if source repositories go away. Go build process will automatically prefer vendored
packages over packages in `$GOPATH`.

If you want to update vendored dependencies or to introduce new dependency, use [dep tool](https://github.com/golang/dep).
Usually all you need is `dep ensure` or `dep ensure -update`.

### man Page

aptly is using combination of [Go templates](http://godoc.org/text/template) and automatically generated text to build `aptly.1` man page. If either source
template [man/aptly.1.ronn.tmpl](man/aptly.1.ronn.tmpl) is changed or any command help is changed, run `make man` to regenerate
final rendered man page [man/aptly.1](man/aptly.1). In the end of the build, new man page is displayed for visual
verification.

Man page is built with small helper [_man/gen.go](man/gen.go) which pulls in template, command-line help from [cmd/](cmd/) folder
and runs that through [forked copy](https://github.com/smira/ronn) of [ronn](https://github.com/rtomayko/ronn).

### Bash Completion

Bash completion for aptly resides in the same repo under in [bash_completion.d/aptly](bash_completion.d/aptly). It's all hand-crafted.
When new option or command is introduced, bash completion should be updated to reflect that change.

When aptly package is being built, it automatically pulls bash completion and man page into the package.

## Design

This section requires future work.

*TBD*

### Database

### Package Pool

### Package

### PackageList, PackageRefList

### LocalRepo, RemoteRepo, Snapshot

### PublishedRepository

### Context

### Collections, CollectionFactory
