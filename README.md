# go-launchd

[![Go Reference](https://pkg.go.dev/badge/github.com/tprasadtp/go-launchd.svg)](https://pkg.go.dev/github.com/tprasadtp/go-launchd)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/tprasadtp/go-launchd?label=go&logo=go&logoColor=white)
[![test](https://github.com/tprasadtp/go-launchd/actions/workflows/test.yml/badge.svg)](https://github.com/tprasadtp/go-launchd/actions/workflows/test.yml)
[![GitHub](https://img.shields.io/github/license/tprasadtp/go-launchd)](https://github.com/tprasadtp/go-launchd/blob/master/LICENSE)
[![GitHub tag (latest SemVer)](https://img.shields.io/github/v/tag/tprasadtp/go-launchd?color=7f50a6&label=release&logo=semver&sort=semver)](https://github.com/tprasadtp/go-launchd/releases)

> **Warning**
>
> This module makes use of syscall and unsafe packages thus may break between go-releases
> and between macOS versions.

## Socket Activation

Supports [Launchd Socket Activation](https://developer.apple.com/documentation/xpc/1505523-launch_activate_socket) _without_ using cgo(`CGO_ENABLED=0`), by making calls to
libc directly. This makes it simple to cross compile from your Linux/Windows CI machines.

See [API docs](https://pkg.go.dev/github.com/tprasadtp/go-launchd) for more info and examples.
