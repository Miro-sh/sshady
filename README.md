<p align="center">
  <img src="assets/sshady-logo.png" alt="sshady logo" />
</p>

# sshady

Stop leaking your real IP every time you SSH into a server.

`sshady` generates SSH proxy configs and writes them straight to `~/.ssh/config`. One wizard, one alias, your traffic routes through SOCKS5 / HTTP / Tor / jump host — the target sees the proxy, not you.

Built for opsec-conscious sysadmins, pentesters, and anyone who cares where their packets come from.

[![CI](https://github.com/Miro-sh/sshady/actions/workflows/ci.yml/badge.svg)](https://github.com/Miro-sh/sshady/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

---

## Contents

- [Security](#security)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Commands](#commands)
- [Usage](#usage)
  - [Interactive wizard](#interactive-wizard)
  - [Non-interactive (flags)](#non-interactive-flags)
  - [Managing entries](#managing-entries)
  - [Shell completion](#shell-completion)
- [Proxy Types](#supported-proxy-types)
- [Create Flags](#create-flags)
- [Config Format](#what-gets-written-to-sshconfig)
- [Security Notes](#security-notes)
- [Development](#development)
- [License](#license)

---

## Security

> **This tool stores proxy credentials in plaintext inside `~/.ssh/config` (with `0600` permissions).**  
> Read [SECURITY.md](SECURITY.md) for best practices on credential handling.

**All inputs are validated** to prevent command injection and SSH config injection attacks.
- Hostnames validated via Go's `net.ParseIP` + strict regex
- Ports validated to range 1–65535
- Credentials validated against shell-safe character set
- Aliases validated against SSH config-safe character set

See [AUDIT.md](AUDIT.md) for the full security audit report.

---

## Prerequisites

- Go 1.21+ (to build from source)
- `ncat` (from the `nmap` package) on the machine you SSH from

```bash
# Debian / Ubuntu
sudo apt install ncat

# Arch Linux
sudo pacman -S nmap

# macOS
brew install nmap
```

---

## Installation

### From source

```bash
git clone https://github.com/Miro-sh/sshady.git
cd sshady
make build
sudo make install
```

### Using `go install`

```bash
go install github.com/Miro-sh/sshady@latest
```

---

## Quick Start

```bash
# Launch the interactive wizard
sshady

# Or use flags for scripting
sshady create \
  --alias myserver \
  --host 1.2.3.4 \
  --user admin \
  --proxy-type socks5 \
  --proxy-host proxy.example.com \
  --proxy-port 1080

# Check it works
sshady validate myserver --test-proxy

# Connect!
ssh myserver
```

---

## Commands

| Command | Description |
|---------|-------------|
| `sshady` | Launch the interactive wizard |
| `sshady create` | Create a new proxy configuration (wizard or flags) |
| `sshady list` | List all entries managed by sshady |
| `sshady show <alias>` | Show details of a managed entry |
| `sshady validate <alias>` | Validate config syntax and proxy reachability |
| `sshady test <alias>` | Test proxy TCP reachability |
| `sshady delete <alias>` | Remove a managed entry |
| `sshady completion <shell>` | Generate shell completion script |
| `sshady --version` | Show version |
| `sshady --help` | Show help |

### Global Flags

| Flag | Description |
|------|-------------|
| `--dry-run` | Preview changes without modifying the config file |
| `--force` | Overwrite existing sshady-managed entries without confirmation |
| `--config <path>` | Use alternative SSH config path (default: `~/.ssh/config`) |
| `--quiet`, `-q` | Suppress non-error output |
| `--verbose`, `-v` | Enable verbose diagnostic output |

---

## Usage

### Interactive wizard

```bash
sshady
```

![sshady wizard](assets/sshady-1.png)

### Non-interactive (flags)

```bash
# SOCKS5 without auth
sshady create \
  --alias myserver \
  --host 1.2.3.4 \
  --user admin \
  --proxy-type socks5 \
  --proxy-host proxy.example.com \
  --proxy-port 1080

# SOCKS5 with auth via env var (most secure)
SSHADY_PROXY_PASS=s3cr3t sshady create \
  --alias myserver \
  --host 1.2.3.4 \
  --proxy-type socks5 \
  --proxy-host proxy.example.com \
  --proxy-user alice

# SOCKS5 with auth via file (recommended)
sshady create \
  --alias myserver \
  --host 1.2.3.4 \
  --proxy-type socks5 \
  --proxy-host proxy.example.com \
  --proxy-user alice \
  --proxy-pass-file ~/.ssh/proxy-pass

# HTTP CONNECT proxy
sshady create \
  --alias web \
  --host 10.0.0.5 \
  --proxy-type http \
  --proxy-host proxy.example.com \
  --proxy-port 8080

# Tor
sshady create --alias hidden --host example.onion --proxy-type tor

# SSH jump host
sshady create \
  --alias internal \
  --host 10.0.0.5 \
  --proxy-type jump \
  --jump-host bastion@192.168.1.1
```

### Managing entries

```bash
# List all
sshady list

# Show details
sshady show myserver

# Validate config syntax + proxy
sshady validate myserver --test-proxy
sshady validate --all

# Test proxy
sshady test myserver --timeout 10

# Delete
sshady delete myserver

# Preview with dry-run
sshady create --alias test --host 1.2.3.4 --proxy-type tor --dry-run

# Scripting with --quiet
sshady create -q --alias srv --host 1.2.3.4 --proxy-type tor

# Verbose diagnostics
sshady test myserver -v
```

### Shell completion

```bash
# Bash
source <(sshady completion bash)

# Zsh
source <(sshady completion zsh)

# Fish
sshady completion fish | source

# PowerShell
sshady completion powershell | Out-String | Invoke-Expression
```

---

## Supported proxy types

| Type | Description | Requirement |
|------|-------------|-------------|
| `socks5` | SOCKS5 proxy, optional user/pass auth | `ncat` |
| `http` | HTTP CONNECT proxy, optional user/pass auth | `ncat` |
| `tor` | Routes through Tor, auto-fills `127.0.0.1:9050` | `ncat` + Tor running |
| `jump` | SSH jump host / bastion server | SSH only, no `ncat` needed |

---

## Create flags

| Flag | Description | Default |
|------|-------------|---------|
| `--alias` | SSH host alias | required |
| `--host` | Target hostname or IP | required |
| `--user` | SSH user | current user |
| `--port` | SSH port | `22` |
| `--identity-file` | Path to SSH private key | none |
| `--proxy-type` | `socks5`, `http`, `tor`, or `jump` | required |
| `--proxy-host` | Proxy hostname or IP | required for socks5/http |
| `--proxy-port` | Proxy port | `1080` |
| `--proxy-user` | Proxy username | none |
| `--proxy-pass` | Proxy password (⚠️ visible in `ps aux`) | none |
| `--proxy-pass-file` | Read proxy password from file (**recommended**) | none |
| `--jump-host` | Jump host (`user@host`) | required for jump |

---

## What gets written to ~/.ssh/config

Each entry is wrapped in marker comments so sshady can track what it manages:

```
# BEGIN SSHADY:myserver
# SSHADY META: type=socks5 addr=proxy.example.com:1080 auth=no
Host myserver
    HostName 1.2.3.4
    User admin
    Port 22
    ProxyCommand ncat --proxy-type socks5 --proxy proxy.example.com:1080 %h %p
# END SSHADY:myserver
```

Then connect normally:

```bash
ssh myserver
```

---

## Security notes

- **Backup rotation**: Up to 5 timestamped backups are kept automatically (`config.sshady.YYYYMMDD-HHMMSS.bak`).
- **Atomic writes**: All writes use temp-file + fsync + rename — no partial writes possible.
- **Permissions**: `~/.ssh/config` is always written with `0600`, `~/.ssh/` with `0700`.
- **Input validation**: Every user-supplied value is validated against strict patterns.
- **Credentials**: Prefer `--proxy-pass-file` or `SSHADY_PROXY_PASS` over `--proxy-pass`.
- **Runtime checks**: `sshady validate` runs `ssh -G` to verify config syntax. `sshady test` verifies proxy reachability.

Read the full [Security Policy](SECURITY.md) and [Audit Report](AUDIT.md).

---

## Development

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup and guidelines.

```bash
make build       # Build binary
make test        # Run tests with race detector
make test-short  # Quick tests only
make lint        # Run golangci-lint
make cover       # Generate coverage report
make dist        # Cross-compile for linux/darwin/windows
```

### Architecture

```
sshady/
├── main.go                   # Entry point, ldflags version injection
├── cmd/                      # CLI commands (cobra)
│   ├── root.go               # Global flags, config override
│   ├── create.go             # Wizard + non-interactive creation
│   ├── list.go, show.go      # Read operations
│   ├── validate.go           # ssh -G syntax check + proxy test
│   ├── test.go               # ncat TCP reachability check
│   ├── delete.go             # Remove with backup rotation
│   └── completion.go         # Shell completion generation
├── internal/
│   ├── proxy/                # Proxy types, validation, command generation
│   ├── sshconf/              # SSH config I/O, atomic writes, backup rotation
│   └── check/                # Runtime prerequisite checks (ncat, Tor)
└── .github/workflows/        # CI/CD (lint, test, security scan, release)
```

---

## License

MIT — see [LICENSE](LICENSE).
