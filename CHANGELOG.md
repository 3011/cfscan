# Changelog

All notable public changes are documented here.

## [Unreleased]

## [2.1.0] - 2026-07-18

### Changed

- Unified all create, edit, and enrollment workflows on centered Dialogs with fixed headers, scrollable content, and persistent action footers.
- Reserved Sheet for complementary interface surfaces such as the mobile Sidebar, with an automated UI boundary preventing business features from reintroducing side sheets.


## [2.0.0] - 2026-07-18

### Changed

- Unified the Web console, management API, and Agent API behind one public origin configured by `CFSCAN_PUBLIC_URL`.
- Updated the Go module path to `github.com/3011/cfscan/v2` for semantic import versioning.
- Agent `run` now requires a saved independent identity; `connect` and `join` are the only enrollment paths.

### Removed

- Removed shared Agent token authentication and the `CFSCAN_AGENT_TOKEN` setting.
- Removed `POST /api/v1/agent/register` and the legacy Agent registration environment variables.
- Removed the `auth_mode` API/database field and legacy authentication labels from the Web console.
- Removed the split `CFSCAN_PUBLIC_WEB_URL` and `CFSCAN_PUBLIC_AGENT_URL` settings.

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
