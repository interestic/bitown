# Contributing to bitown

Thank you for your interest in contributing!

## Before You Start

bitown is in early development (Phase 1 prototype). Large architectural changes
should be discussed in an issue before implementation.

The `assets/sprites-v1/` directory contains assets derived from
[motion-twin/WebGamesArchives](https://github.com/motion-twin/WebGamesArchives)
under **CC BY-NC-SA 4.0**. By contributing to this repository you agree that
your code contributions are licensed under **MIT**, and any asset contributions
to `assets/sprites-v1/` remain under **CC BY-NC-SA 4.0**.

## How to Contribute

### Reporting Bugs

1. Search [existing issues](https://github.com/interestic/bitown/issues) first.
2. Open a new issue with a clear title, steps to reproduce, and expected vs
   actual behavior.

### Suggesting Features

Open an issue describing the feature and the use case it solves.  
For visites / ranking mechanics changes, please include reasoning related to
the anti-bot / fairness goals described in the README.

### Submitting a Pull Request

1. Fork the repository and create a branch from `main`:
   ```
   git checkout -b fix/your-change
   ```
2. Set up local dev:
   ```bash
   cp .env.example .env
   # Edit .env — at minimum set POSTGRES_PASSWORD and DAILY_SALT_SEED
   make up
   make health   # should return {"status":"ok"}
   ```
3. Make your changes. Run `make test` and `make lint` before committing.
4. Open a PR against `main`. Describe what you changed and why.

### Commit Style

We follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add city deletion endpoint
fix(security): sanitize slug input
docs: update API reference for /support
chore: bump Go to 1.25.1
```

## Code Style

- Go: `gofmt` + `golangci-lint` (run `make lint`)
- No secrets or credentials in source — use `.env` (gitignored)
- Error paths must log at `slog.Warn` or above; never silently discard errors

## License

By submitting a pull request you agree that your contributions are licensed
under the [MIT License](./LICENSE) (code) or
[CC BY-NC-SA 4.0](./assets/sprites-v1/LICENSE) (assets in `sprites-v1/`),
as applicable.
