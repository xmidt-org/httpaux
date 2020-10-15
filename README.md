# httpaux

The missing functionality from `net/http`

[![Build Status](https://travis-ci.com/xmidt-org/httpaux.svg?branch=main)](https://travis-ci.com/xmidt-org/httpaux)
[![codecov.io](http://codecov.io/github/xmidt-org/httpaux/coverage.svg?branch=main)](http://codecov.io/github/xmidt-org/httpaux?branch=main)
[![Code Climate](https://codeclimate.com/github/xmidt-org/httpaux/badges/gpa.svg)](https://codeclimate.com/github/xmidt-org/httpaux)
[![Issue Count](https://codeclimate.com/github/xmidt-org/httpaux/badges/issue_count.svg)](https://codeclimate.com/github/xmidt-org/httpaux)
[![Go Report Card](https://goreportcard.com/badge/github.com/xmidt-org/httpaux)](https://goreportcard.com/report/github.com/xmidt-org/httpaux)
[![Apache V2 License](http://img.shields.io/badge/license-Apache%20V2-blue.svg)](https://github.com/xmidt-org/httpaux/blob/main/LICENSE)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=xmidt-org_httpaux&metric=alert_status)](https://sonarcloud.io/dashboard?id=xmidt-org_httpaux)
[![GitHub release](https://img.shields.io/github/release/xmidt-org/httpaux.svg)](CHANGELOG.md)
[![GoDoc](https://godoc.org/github.com/xmidt-org/httpaux?status.svg)](https://godoc.org/github.com/xmidt-org/httpaux)

## Summary

httpaux augments golang's `net/http` package with a few extra goodies:

- `RoundTripperFunc` that implements http.RoundTripper.  This is an analog to http.HandlerFunc for clients.
- `Busy` server middleware that constrains the number of concurrent requests by consulting a `Limiter` strategy
- `ObservableWriter` which decorates the typical `http.ResponseWriter`, providing visibility into what handlers have written to the response.  This type is intended to enable other middleware such as logging and metrics.
- `Gate` middleware for both servers and clients which can explicity shut off code.  One typical use case for this is putting an application into maintenance mode.
- `Header` immutable data structure that is much more efficient in situations where the same set of headers are iterated over repeatedly.  Useful when static headers need to be placed unconditionally on every request or response.
- `ConstantHandler` which returns a statically specified set of information in every response
- `Error` which can be used to wrap service and middleware errors in a form that makes rendering responses easier.  This type is compatible with frameworks like `go-kit`.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Install](#install)
- [Contributing](#contributing)

## Code of Conduct

This project and everyone participating in it are governed by the [XMiDT Code Of Conduct](https://xmidt.io/code_of_conduct/).
By participating, you agree to this Code.

## Install

go get github.com/xmidt-org/httpaux

## Contributing

Refer to [CONTRIBUTING.md](CONTRIBUTING.md).
