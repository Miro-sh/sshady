# Changelog

All notable changes to sshady will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Security
- **Critical**: Input validation for all user-supplied values to prevent command injection in ncat ProxyCommand directives
- **Critical**: Add `--proxy-pass-file` flag and `SSHADY_PROXY_PASS` env var to avoid credential exposure in process list
- **High**: SSH alias validation to prevent config injection via malformed aliases
- Port validation now checks range 1-65535 (not just numeric)
- Host validation uses Go's `net.ParseIP` for authoritative IP parsing

### Added
- `delete` command — remove sshady-managed entries from SSH config
- `show` command — display details of a managed entry
- `test` command — verify proxy reachability via ncat
- `completion` command — generate shell completion for bash, zsh, fish, powershell
- `--dry-run` global flag — preview changes without writing
- `--force` global flag — overwrite existing entries without confirmation
- `--config` global flag — use alternative SSH config path
- `--version` flag — display build version
- Backup rotation: keeps last 5 timestamped backups automatically
- Comprehensive test suite (80+ test cases, fuzz tests, benchmarks)
- GitHub Actions CI/CD pipeline
- Makefile with build, test, lint, cover, dist targets
- `.golangci.yml` linting configuration
- `SECURITY.md` vulnerability reporting policy
- `CONTRIBUTING.md` development guide
- `AUDIT.md` security audit report
- `CHANGELOG.md`
- `.editorconfig` for consistent coding style
- `CODE_OF_CONDUCT.md`
- `LICENSE` (MIT)

### Fixed
- `atomicWrite` defer pattern: added `Sync()` before close, proper cleanup on error
- `currentUser()` no longer defaults to hardcoded `"root"` on error
- `go.mod` fixed: `go 1.26.4` → `go 1.21` (1.26.4 does not exist)

### Changed
- Refactored validation into centralized `proxy.Validate*()` functions
- Improved error messages with remediation hints
- Wizard now validates inputs inline during prompts
- Enhanced README with security notes, command table, and flags reference

## [1.0.0] — Initial Release

### Added
- Interactive wizard for creating SSH proxy configs
- Non-interactive mode via CLI flags
- Support for SOCKS5, HTTP CONNECT, Tor, and SSH jump hosts
- Atomic writes to `~/.ssh/config`
- Automatic backup before modifications
- `list` command to display managed entries
- Comment markers for tracking sshady-managed blocks
