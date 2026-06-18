# Contributing to sshady

Thanks for your interest in contributing!

## Development Setup

```bash
git clone https://github.com/Miro-sh/sshady.git
cd sshady
go mod download
```

## Building

```bash
# Build
make build
# or
go build -o sshady .
```

## Testing

```bash
# Run all tests with race detector
make test

# Quick tests
make test-short
```

## Linting

```bash
# Install golangci-lint: https://golangci-lint.run/usage/install/
make lint
```

## Code Style

- Follow standard Go conventions (`go fmt`, `go vet`)
- All user input must be validated (see `internal/proxy/proxy.go` for validators)
- New features must include tests
- Security-sensitive code must be reviewed by a maintainer

## Pull Request Process

1. Fork the repo and create your branch from `main`
2. Add tests for any new functionality
3. Ensure `make lint` and `make test` pass
4. Update documentation if needed
5. Submit your PR with a clear description

## Commit Conventions

We use [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` — new feature
- `fix:` — bug fix
- `docs:` — documentation
- `test:` — tests
- `refactor:` — code restructuring
- `ci:` — CI/CD changes
- `security:` — security improvements
