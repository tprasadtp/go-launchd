# go-launchd

[![Go Reference](https://pkg.go.dev/badge/github.com/tprasadtp/go-launchd.svg)](https://pkg.go.dev/github.com/tprasadtp/go-launchd)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/tprasadtp/go-launchd?label=go&logo=go&logoColor=white)
[![test](https://github.com/tprasadtp/go-launchd/actions/workflows/test.yml/badge.svg)](https://github.com/tprasadtp/go-launchd/actions/workflows/test.yml)
[![GitHub](https://img.shields.io/github/license/tprasadtp/go-launchd)](https://github.com/tprasadtp/go-launchd/blob/master/LICENSE)
[![GitHub tag (latest SemVer)](https://img.shields.io/github/v/tag/tprasadtp/go-launchd?color=7f50a6&label=release&logo=semver&sort=semver)](https://github.com/tprasadtp/go-launchd/releases)


## Socket Activation

- Supports [Launchd Socket Activation][] __without using [cgo][]__(`CGO_ENABLED=0`).
- This makes it simple to cross compile from your Linux/Windows CI machines.
- Both UDP and TCP sockets are supported.
- Supports both IPv4 and IPv6 and IPv4v6(single socket for both IPv4 and IPV6)

## How it works

This uses a similar technique as the [crypto/x509 package](https://go-review.googlesource.com/c/go/+/232397) via the `go:cgo_import_dynamic` directive.

As this implementation depends on linker directives which is not part of go spec,
and makes use of [syscall] and [unsafe] packages it may may break between go-releases and
between macOS versions.

## Usage

See [API docs](https://pkg.go.dev/github.com/tprasadtp/go-launchd) for more info and examples.


## Testing

Testing requires macOS and go toolchain 1.21 or later.

```
go test -v ./...
```

[syscall]: https://pkg.go.dev/syscall
[unsafe]: https://pkg.go.dev/unsafe
[cgo]: https://pkg.go.dev/cmd/cgo
[Launchd Socket Activation]: https://developer.apple.com/documentation/xpc1505523-launch_activate_socket
