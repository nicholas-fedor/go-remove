<!-- markdownlint-disable -->
<div align="center">

# go-remove

<img src="/.github/assets/logo.svg" alt="go-remove Logo" width="150">

A CLI tool to safely remove Go binaries with undo and history support

[![Latest Version](https://img.shields.io/github/tag/nicholas-fedor/go-remove.svg)](https://github.com/nicholas-fedor/go-remove/releases)
[![CircleCI](https://dl.circleci.com/status-badge/img/gh/nicholas-fedor/go-remove/tree/main.svg?style=shield)](https://dl.circleci.com/status-badge/redirect/gh/nicholas-fedor/go-remove/tree/main)
[![Codecov](https://codecov.io/gh/nicholas-fedor/go-remove/branch/main/graph/badge.svg)](https://codecov.io/gh/nicholas-fedor/go-remove)
[![GoDoc](https://godoc.org/github.com/nicholas-fedor/go-remove?status.svg)](https://godoc.org/github.com/nicholas-fedor/go-remove)
[![Go Report Card](https://goreportcard.com/badge/github.com/nicholas-fedor/go-remove)](https://goreportcard.com/report/github.com/nicholas-fedor/go-remove)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/nicholas-fedor/go-remove)
[![License](https://img.shields.io/badge/License-AGPL%20v3-blue.svg)](https://www.gnu.org/licenses/agpl-3.0)

</div>
<!-- markdownlint-restore -->

## Table of Contents

- [Features](#features)
- [Installation](#installation)
  - [From Source](#from-source)
  - [From Releases](#from-releases)
    - [Linux](#linux)
    - [Windows](#windows)
    - [Archive Naming](#archive-naming)
- [Quick Start](#quick-start)
- [Usage](#usage)
  - [Direct Removal](#direct-removal)
  - [Interactive TUI](#interactive-tui)
  - [Undo Deletion](#undo-deletion)
  - [Restore from History](#restore-from-history)
- [Command Reference](#command-reference)
- [Filesystem Locations](#filesystem-locations)
  - [Data Storage](#data-storage)
  - [Trash Locations](#trash-locations)
  - [Binary Directories (in precedence order)](#binary-directories-in-precedence-order)
- [Building from Source](#building-from-source)
- [Requirements](#requirements)
- [Contributing](#contributing)
- [License](#license)

## Features

- **Interactive TUI**: Browse and select binaries to remove from a grid interface
- **Undo Support**: Restore the most recently deleted binary with a single command
- **Deletion History**: Browse and restore any previously removed binary
- **Safe Removal**: Moves binaries to system trash instead of permanent deletion
- **Cross-Platform**: Works on Linux (XDG-compliant) and Windows
- **Verbose Logging**: Optional detailed output for debugging

## Installation

### From Source

```bash
go install github.com/nicholas-fedor/go-remove@latest
```

The binary will be installed to `$GOPATH/bin` (typically `~/go/bin/go-remove`).

### From Releases

Download the latest release for your platform from the [releases page](https://github.com/nicholas-fedor/go-remove/releases).

#### Linux

Available architectures: `amd64`, `i386`, `armhf`, `arm64v8`

```bash
# Download and extract (replace amd64 with your architecture)
curl -LO https://github.com/nicholas-fedor/go-remove/releases/latest/download/go-remove_linux_amd64_latest.tar.gz
tar -xzf go-remove_linux_amd64_latest.tar.gz
chmod +x go-remove
sudo mv go-remove /usr/local/bin/

# Verify checksum (optional)
curl -LO https://github.com/nicholas-fedor/go-remove/releases/latest/download/checksums.txt
sha256sum -c checksums.txt --ignore-missing
```

#### Windows

Available architectures: `amd64`, `i386`, `arm64v8`

```powershell
# Download and extract (replace amd64 with your architecture)
Invoke-WebRequest -Uri "https://github.com/nicholas-fedor/go-remove/releases/latest/download/go-remove_windows_amd64_latest.zip" -OutFile "go-remove.zip"
Expand-Archive -Path "go-remove.zip" -DestinationPath "."

# Move to a directory in your PATH (requires Administrator privileges)
Move-Item -Path ".\go-remove.exe" -Destination "$env:LOCALAPPDATA\Microsoft\WindowsApps\"
```

#### Archive Naming

Release archives follow the pattern: `go-remove_{OS}_{ARCH}_{VERSION}.{ext}`

| OS      | Architecture | Archive Name Example                    |
|---------|--------------|-----------------------------------------|
| Linux   | amd64        | `go-remove_linux_amd64_v1.0.0.tar.gz`   |
| Linux   | arm64        | `go-remove_linux_arm64v8_v1.0.0.tar.gz` |
| Windows | amd64        | `go-remove_windows_amd64_v1.0.0.zip`    |

## Quick Start

```bash
# Launch interactive TUI to select binaries
go-remove

# Remove a specific binary
go-remove vhs

# Undo the last deletion
go-remove --undo
```

## Usage

### Direct Removal

Remove a specific binary by name:

```bash
go-remove vhs
```

With verbose output:

```bash
go-remove -v vhs
```

Remove from `GOROOT/bin` instead of `GOBIN`/`GOPATH/bin`:

```bash
go-remove --goroot vhs
```

### Interactive TUI

Launch without arguments to use the interactive TUI:

```bash
go-remove
```

**TUI Controls:**

| Key                                | Action                                   |
|------------------------------------|------------------------------------------|
| `↑`/`↓`/`←`/`→` or `k`/`j`/`h`/`l` | Navigate grid                            |
| `Enter`                            | Remove selected binary                   |
| `s`                                | Toggle sort order (ascending/descending) |
| `r`                                | Open deletion history                    |
| `q` or `Ctrl+C`                    | Quit                                     |

### Undo Deletion

Restore the most recently deleted binary:

```bash
go-remove --undo
# or
go-remove -u
```

### Restore from History

Browse and restore from deletion history:

```bash
go-remove --restore
# or
go-remove -r
```

**History View Controls:**

| Key     | Action                                       |
|---------|----------------------------------------------|
| `↑`/`↓` | Navigate history                             |
| `Enter` | Restore selected binary to original location |
| `d`     | Permanently delete from trash                |
| `u`     | Undo most recent deletion                    |
| `q`     | Return to main view                          |

## Command Reference

| Flag          | Short | Description                                         |
|---------------|-------|-----------------------------------------------------|
| `--undo`      | `-u`  | Restore the most recently deleted binary            |
| `--restore`   | `-r`  | Open the deletion history view                      |
| `--goroot`    |       | Target `GOROOT/bin` instead of `GOBIN`/`GOPATH/bin` |
| `--log-level` |       | Set log level (`debug`, `info`, `warn`, `error`)    |
| `--version`   | `-v`  | Show version information                            |
| `--help`      | `-h`  | Show help message                                   |

## Filesystem Locations

### Data Storage

Deletion history is stored in a Badger KV database:

**Linux:**

- `$XDG_DATA_HOME/go-remove/history.badger`
- Fallback: `~/.local/share/go-remove/history.badger`

**Windows:**

- `%LOCALAPPDATA%\go-remove\history.badger`
- Fallback: `%USERPROFILE%\go-remove\history.badger`

### Trash Locations

**Linux:** XDG-compliant trash at `$XDG_DATA_HOME/Trash` (fallback: `~/.local/share/Trash`)

- `files/` - Trashed binaries
- `info/` - Metadata files (`.trashinfo`)

**Windows:** Windows Recycle Bin via Shell API

### Binary Directories (in precedence order)

1. `GOROOT/bin` (when using `--goroot` flag)
2. `GOBIN` (environment variable)
3. `GOPATH/bin` (from `GOPATH` environment variable)
4. Default fallback: `~/go/bin` (Linux/macOS) or `%USERPROFILE%\go\bin` (Windows)

## Building from Source

```bash
git clone https://github.com/nicholas-fedor/go-remove.git
cd go-remove
go build -o go-remove .
```

Run locally:

```bash
./go-remove --help
```

## Requirements

- Go 1.26 or later

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for detailed guidelines on:

- Setting up your development environment
- Code standards and testing requirements
- Submitting pull requests
- Commit signing requirements
- AI policy

You can also submit issues or pull requests on [GitHub](https://github.com/nicholas-fedor/go-remove).

## License

This project is licensed under the [GNU Affero General Public License v3](LICENSE.md).

---

**Logo Credits:** Special thanks to [Maria Letta](https://github.com/MariaLetta) for the awesome [Free Gophers Pack](https://github.com/MariaLetta/free-gophers-pack).
