# aptly-zsh
zsh completion for aptly

The zsh completion function and this README were imported from
[steinymity's repository](https://github.com/steinymity/aptly-zsh).

[Aptly](https://www.aptly.info/) is a great tool to setup Debian APT repositories
and mirrors. However, it's sometimes hard to remember all the command line
syntax and names of all options. Therefore I wrote this zsh completion modules
for aptly.

## License

This project is licensed under the terms of the MIT license. See file `LICENSE`
for details.

## Installation

Clone/copy the file `_aptly` to a place in your `$fpath` (show with
`echo $fpath`), or create a new directory and extend the fpath:

    mkdir -p ~/.zsh/functions
    fpath=(~/.zsh/functions $fpath)
    editor ~/.zsh/functions/_aptly

To profit most from the provided help messages and completions, make sure that
your zsh is setup properly. I have tested with the grml-zsh configuration that
is [available on Github](https://github.com/grml/grml-etc-core/) and on the
[grml homepage](http://grml.org/zsh/).

## Compatibility

The command line completion was developed based on the manpage of aptly 1.2.0
(currently in Debian Testing). However, most completions will work on older
versions (e.g., 0.9.7 in Debian Stable), too.

The completion function completes most arguments and options that can be passed
to aptly, including mirror/repository/snapshot/publish names. However, not all
arguments are handled yet. See the next section for known limitations.

## Known Bugs and Limitation

 * Boolean options are always completed with an explicit value `true` or
   `false`, although omitting the value is implicitly interpreted as `true`.
 * The source and destination names of copy and move operations must not be the
   same. This is currently not enforced.
 * The package query and display format strings are currently not completed.
 * Endpoints are not completed.
 * In `publish snapshot` there is no connection between the number of
   components passed to `-component` and the number of given snapshots to
   publish. Furthermore, the help text for `endpoint:prefix` disappears
   after its first possible location.
 * In `publish switch` the distribution can be set independently of the
   `endpoint:prefix` (i.e., all published distributions can be combined with
   all published `endpoint:prefix`).
 * Neither `publish switch` nor `publish update` check if publish was created
   from a snapshot or directly from a local repo.
 * Task commands are not completed. There are no commas added.
 * `help` won't complete after a sub command (i.e., `aptly mirror list help`).
 * Probably more, feel free to open issues, and submit patches/merge requests.

--------------------------------------------------------------------------------

Maximilian Stein <m@steiny.biz>
2018-02-03
