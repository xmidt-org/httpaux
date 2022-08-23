# httpaux

The missing functionality from `net/http`

[![Build Status](https://github.com/xmidt-org/httpaux/actions/workflows/ci.yml/badge.svg)](https://github.com/xmidt-org/httpaux/actions/workflows/ci.yml)
[![codecov.io](http://codecov.io/github/xmidt-org/httpaux/coverage.svg?branch=main)](http://codecov.io/github/xmidt-org/httpaux?branch=main)
[![Go Report Card](https://goreportcard.com/badge/github.com/xmidt-org/httpaux)](https://goreportcard.com/report/github.com/xmidt-org/httpaux)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=xmidt-org_httpaux&metric=alert_status)](https://sonarcloud.io/dashboard?id=xmidt-org_httpaux)
[![Apache V2 License](http://img.shields.io/badge/license-Apache%20V2-blue.svg)](https://github.com/xmidt-org/httpaux/blob/main/LICENSE)
[![GitHub Release](https://img.shields.io/github/release/xmidt-org/httpaux.svg)](CHANGELOG.md)
[![GoDoc](https://pkg.go.dev/badge/github.com/xmidt-org/httpaux)](https://pkg.go.dev/github.com/xmidt-org/httpaux)

## Summary

httpaux augments golang's `net/http` package with a few extra goodies.

- Middleware for clients in the form of http.RoundTripper decoration
- Standardized middleware interfaces
- Busy server middleware to limit serverside traffic
- Observable http.ResponseWriter middleware that provides a standard way for http.Handler decorators to examine what was written to the response by nested handlers
- Gate middleware to control server and client traffic
- An efficient, immutable Header type for writing static headers
- A configurable `ConstantHandler` that writes hardcoded information to responses
- `httpmock` package for mock-style testing with HTTP clients
- `retry` package with configurable retry for clients including exponential backoff
- `erraux` package with a configurable Encoder ruleset for representing errors as HTTP responses

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
