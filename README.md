<div align="center">

# go-remove

<img src="/.github/assets/logo.svg" alt="go-remove Logo" width="150">

A command-line tool to remove Go binaries with ease

[![Latest Version](https://img.shields.io/github/tag/nicholas-fedor/go-remove.svg)](https://github.com/nicholas-fedor/go-remove/releases)
[![CircleCI](https://dl.circleci.com/status-badge/img/gh/nicholas-fedor/go-remove/tree/main.svg?style=shield)](https://dl.circleci.com/status-badge/redirect/gh/nicholas-fedor/go-remove/tree/main)
[![Codecov](https://codecov.io/gh/nicholas-fedor/go-remove/branch/main/graph/badge.svg)](https://codecov.io/gh/nicholas-fedor/go-remove)
[![Codacy Badge](https://app.codacy.com/project/badge/Grade/1c48cfb7646d4009aa8c6f71287670b8)](https://www.codacy.com/gh/nicholas-fedor/go-remove/dashboard?utm_source=github.com&amp;utm_medium=referral&amp;utm_content=nicholas-fedor/go-remove&amp;utm_campaign=Badge_Grade)
[![GoDoc](https://godoc.org/github.com/nicholas-fedor/go-remove?status.svg)](https://godoc.org/github.com/nicholas-fedor/go-remove)
[![Go Report Card](https://goreportcard.com/badge/github.com/nicholas-fedor/go-remove)](https://goreportcard.com/report/github.com/nicholas-fedor/go-remove)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/nicholas-fedor/go-remove)
[![License](https://img.shields.io/badge/License-AGPL%20v3-blue.svg)](https://www.gnu.org/licenses/agpl-3.0)

</div>

`go-remove` is a command-line tool for removing Go binaries from standard binary directories (`GOROOT/bin`, `GOBIN`, or `GOPATH/bin`).
It supports both direct removal of a specified binary and an interactive TUI (Terminal User Interface) mode for selecting binaries to delete.

## Features

- **Direct Removal**: Remove a specific binary by name.
- **Interactive TUI**: Browse and select binaries to remove from a grid interface.
- **Verbose Logging**: Optional detailed output for debugging or confirmation.
- **Platform Support**: Handles Windows `.exe` extensions and cross-platform paths.

## Installation

Install `go-remove` using Go:

```bash
go install github.com/nicholas-fedor/go-remove@latest
```

This places the go-remove binary in your $GOPATH/bin (e.g., ~/go/bin/).

## Usage

### Direct Removal

Remove a specific binary:

```bash
go-remove vhs
```

With verbose output:

```bash
go-remove -v vhs
```

Target `GOROOT/bin` instead of `GOBIN` or `GOPATH/bin`:

```bash
go-remove --goroot vhs
```

### TUI Mode

Launch the interactive TUI to select binaries:

```bash
go-remove
```

- Use arrow keys (`↑`/`↓`/`←`/`→`) or `k`/`j`/`h`/`l` to navigate.
- Press `Enter` to remove the selected binary.
- Press `q` or `Ctrl+C` to quit.

### Help

View available options:

```bash
go-remove -h
```

## Building from Source

Clone the repository and build:

```bash
git clone https://github.com/nicholas-fedor/go-remove.git
cd go-remove
go build -o go-remove ./cmd
```

Run locally:

```bash
./go-remove
```

## Requirements

- Go 1.16 or later (for module support).

## License

This project is licensed under the GNU Affero General Public License v3 — see the [LICENSE](LICENSE.md) file for details.

## Contributing

Contributions are welcome! Please submit issues or pull requests on GitHub.

## Logo

Special thanks to [Maria Letta](https://github.com/MariaLetta) for providing an awesome [collection](https://github.com/MariaLetta/free-gophers-pack) of Go gophers.
