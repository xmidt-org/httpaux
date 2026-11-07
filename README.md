# httpaux

The missing functionality from `net/http`

[![Build Status](https://github.com/xmidt-org/httpaux/actions/workflows/ci.yml/badge.svg)](https://github.com/xmidt-org/httpaux/actions/workflows/ci.yml)
[![codecov.io](http://codecov.io/github/xmidt-org/httpaux/coverage.svg?branch=main)](http://codecov.io/github/xmidt-org/httpaux?branch=main)
[![Apache V2 License](http://img.shields.io/badge/license-Apache%20V2-blue.svg)](https://github.com/xmidt-org/httpaux/blob/main/LICENSE)
[![GitHub Release](https://img.shields.io/github/release/xmidt-org/httpaux.svg)](https://github.com/xmidt-org/httpaux/releases)
[![GoDoc](https://pkg.go.dev/badge/github.com/xmidt-org/httpaux)](https://pkg.go.dev/github.com/xmidt-org/httpaux)

## Summary

httpaux augments golang's `net/http` package with a few extra goodies.

- `client` package with middleware, header, and redirect helpers for `http.Client`
- `roundtrip` package with `http.RoundTripper` decoration, middleware, and header helpers
- `server` package with server-side middleware and header helpers
- `busy` package with middleware to limit server-side traffic
- `gate` package with middleware to control server and client traffic
- `observe` package with an observable `http.ResponseWriter` that lets `http.Handler` decorators examine what nested handlers wrote
- `recovery` package with middleware that recovers from handler panics
- `retry` package with configurable client retries, including exponential backoff
- `erraux` package with a configurable Encoder ruleset for representing errors as HTTP responses
- `httpmock` package for mock-style testing with HTTP clients
- An efficient, immutable `Header` type for writing static headers
- A configurable `ConstantHandler` that writes hardcoded information to responses

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Install](#install)
- [Contributing](#contributing)

## Code of Conduct

This project and everyone participating in it are governed by the [XMiDT Code Of Conduct](https://xmidt.io/code_of_conduct/).
By participating, you agree to this Code.

## Install

```sh
go get github.com/xmidt-org/httpaux
```

Requires Go 1.24 or later.

## Contributing

Refer to [CONTRIBUTING.md](CONTRIBUTING.md).
