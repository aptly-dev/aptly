[![Sourcegraph](https://sourcegraph.com/github.com/ugorji/go/-/badge.svg?v=4)](https://sourcegraph.com/github.com/ugorji/go/-/tree/codec?badge)
[![Build Status](https://travis-ci.org/ugorji/go.svg?branch=master)](https://travis-ci.org/ugorji/go)
[![codecov](https://codecov.io/gh/ugorji/go/branch/master/graph/badge.svg?v=4)](https://codecov.io/gh/ugorji/go)
[![GoDoc](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)](http://godoc.org/github.com/ugorji/go/codec)
[![rcard](https://goreportcard.com/badge/github.com/ugorji/go/codec?v=4)](https://goreportcard.com/report/github.com/ugorji/go/codec)
[![License](http://img.shields.io/badge/license-mit-blue.svg?style=flat-square)](https://raw.githubusercontent.com/ugorji/go/master/LICENSE)

# go-codec

This repository contains the `go-codec` library,
a High Performance and Feature-Rich Idiomatic encode/decode and rpc library for

  - msgpack: https://github.com/msgpack/msgpack
  - binc:    http://github.com/ugorji/binc
  - cbor:    http://cbor.io http://tools.ietf.org/html/rfc7049
  - json:    http://json.org http://tools.ietf.org/html/rfc7159 

For more information:

  - [see the codec/Readme for quick usage information](https://github.com/ugorji/go/tree/master/codec#readme)
  - [view the API on godoc](http://godoc.org/github.com/ugorji/go/codec)
  - [read the detailed usage/how-to primer](http://ugorji.net/blog/go-codec-primer)

Install using:

    go get github.com/ugorji/go/codec

# Import Path

The name of the github repo is being changed from `go` to `go-codec`.
This clarifies that this repo is only for the `go-codec`.

However, this package has always had a stable import path of "github.com/ugorji/go/codec".
We CANNOT change that without negatively affecting all our users.

Consequently, we enforce this import path by specifying it in the following files:

  - `go.mod`
  - `codec/0_importpath.go`

If you have a fork and want to change the package path, you will need to modify the above files.




