# ADR-001: Input Validation Strategy

**Status**: Accepted  
**Date**: 2026-06-18  
**Deciders**: Lycaris (auditor)

## Context

sshady generates ncat ProxyCommand directives that are embedded in ~/.ssh/config.
SSH executes ProxyCommand via `$SHELL -c`, meaning unvalidated user input in
hostnames, ports, or credentials could lead to command injection.

## Decision

**All user-supplied values are validated at the boundary** before reaching any
string formatting or command generation code.

Validation layers:
1. **Type-level**: `proxy.Type` is a string enum, validated via `AllowedTypes` map
2. **Format-level**: `net.ParseIP` for IPs, regex for hostnames, `strconv.Atoi` for ports
3. **Safety-level**: `ValidateUserPass` restricts to shell-safe charset only
4. **Structural**: `ValidateHostConfig` validates the entire config holistically

Validators are centralized in `internal/proxy/proxy.go` as public functions so
they can be reused by wizard validators, CLI flag validation, and tests.

## Alternatives Considered

- **Shell escaping**: Using `shellescape` or similar libraries would be fragile
  and platform-dependent. Prefer rejecting unsafe input entirely.
- **Template-based**: Using Go templates with strict escaping. Rejected as
  over-engineering for a simple command string.
- **No validation**: Original approach. Rejected due to critical vulnerability.

## Consequences

- All inputs must pass validation before WriteEntry is called
- New proxy types must implement `Validate()` method
- Fuzz testing verifies validators never panic on arbitrary input
