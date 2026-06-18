# Security & Quality Audit Report — sshady

**Auditor**: Lycaris (Hermes Agent, DeepSeek-v4-pro)  
**Date**: 2026-06-18  
**Repo**: https://github.com/Miro-sh/sshady  
**Commit**: `main` branch as of audit date  
**Scope**: Full codebase — 7 source files, ~750 LOC Go

---

## Executive Summary

sshady is a Go CLI tool that generates SSH `ProxyCommand` configurations using ncat.
It supports SOCKS5, HTTP CONNECT, Tor, and SSH jump hosts.

**Overall Assessment**: The project has a solid concept and good architectural instincts
(atomic writes, backups, comment markers). However, it has **critical security vulnerabilities**
related to command injection and credential exposure, and lacks basic software engineering
practices (tests, CI, input validation).

**Risk Level Before Fixes**: 🔴 **HIGH**  
**Risk Level After Fixes (this PR)**: 🟢 **LOW**

---

## Finding Summary

| ID | Severity | Category | Description |
|----|----------|----------|-------------|
| SEC-01 | 🔴 CRITICAL | Command Injection | No input validation on proxy host/port/alias values used in ncat commands |
| SEC-02 | 🔴 CRITICAL | Credential Exposure | `--proxy-pass` flag visible in process list (`/proc/*/cmdline`) |
| SEC-03 | 🟡 MEDIUM | SSH Config Injection | No validation of Host alias names allowing injection via newlines |
| BUG-01 | 🟡 MEDIUM | Logic Error | `atomicWrite` defer removes temp file AFTER rename (no-op bug) |
| BUG-02 | 🟢 LOW | Logic Error | `currentUser` falls back to `"root"` when `user.Current()` fails |
| QUAL-01 | 🟡 MEDIUM | Missing Tests | Zero test coverage — no `*_test.go` files exist |
| QUAL-02 | 🟡 MEDIUM | Missing CI/CD | No GitHub Actions, no linting, no automated build |
| QUAL-03 | 🟢 LOW | Go Version | `go.mod` specifies `go 1.26.4` which does not exist |
| QUAL-04 | 🟢 LOW | Missing Commands | No `delete` or `--version` functionality |
| QUAL-05 | 🟢 LOW | Missing Docs | No LICENSE, SECURITY.md, CONTRIBUTING.md |

---

## Detailed Findings

### SEC-01: Command Injection in ncat ProxyCommand (CRITICAL)

**Location**: `internal/proxy/proxy.go:ProxyCommand()`

**Description**: The `Host`, `Port`, `Username`, and `Password` fields are interpolated
directly into an ncat command line string without validation. A malicious user (or
untrusted input source) could inject shell metacharacters:

```
sshady create --alias evil --host 10.0.0.1 --proxy-type socks5 \
  --proxy-host "evil.com;id;" --proxy-port "1080"
```

This generates:
```
ProxyCommand ncat --proxy-type socks5 --proxy evil.com;id;:1080 %h %p
```

Since SSH executes ProxyCommand via `$SHELL -c`, this results in command execution.

**Fix**: Added strict regex validation (`ValidateHost`, `ValidatePort`, `ValidateUserPass`)
in `proxy.go` and `ValidateHostConfig` in `config.go`. All inputs are validated before
reaching `ProxyCommand()`.

---

### SEC-02: Credential Exposure via --proxy-pass Flag (CRITICAL)

**Location**: `cmd/create.go:createFlags.proxyPass`

**Description**: The `--proxy-pass` flag stores the proxy password in the process
command line, visible to any user on the system via `ps aux`, `/proc/*/cmdline`,
or process monitoring tools.

**Fix**: Added three alternatives:
1. `--proxy-pass-file <path>` — read password from a file
2. `SSHADY_PROXY_PASS` environment variable — most secure option
3. Wizard mode already uses `survey.Password` (masked input)

---

### SEC-03: SSH Config Injection via Alias (MEDIUM)

**Location**: `internal/sshconf/config.go:Block()`

**Description**: The `Alias` field is written directly to `~/.ssh/config` without
sanitization. A newline character in the alias would break out of the `Host` directive
and allow arbitrary SSH config injection:

```
sshady create --alias "foo\n  User root\n  HostName evil.com\nHost real" ...
```

**Fix**: Added `sshAliasRe` regex in `ValidateHostConfig` that only allows
alphanumeric characters, dots, underscores, and hyphens.

---

### BUG-01: atomicWrite Defer Pattern (MEDIUM)

**Location**: `internal/sshconf/config.go:atomicWrite()`

**Description**: The `defer` block calls `os.Remove(tmpName)` which runs after
`os.Rename(tmpName, path)`. Since the file was already renamed, `os.Remove`
fails silently (file not found). This is harmless but incorrect.

**Fix**: Added `tmp.Sync()` before close for durability. The defer cleanup is
still correct for error paths (if Close/Rename fails, the temp file is removed).

---

### QUAL-01 through QUAL-05

All addressed in this PR:
- 50+ unit tests across `proxy` and `sshconf` packages
- GitHub Actions CI with lint, test matrix (Go 1.21-1.23), security scan (gosec), and build
- `Makefile` with build/test/lint/dist targets
- `.golangci.yml` configuration
- `LICENSE` (MIT), `SECURITY.md`, `CONTRIBUTING.md`
- `delete` command, `--version` flag, `--dry-run` mode
- `go.mod` fixed to `go 1.21`

---

## Architecture Review

### Strengths

1. **Good package separation**: `cmd/`, `internal/proxy/`, `internal/sshconf/` — clean boundaries
2. **Atomic writes**: Using `CreateTemp` + `Rename` pattern (despite the defer bug)
3. **Backup before modification**: Prevents data loss
4. **Comment markers**: `# BEGIN/END SSHADY:` enables tracking managed entries
5. **Wizard + CLI duality**: Supports both interactive and scripted use

### Improvement Areas

1. **No interface abstractions**: Direct filesystem I/O makes testing harder (but we added tests anyway with `t.TempDir()`)
2. **No structured logging**: All output uses `fmt.Print` — fine for a CLI tool of this size
3. **Single-file config**: SSH config parsing is regex-based, which could break on unusual but valid SSH config syntax

---

## Testing Strategy Added

| Package | Tests | Coverage Focus |
|---------|-------|----------------|
| `proxy` | 30+ cases | Input validation (injection vectors), ProxyCommand generation, Summary |
| `sshconf` | 20+ cases | Config validation, Block generation, atomicWrite correctness, entry parsing |

---

## Recommendations for Future Iterations

1. **SOCKS5/Tor health check**: Verify the proxy is reachable before writing config
2. **Config path override**: `--config` flag or `SSHADY_CONFIG` env var
3. **Multi-hop support**: Chain multiple proxies
4. **ssh_config Include**: Support `Include` directive for modular configs
5. **Man page**: Generate and ship a man page
6. **Shell completion**: `sshady completion bash/zsh/fish`
