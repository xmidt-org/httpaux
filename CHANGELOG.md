# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## [Unreleased]
- allow http.Client to be decorated as with http.RoundTripper
- cleaned up middleware

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

[Unreleased]: https://github.com/xmidt-org/httpaux/compare/v0.1.1..HEAD
[v0.1.1]: https://github.com/xmidt-org/httpaux/compare/0.1.0...v0.1.1
[v0.1.0]: https://github.com/xmidt-org/httpaux/compare/0.0.0...v0.1.0
