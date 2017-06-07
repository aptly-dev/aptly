# go-xz

[![GoDoc](https://godoc.org/github.com/smira/go-xz?status.svg)](https://godoc.org/github.com/smira/go-xz)
[![Build Status](https://travis-ci.org/smira/go-xz.svg?branch=master)](https://travis-ci.org/smira/go-xz)

Simple .xz decompression using external program (xz --decompress)

## Why?

All go implementation of .xz decompression rely on `liblzma` dependency providing
in-process decompression.

This package uses external `xz` utility, so no depdendencies for the compiled binary.

## License

MIT