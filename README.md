<!-- markdownlint-disable -->
<div align="center">

# go-remove

<img src="/.github/assets/logo.svg" alt="go-remove Logo" width="150">

A CLI tool to safely remove Go binaries with undo and history support

[![Latest Version](https://img.shields.io/github/tag/nicholas-fedor/go-remove.svg)](https://github.com/nicholas-fedor/go-remove/releases)
[![CircleCI](https://dl.circleci.com/status-badge/img/gh/nicholas-fedor/go-remove/tree/main.svg?style=shield)](https://dl.circleci.com/status-badge/redirect/gh/nicholas-fedor/go-remove/tree/main)
[![Codecov](https://codecov.io/gh/nicholas-fedor/go-remove/branch/main/graph/badge.svg)](https://codecov.io/gh/nicholas-fedor/go-remove)
[![GoDoc](https://godoc.org/github.com/nicholas-fedor/go-remove?status.svg)](https://godoc.org/github.com/nicholas-fedor/go-remove)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/nicholas-fedor/go-remove)
[![License](https://img.shields.io/badge/License-AGPL%20v3-blue.svg)](https://www.gnu.org/licenses/agpl-3.0)

</div>
<!-- markdownlint-restore -->

## Table of Contents

- [Features](#features)
- [Installation](#installation)
  - [Install script](#install-script)
  - [Linux packages](#linux-packages)
  - [Updating and uninstalling](#updating-and-uninstalling)
  - [Source](#source)
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

### Install script

```bash
tmp=$(mktemp)
curl -sSfL https://raw.githubusercontent.com/nicholas-fedor/go-remove/main/scripts/install.sh -o "$tmp" && sh "$tmp"
rm -f "$tmp"
```

On Linux, the script installs a native package (`.deb`, `.rpm`, `.apk`, or Arch) when one is attached to the release and a package manager plus root/sudo are available. Otherwise it extracts the release archive into `$HOME/go/bin`.

| Variable  | Meaning                                                  |
|-----------|----------------------------------------------------------|
| `VERSION` | Release tag (`v1.2.0` or `1.2.0`). Default: latest.      |
| `PREFIX`  | Directory for archive installs. Default: `$HOME/go/bin`. |
| `METHOD`  | `auto` (default), `package`, or `archive`.               |

```bash
# Pin a version
VERSION=v1.2.0 sh scripts/install.sh

# Always use the tarball into a custom directory
METHOD=archive PREFIX="$HOME/.local/bin" sh scripts/install.sh
```

Windows: download the `.zip` from the [releases page](https://github.com/nicholas-fedor/go-remove/releases).

### Linux packages

GitHub Releases include distro packages built by GoReleaser (nFPM):

| Format         | Distros                       |
|----------------|-------------------------------|
| `.deb`         | Debian, Ubuntu                |
| `.rpm`         | Fedora, RHEL, Rocky, openSUSE |
| `.apk`         | Alpine                        |
| `.pkg.tar.zst` | Arch, Manjaro                 |

Install a downloaded package with `dpkg -i`, `rpm -Uvh`, `apk add --allow-untrusted`, or `pacman -U`. Checksums are in `checksums.txt` on the same release.

### Updating and uninstalling

Re-run the script to replace the current install with the latest release (or `VERSION=…`). Native packages are upgraded in place (`dpkg`/`rpm`/`apk`/`pacman`); archive installs overwrite `$PREFIX/go-remove`.

```bash
tmp=$(mktemp)
curl -sSfL https://raw.githubusercontent.com/nicholas-fedor/go-remove/main/scripts/install.sh -o "$tmp"

# Update
sh "$tmp" update

# Uninstall (native package and/or the stored archive prefix)
sh "$tmp" uninstall

rm -f "$tmp"
```

### Source

```bash
go install github.com/nicholas-fedor/go-remove@latest
```

The binary is installed to `$GOPATH/bin` (typically `~/go/bin/go-remove`).

### Windows

Available architectures: `amd64`, `i386`, `arm64v8`

```powershell
# Download and extract (replace amd64 with your architecture)
Invoke-WebRequest -Uri "https://github.com/nicholas-fedor/go-remove/releases/latest/download/go-remove_windows_amd64_v1.0.0.zip" -OutFile "go-remove.zip"
Expand-Archive -Path "go-remove.zip" -DestinationPath "."

# Move to a directory in your PATH
Move-Item -Path ".\go-remove.exe" -Destination "$env:LOCALAPPDATA\Microsoft\WindowsApps\"
```

### Archive Naming

Release archives follow the pattern: `go-remove_{OS}_{ARCH}_{VERSION}.{ext}`

| OS      | Architecture | Archive Name Example                    |
|---------|--------------|-----------------------------------------|
| Linux   | amd64        | `go-remove_linux_amd64_v1.0.0.tar.gz`   |
| Linux   | arm64        | `go-remove_linux_arm64v8_v1.0.0.tar.gz` |
| macOS   | arm64        | `go-remove_macOS_arm64v8_v1.0.0.tar.gz` |
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

- Go 1.27.0 or later

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
