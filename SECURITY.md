# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability in sshady, please **do not** open a public issue.
Instead, send details to the maintainer privately.

## Security Model

sshady generates SSH proxy configurations and writes them to `~/.ssh/config`.
It is designed with the following security properties:

### What we protect

- **Atomicity**: All writes to `~/.ssh/config` are atomic (write to temp file, fsync, rename).
- **Backups**: A backup is created at `~/.ssh/config.sshady.bak` before every modification.
- **Permissions**: `~/.ssh/config` is always written with `0600` permissions.
- **Input validation**: All user-supplied values (hostnames, aliases, ports, credentials) are validated
  against strict regexes to prevent SSH config injection and command injection in ncat ProxyCommand directives.

### What we do NOT protect

- **Credentials at rest**: If you use proxy authentication, the username and password are stored
  in **plaintext** inside `~/.ssh/config`. The file has `0600` permissions, but anyone with root
  or filesystem access can read them.
- **Credentials in process list**: Using the `--proxy-pass` flag exposes the password in the
  process list visible via `ps aux`. Use `--proxy-pass-file` or the `SSHADY_PROXY_PASS` environment
  variable instead.
- **ncat dependency**: The generated ProxyCommand relies on `ncat` being present. If `ncat` is
  compromised or replaced, SSH traffic may be intercepted.

### Recommendations for users

1. **Use `--proxy-pass-file` or `SSHADY_PROXY_PASS` env var** instead of `--proxy-pass`
2. **Ensure `~/.ssh/` has `0700` permissions** and `~/.ssh/config` has `0600`
3. **Never commit `~/.ssh/config`** to version control
4. **Verify your ncat installation** is from a trusted source (nmap package)
5. **Use Tor for maximum anonymity** when the proxy itself might be traced

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 1.x.x   | :white_check_mark: |

## Security Features by Version

- **1.0.0+**: Atomic writes, input validation, backup before write, 0600 permissions, env var support for passwords
