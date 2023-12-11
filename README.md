# 🚀 go-launchd

[![Go Reference](https://img.shields.io/badge/go-reference-00758D?logo=go&logoColor=white)](https://pkg.go.dev/github.com/tprasadtp/go-launchd)
[![go.mod Go version](https://img.shields.io/github/go-mod/go-version/tprasadtp/go-launchd?label=go-version&logo=go&logoColor=white&color=CE3262)](https://github.com/tprasadtp/go-launchd/blob/master/go.mod)
[![test](https://github.com/tprasadtp/go-launchd/actions/workflows/test.yml/badge.svg)](https://github.com/tprasadtp/go-launchd/actions/workflows/test.yml)
[![lint](https://github.com/tprasadtp/go-launchd/actions/workflows/lint.yml/badge.svg)](https://github.com/tprasadtp/go-launchd/actions/workflows/lint.yml)
[![GitHub](https://img.shields.io/github/license/tprasadtp/go-launchd)](https://github.com/tprasadtp/go-launchd/blob/master/LICENSE)
[![GitHub tag (latest SemVer)](https://img.shields.io/github/v/tag/tprasadtp/go-launchd?color=7f50a6&label=release&logo=semver&sort=semver)](https://github.com/tprasadtp/go-launchd/releases)

## Socket Activation

- Supports [Launchd Socket Activation][socket-activation]
([`launch_activate_socket`][socket-activation]) _without using_ [cgo].
- Supports `tcp`, `unix`, `udp`,`unixgram` sockets.
- Supports `IPv4`, `IPv6` and `IPv4v6` sockets.


## Usage

See [API docs](https://pkg.go.dev/github.com/tprasadtp/go-launchd) for more info and examples.

## See Also

This package only supports launchd socket activation. For systemd socket activation,
Use [github.com/tprasadtp/go-systemd](https://github.com/tprasadtp/go-systemd).

## Testing

Testing requires macOS and go version 1.21 or later.

- Create `.gocover` directory to gather coverage data

    ```bash
    mkdir .gocover
    ```

- Run Tests

    ```console
    go test -cover --test.gocoverdir .gocover ./...
    ```

[syscall]: https://pkg.go.dev/syscall
[unsafe]: https://pkg.go.dev/unsafe
[cgo]: https://pkg.go.dev/cmd/cgo
[socket-activation]: https://developer.apple.com/documentation/xpc/1505523-launch_activate_socket
