# Changelog

All notable public changes are documented here.

## [Unreleased]

## [1.1.1] - 2026-07-17

### Security

- Enforced the 10,000-target sampling limit inside the target sampler itself.
- Removed request-sized Slice and Map preallocations flagged by CodeQL while retaining existing API and query limits.

## [1.1.0] - 2026-07-17

### Added

- Added browser-approved Agent pairing with short-lived UUID enrollment keys.
- Added preauthorized one-time deployment commands for unattended installations.
- Added per-Agent long-term credentials, atomic local identity persistence, and `connect` / `join` CLI commands.
- Added pending enrollment management and approval UI for desktop and mobile layouts.

### Security

- Center stores only hashes of pairing keys and long-term Agent secrets.
- Per-Agent credentials bind heartbeat, task claim, and result upload requests to the authenticated Agent identity.
- New Agent enrollment requires HTTPS except for loopback development or an explicit insecure override.

### Compatibility

- Existing shared-token Agents remain supported during migration and are marked as legacy authentication in the UI.

## [1.0.0] - 2026-07-17

### Security

- Updated `chi`, `pgx`, and `golang.org/x/crypto` to patched release lines before the first public release.
- Added CodeQL, Gitleaks, Dependabot, dependency review, and GitHub secret-scanning guidance.

### Added

- Initial public release of the Center, regional Agent, PostgreSQL store, and React management console.
- Distributed Cloudflare Anycast probing with TCP, TLS, TTFB, availability, CF-RAY, and colo measurements.
- Server-side ranking, pagination, automation, temporary blacklist rechecks, RBAC, and Rhea-based responsive UI.
- Docker Compose development environment, multi-architecture GHCR release workflow, CI, security policy, and contributor documentation.
