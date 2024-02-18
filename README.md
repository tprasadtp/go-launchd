<div align="center">

# 🚀 go-launchd

[![go-reference](https://img.shields.io/badge/godoc-reference-5272b4?logo=go&labelColor=3a3a3a&logoColor=959da5)](https://pkg.go.dev/github.com/tprasadtp/go-launchd)
[![go-version](https://img.shields.io/github/go-mod/go-version/tprasadtp/go-launchd?logo=go&labelColor=3a3a3a&logoColor=959da5&color=00add8&label=go)](https://github.com/tprasadtp/go-launchd/blob/master/go.mod)
[![test](https://github.com/tprasadtp/go-launchd/actions/workflows/test.yml/badge.svg)](https://github.com/tprasadtp/go-launchd/actions/workflows/test.yml)
[![lint](https://github.com/tprasadtp/go-launchd/actions/workflows/lint.yml/badge.svg)](https://github.com/tprasadtp/go-launchd/actions/workflows/lint.yml)
[![release](https://github.com/tprasadtp/go-launchd/actions/workflows/release.yml/badge.svg)](https://github.com/tprasadtp/go-launchd/actions/workflows/release.yml)
[![license](https://img.shields.io/github/license/tprasadtp/go-launchd?logo=github&labelColor=3a3a3a&logoColor=959da5)](https://github.com/tprasadtp/go-launchd/blob/master/LICENSE)
[![version](https://img.shields.io/github/v/tag/tprasadtp/go-launchd?label=version&sort=semver&logo=semver&labelColor=3a3a3a&logoColor=959da5&color=ce3262)](https://github.com/tprasadtp/go-launchd/releases)

</div>

## Socket Activation

- Supports [Launchd Socket Activation][socket-activation]
([`launch_activate_socket`][socket-activation]) _without using_ [cgo].
- Supports `tcp`, `unix`, `udp` and `unixgram` sockets.
- Supports `IPv4`, `IPv6` and `IPv4v6` sockets.

## Usage

See [API docs][godoc] for more info and examples.

## See Also

For systemd socket activation, Use [github.com/tprasadtp/go-systemd][go-systemd].

## Testing

Testing requires macOS and go version 1.21 or later.

```console
go test -cover ./...
```

[cgo]: https://pkg.go.dev/cmd/cgo
[socket-activation]: https://developer.apple.com/documentation/xpc/1505523-launch_activate_socket
[godoc]: https://pkg.go.dev/github.com/tprasadtp/go-launchd
[go-systemd]: https://github.com/tprasadtp/go-systemd
