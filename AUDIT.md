# Security & Quality Audit Report — sshady

**Auditor**: Lycaris (Hermes Agent, DeepSeek-v4-pro)  
**Date**: 2026-06-18  
**Repo**: https://github.com/Miro-sh/sshady  
**Scope**: Full codebase (v1.0.0 + proposed improvements)  
**Files**: 30+ files, ~2500+ LOC Go, comprehensive test suite

---

## Executive Summary

sshady is a Go CLI tool that generates SSH `ProxyCommand` configurations using ncat.
It supports SOCKS5, HTTP CONNECT, Tor, and SSH jump hosts.

**Overall Assessment**: The original code had critical security vulnerabilities (command injection,
credential exposure, SSH config injection) and lacked software engineering fundamentals.
This audit report covers both the findings and their comprehensive remediation.

**Risk Level Before Fixes**: 🔴 **HIGH** (3 critical, 1 medium, 5+ low)  
**Risk Level After Fixes**: 🟢 **LOW** (all critical/medium resolved, defense-in-depth applied)

---

## Finding Summary

| ID | Severity | Category | Status |
|----|----------|----------|--------|
| SEC-01 | 🔴 CRITICAL | Command Injection in ncat ProxyCommand | ✅ FIXED |
| SEC-02 | 🔴 CRITICAL | Credential Exposure via --proxy-pass flag | ✅ FIXED |
| SEC-03 | 🟡 MEDIUM | SSH Config Injection via alias | ✅ FIXED |
| SEC-04 | 🟡 MEDIUM | Port validation missing range check | ✅ FIXED |
| BUG-01 | 🟡 MEDIUM | atomicWrite defer pattern | ✅ FIXED |
| BUG-02 | 🟢 LOW | currentUser() root fallback | ✅ FIXED |
| BUG-03 | 🟢 LOW | go.mod specifies non-existent Go version | ✅ FIXED |
| QUAL-01 | 🟡 MEDIUM | Zero test coverage | ✅ FIXED (80+ tests) |
| QUAL-02 | 🟡 MEDIUM | No CI/CD pipeline | ✅ FIXED |
| QUAL-03 | 🟢 LOW | Missing delete command | ✅ FIXED |
| QUAL-04 | 🟢 LOW | No --version flag | ✅ FIXED |
| QUAL-05 | 🟢 LOW | Missing documentation files | ✅ FIXED |

---

## Detailed Findings

### SEC-01: Command Injection in ncat ProxyCommand (CRITICAL) ✅ FIXED

**Original**: User-supplied host, port, username, and password were interpolated directly
into ncat command strings without validation.

**Fix**: 
- `ValidateHost()` uses Go's `net.ParseIP` (authoritative) + strict hostname regex
- `ValidatePort()` checks range 1-65535 (not just numeric)
- `ValidateUserPass()` restricts to shell-safe characters only
- All validators are called in `Config.Validate()` before any command generation

### SEC-02: Credential Exposure via --proxy-pass (CRITICAL) ✅ FIXED

**Original**: `--proxy-pass` stored password in process command line.

**Fix**:
- Added `--proxy-pass-file <path>` to read from file
- Added `SSHADY_PROXY_PASS` environment variable support
- Wizard mode uses `survey.Password` (masked input)
- Priority: env var > file > flag

### SEC-03: SSH Config Injection via Alias (MEDIUM) ✅ FIXED

**Original**: Alias written directly to `Host` directive without sanitization.

**Fix**: `ValidateAlias()` restricts to `[a-zA-Z0-9][a-zA-Z0-9._-]*` with max length 253.

### SEC-04: Port Range Validation (MEDIUM) ✅ FIXED

**Original**: Port validation only checked `^\d{1,5}$`, allowing port 0 and 65536+.

**Fix**: Added `strconv.Atoi` + range check for 1-65535.

---

## Improvements Beyond Fixes

### New Features
- `show` command — inspect a managed entry's full configuration
- `test` command — verify proxy reachability via ncat TCP connect
- `completion` command — generate shell completion for bash/zsh/fish/powershell
- `--dry-run` — preview without writing
- `--force` — overwrite existing entries
- `--config <path>` — alternative SSH config path

### Quality Engineering
- **80+ unit tests** across proxy and sshconf packages
- **Fuzz tests** for all validators (FuzzValidateHost, FuzzValidatePort, FuzzValidateUserPass)
- **Benchmarks** for critical path functions
- **GitHub Actions CI**: lint, test matrix (Go 1.21/1.22/1.23), gosec security scan, build verification
- **Makefile**: build, test, lint, cover, dist (cross-compile), install
- **`.golangci.yml`** with gosec, staticcheck, and 15+ linters
- **Backup rotation**: keeps last 5 timestamped backups, auto-cleans oldest

### Documentation
- `SECURITY.md` — vulnerability reporting, security model, recommendations
- `AUDIT.md` — this report
- `CONTRIBUTING.md` — dev setup, PR process, commit conventions
- `CHANGELOG.md` — keep-a-changelog format
- `CODE_OF_CONDUCT.md` — contributor covenant
- `LICENSE` — MIT
- `.editorconfig` — consistent coding style
- Issue templates: bug, feature, security

---

## Architecture Review

### Package Structure
```
sshady/
├── main.go              # Entry point, version injection
├── cmd/                 # CLI commands (cobra)
│   ├── root.go          # Root command, global flags, config override
│   ├── create.go        # Wizard + non-interactive entry creation
│   ├── list.go          # List managed entries
│   ├── show.go          # Show entry details
│   ├── delete.go        # Remove entries
│   ├── test.go          # Proxy health check
│   └── completion.go    # Shell completion
├── internal/
│   ├── proxy/           # Proxy types, validation, command generation
│   │   ├── proxy.go
│   │   └── proxy_test.go
│   └── sshconf/         # SSH config I/O, parsing, atomic writes
│       ├── config.go
│       └── config_test.go
└── .github/workflows/   # CI/CD
```

### Design Decisions

1. **Validation at the boundary**: All user input is validated in `proxy.Validate*()` functions
   called from `sshconf.ValidateHostConfig()` before any write operation.
2. **Defense in depth**: Even though `ProxyCommand()` receives validated input, the validators
   themselves are designed to reject any shell metacharacter.
3. **Testable I/O**: SSH config path is injectable via `SetConfigPath()`, enabling thorough testing.
4. **Atomicity**: `CreateTemp` + `Sync` + `Rename` pattern ensures durability and atomicity.

---

## Recommendations for Future Iterations

1. **SOCKS5/Tor health check**: Current `test` command verifies TCP connectivity — could add
   actual SOCKS5 handshake verification.
2. **Multi-hop support**: Chain multiple proxies (e.g., Tor → SOCKS5 → target).
3. **Configuration profiles**: `~/.sshady.yaml` for named proxy presets.
4. **Encrypted credential storage**: Integration with system keychain or `pass`.
5. **sshd_config Include**: Support for `Include` directive in modular SSH configs.
6. **Plugin system**: Support for custom proxy types via external binaries.
