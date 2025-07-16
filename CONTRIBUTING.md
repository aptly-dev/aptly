# Contributing to aptly

:+1::tada: First off, thanks for taking the time to contribute! :tada::+1:

The following is a set of guidelines for contributing to [aptly](https://github.com/aptly-dev/aplty) and related repositories, which are hosted in the [aptly-dev Organization](https://github.com/aptly-dev) on GitHub.
These are just guidelines, not rules. Use your best judgment, and feel free to propose changes to this document in a pull request.

## What should I know before I get started?

### Code of Conduct

This project adheres to the Contributor Covenant [code of conduct](CODE_OF_CONDUCT.md).
By participating, you are expected to uphold this code.
Please report unacceptable behavior on [https://github.com/aptly-dev/aptly/discussions](https://github.com/aptly-dev/aptly/discussions)

### List of Repositories

* [aptly-dev/aptly](https://github.com/aptly-dev/aptly) - aptly source code, functional tests, man page
* [apty-dev/aptly-dev.github.io](https://github.com/aptly-dev/aptly-dev.github.io) - aptly website (https://www.aptly.info/)
* [aptly-dev/aptly-fixture-db](https://github.com/aptly-dev/aptly-fixture-db) & [aptly-dev/aptly-fixture-pool](https://github.com/aptly-dev/aptly-fixture-pool) provide
  fixtures for aptly functional tests

## How Can I Contribute?

### Reporting Bugs

1. Please search for similar bug report in [issue tracker](https://github.com/aptly-dev/aptly/issues)
2. Please verify that bug is not fixed in latest aptly nightly ([download information](https://www.aptly.info/download/))
3. Steps to reproduce increases chances for bug to be fixed quickly. If possible, submit PR with new functional test which fails.
4. If bug is reproducible with specific package, please provide link to package file.
5. Open issue at [GitHub](https://github.com/aptly-dev/aptly/issues)

### Suggesting Enhancements

1. Please search [issue tracker](https://github.com/aptly-dev/aptly/issues) for similar feature requests.
2. Describe why enhancement is important to you.
3. Include any additional details or implementation details.

### Improving Documentation

There are two kinds of documentation:

* [aptly website](https://www.aptly.info)
* aptly `man` page

Core content is mostly the same, but website contains more information, tutorials, examples.

If you want to update `man` page, please open PR to [main aptly repo](https://github.com/aptly-dev/aptly),
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

### Code Contribution

Please follow [next section](#development-setup) on development process. When change is ready, please submit PR
following [PR template](.github/PULL_REQUEST_TEMPLATE.md).

Make sure that purpose of your change is clear, all the tests and checks pass, and all new code is covered with tests
if that is possible.

### Get the Source

To clone the git repo, run the following commands:
```
git clone git@github.com:aptly-dev/aptly.git
cd aptly
```

## Development Setup

Working on aptly code can be done locally on the development machine, or for convenience by using docker. The next sections describe the setup process.

### Docker Development Setup

This section describes the docker setup to start contributing to aptly.

#### Dependencies

Install the following on your development machine:
- docker
- make
- git

##### Docker installation on macOS
1. Install [Docker Desktop on Mac](https://docs.docker.com/desktop/setup/install/mac-install/) (or via [Homebrew](https://brew.sh/))
2. Allow directory sharing
   - Open Docker Desktop
   - Go to `Settings → Resources → File Sharing → Virtual File Shares`
   - Add the aptly git repository path to the shared list (eg. /home/Users/john/aptly)

#### Create docker container

To build the development docker image, run:
```
make docker-image
```

#### Build aptly

To build the aptly in the development docker container, run:
```
make docker-build
```

#### Running aptly commands

To run aptly commands in the development docker container, run:
```
make docker-shell
```

Example:
```
$ make docker-shell
aptly@b43e8473ef81:/work/src$ aptly version
aptly version: 1.5.0+189+g0fc90dff
```

#### Running unit tests

In order to run aptly unit tests, enter the following:
```
make docker-unit-tests
```

#### Running system tests

In order to run aptly system tests, enter the following:
```
make docker-system-tests
```

#### Running golangci-lint

In order to run aptly unit tests, run:
```
make docker-lint
```

#### More info

Run `make help` for more information.


### Local Development Setup

This section describes local setup to start contributing to aptly.

#### Dependencies

Building aptly requires go version 1.22.

On Debian bookworm with backports enabled, go can be installed with:

    apt install -t bookworm-backports golang-go

#### Building

To build aptly, run:

    make build

Run aptly:

    build/aptly

To install aptly into `$GOPATH/bin`, run:

    make install

#### Unit-tests

aptly has two kinds of tests: unit-tests and functional (system) tests. Functional tests are preferred way to test any
feature, but some features are much easier to test with unit-tests (e.g. algorithms, failure scenarios, ...)

aptly is using standard Go unit-test infrastructure plus [gocheck](http://labix.org/gocheck). Run the unit-tests with:

    make test

#### Functional Tests

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

### man Page

aptly is using combination of [Go templates](http://godoc.org/text/template) and automatically generated text to build `aptly.1` man page. If either source
template [man/aptly.1.ronn.tmpl](man/aptly.1.ronn.tmpl) is changed or any command help is changed, run `make man` to regenerate
final rendered man page [man/aptly.1](man/aptly.1). In the end of the build, new man page is displayed for visual
verification.

Man page is built with small helper [\_man/gen.go](man/gen.go) which pulls in template, command-line help from [cmd/](cmd/) folder
and runs that through [forked copy](https://github.com/smira/ronn) of [ronn](https://github.com/rtomayko/ronn).

### Bash and Zsh Completion

Bash and Zsh completion for aptly reside in the same repo under in [completion.d/aptly](completion.d/aptly) and
[completion.d/\_aptly](completion.d/_aptly), respectively. It's all hand-crafted.
When new option or command is introduced, bash completion should be updated to reflect that change.

When aptly package is being built, it automatically pulls bash completion and man page into the package.
