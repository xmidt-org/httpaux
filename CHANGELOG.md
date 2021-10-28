# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## [Unreleased]
- don't cancel contexts during retries to allow clients to read responses
- always stop retries when the enclosing context is canceled

## [v0.2.1]
- fixed the changelog syntax

## [v0.2.0]
- added erraux.Causer and the ability to customize cause in erraux.Encoder
- changed the signature of ErrorFielder.ErrorFields to avoid a dependency on erraux
- added erraux.Encoder.Body to dynamically disable/enable bodies for error rules
- erraux.Encoder now honors the optional error interfaces when no rules apply

## [v0.1.6]
- allow appending and extending Header while mainting immutability
- consistently defined middleware in subpackages
- ensure all error JSON representations are properly escaped
- configurable Encoder ruleset for HTTP error representations

## [v0.1.5]
- force a new release just to get github actions to run

## [v0.1.4]
- more consistent ErrorEncoder compared to gokit's default
- downgrade io.ReadAll in tests for pre-1.16 go environments

## [v0.1.3]
- httpmock.RoundTripper can now use a delegate in addition to an expected return
- http.Request.GetBody can now be nil for a retry.Client (https://github.com/xmidt-org/httpaux/issues/23)
- simplified retry.New and retry.NewClient (https://github.com/xmidt-org/httpaux/issues/24)

## [v0.1.2]
- allow http.Client to be decorated as with http.RoundTripper
- cleaned up middleware
- retry package allows linear and exponential backoff for HTTP clients

## [v0.1.1]
- Moved gate functionality to a subpackage
- Configurable gate control handler
- Client middleware
- Preserve CloseIdleConnections in decorated http.RoundTripper instances
- httpmock package now has convenient stretchr/testify/mock integrations
- observe is now a subpackage and exposes middleware
- normalized client and server middleware across the library
- busy functionality is now in its own package and consistently named

## [v0.1.0]
- Busy server middleware for controlling http.Handler concurrency
- Gate which allows shutoff of both client and server code
- Immutable Header
- Observable http.ResponseWriter
- Constant http.Handler which serves up configurable, static content
- Error type that exposes HTTP metadata for rendering a response
- Sonar integration

[Unreleased]: https://github.com/xmidt-org/httpaux/compare/v0.2.1..HEAD
[v0.2.0]: https://github.com/xmidt-org/httpaux/compare/v0.2.0...v0.2.1
[v0.2.0]: https://github.com/xmidt-org/httpaux/compare/v0.1.6...v0.2.0
[v0.1.6]: https://github.com/xmidt-org/httpaux/compare/v0.1.5...v0.1.6
[v0.1.5]: https://github.com/xmidt-org/httpaux/compare/v0.1.4...v0.1.5
[v0.1.4]: https://github.com/xmidt-org/httpaux/compare/v0.1.3...v0.1.4
[v0.1.3]: https://github.com/xmidt-org/httpaux/compare/v0.1.2...v0.1.3
[v0.1.2]: https://github.com/xmidt-org/httpaux/compare/v0.1.1...v0.1.2
[v0.1.1]: https://github.com/xmidt-org/httpaux/compare/v0.1.0...v0.1.1
[v0.1.0]: https://github.com/xmidt-org/httpaux/compare/v0.0.0...v0.1.0
