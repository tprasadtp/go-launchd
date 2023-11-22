# go-launchd

[![Go Reference](https://pkg.go.dev/badge/github.com/tprasadtp/go-launchd.svg)](https://pkg.go.dev/github.com/tprasadtp/go-launchd)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/tprasadtp/go-launchd?label=go&logo=go&logoColor=white)
[![test](https://github.com/tprasadtp/go-launchd/actions/workflows/test.yml/badge.svg)](https://github.com/tprasadtp/go-launchd/actions/workflows/test.yml)
[![GitHub](https://img.shields.io/github/license/tprasadtp/go-launchd)](https://github.com/tprasadtp/go-launchd/blob/master/LICENSE)
[![GitHub tag (latest SemVer)](https://img.shields.io/github/v/tag/tprasadtp/go-launchd?color=7f50a6&label=release&logo=semver&sort=semver)](https://github.com/tprasadtp/go-launchd/releases)

## Socket Activation

- Supports [Launchd Socket Activation][socket-activation]
([`launch_activate_socket`][socket-activation]) without using [cgo].
- Simple to cross compile from Linux/Windows.
- Supports `tcp`, `unix`, `udp`,`unixgram` sockets.
- Supports `IPv4`, `IPv6` and `IPv4v6` sockets.

> **Note**
>
> As this module uses `go:cgo_import_dynamic` linker directive, which is not part
> of go spec as well as [syscall] and [unsafe] packages, it _may_ break between
> major go versions.

## Usage

See [API docs](https://pkg.go.dev/github.com/tprasadtp/go-launchd) for more info and examples.

## Testing

Testing requires macOS and go version 1.21 or later.

```
go test -v ./...
```

[syscall]: https://pkg.go.dev/syscall
[unsafe]: https://pkg.go.dev/unsafe
[cgo]: https://pkg.go.dev/cmd/cgo
[socket-activation]: https://developer.apple.com/documentation/xpc/1505523-launch_activate_socket
