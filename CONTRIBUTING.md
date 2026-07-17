# Contributing to CF Scanner

Thank you for helping improve CF Scanner.

## Before opening a change

1. Read [`AGENTS.md`](AGENTS.md) and [`docs/README.md`](docs/README.md).
2. Search existing issues and discussions.
3. For large behavioral, schema, protocol, or design-system changes, open an issue first.
4. Do not include credentials, production data, internal hostnames, or screenshots with sensitive values.

## Development

```bash
cp .env.example .env
docker compose --profile agent up -d --build
```

The console is available at `http://localhost:18081`. See [`docs/development.md`](docs/development.md) for split-process development and browser tests.

## Required checks

```bash
make check
```

Changes to UI behavior should also run the browser regression suite described in [`docs/development.md`](docs/development.md).

## Pull requests

- Keep one concern per pull request.
- Add or update tests for behavior changes.
- Update documentation when configuration, API behavior, data semantics, UI rules, or deployment steps change.
- Add a changelog entry under `Unreleased` when the change affects users.
- Explain security and compatibility impact.
- Do not mix formatting-only changes with behavioral changes.

## Commit messages

Use concise imperative messages. Conventional Commit prefixes such as `feat:`, `fix:`, `docs:`, `test:`, and `refactor:` are preferred.

## License

By submitting a contribution, you agree that it will be licensed under the Apache License 2.0.
