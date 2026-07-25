<p align="center">
  <img src="assets/sshady-logo.png" alt="sshady logo" />
</p>

<h1 align="center">sshady</h1>

<p align="center">
  <strong>Stop leaking your real IP every time you SSH into a server.</strong>
</p>

<p align="center">
  <a href="https://go.dev"><img src="https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go&logoColor=white" alt="Go version" /></a>
  <img src="https://img.shields.io/badge/platform-Linux%20%7C%20macOS-lightgrey" alt="Platform" />
</p>

---

`sshady` generates SSH proxy configurations and writes them straight to `~/.ssh/config`. One wizard, one alias — your traffic routes through SOCKS5, HTTP, Tor, or a jump host. The target sees the proxy, not you.

Built for opsec-conscious sysadmins, pentesters, and anyone who cares where their packets come from.

## Features

- **Interactive wizard** — run `sshady`, answer the prompts, done.
- **Non-interactive mode** — full flag support for scripting and automation.
- **Four proxy types** — SOCKS5, HTTP CONNECT, Tor, and SSH jump hosts.
- **Config management** — every entry is tracked, listed with `sshady list`, and never touches your existing `~/.ssh/config` entries.
- **Safe by default** — automatic backups, atomic writes, strict `0600` permissions.

## Prerequisites

- Go 1.26+ (to build from source)
- `ncat` (from the `nmap` package) on the machine you SSH from

```bash
# Debian / Ubuntu
sudo apt install ncat

# Arch Linux
sudo pacman -S nmap

# macOS
brew install nmap
```

## Installation

```bash
git clone https://github.com/Miro-sh/sshady
cd sshady
go build -o sshady .
sudo mv sshady /usr/local/bin/
```

## Usage

### Interactive wizard

Run `sshady` with no arguments and answer the prompts:

```bash
sshady
```

![sshady wizard](assets/sshady-1.png)

### Non-interactive mode

Provide `--alias`, `--host`, and `--proxy-type` to skip the wizard entirely:

```bash
# SOCKS5 without auth
sshady create \
  --alias myserver \
  --host 1.2.3.4 \
  --user admin \
  --proxy-type socks5 \
  --proxy-host proxy.example.com \
  --proxy-port 1080

# SOCKS5 with auth
sshady create \
  --alias myserver \
  --host 1.2.3.4 \
  --proxy-type socks5 \
  --proxy-host proxy.example.com \
  --proxy-port 1080 \
  --proxy-user alice \
  --proxy-pass s3cr3t

# HTTP CONNECT proxy
sshady create \
  --alias myserver \
  --host 1.2.3.4 \
  --proxy-type http \
  --proxy-host proxy.example.com \
  --proxy-port 8080

# Tor (no proxy address needed, auto-fills 127.0.0.1:9050)
sshady create --alias hidden --host 1.2.3.4 --proxy-type tor

# SSH jump host / bastion
sshady create \
  --alias internal \
  --host 10.0.0.5 \
  --proxy-type jump \
  --jump-host bastion@192.168.1.1
```

## Supported proxy types

| Type     | Description                                   | Requirement                |
|----------|-----------------------------------------------|----------------------------|
| `socks5` | SOCKS5 proxy, optional user/pass auth         | `ncat`                     |
| `http`   | HTTP CONNECT proxy, optional user/pass auth   | `ncat`                     |
| `tor`    | Routes through Tor, auto-fills `127.0.0.1:9050` | `ncat` + Tor running     |
| `jump`   | SSH jump host / bastion server                | SSH only, no `ncat` needed |

## What gets written to ~/.ssh/config

Each entry is wrapped in marker comments so `sshady list` can track what it manages:

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

## Commands

| Command               | Description                              |
|-----------------------|------------------------------------------|
| `sshady`              | Launch the interactive wizard            |
| `sshady create`       | Same as above (explicit subcommand)      |
| `sshady create [flags]` | Non-interactive mode, see flags below  |
| `sshady list`         | List all entries managed by sshady       |
| `sshady --help`       | Show help                                |

### create flags

| Flag             | Description                             | Default                  |
|------------------|-----------------------------------------|--------------------------|
| `--alias`        | SSH host alias                          | required                 |
| `--host`         | Target hostname or IP                   | required                 |
| `--user`         | SSH user                                | current user             |
| `--port`         | SSH port                                | `22`                     |
| `--identity-file`| Path to SSH private key                 | none                     |
| `--proxy-type`   | `socks5`, `http`, `tor`, or `jump`      | required                 |
| `--proxy-host`   | Proxy hostname or IP                    | required for socks5/http |
| `--proxy-port`   | Proxy port                              | `1080`                   |
| `--proxy-user`   | Proxy username                          | none                     |
| `--proxy-pass`   | Proxy password                          | none                     |
| `--jump-host`    | Jump host (`user@host`)                 | required for jump        |

## Security notes

- A backup is created at `~/.ssh/config.sshady.bak` before every write.
- All writes are atomic (temp file + rename, no partial writes).
- `~/.ssh/config` is always written with `0600` permissions.
- If you use proxy authentication, credentials are stored in plaintext inside `~/.ssh/config`. Keep that file private and never commit it.

## License

_Not yet licensed._ Add a `LICENSE` file (MIT, GPL-3.0, Apache-2.0…) before publishing.
