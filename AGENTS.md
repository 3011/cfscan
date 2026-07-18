# AGENTS.md

## Project goal

Build and maintain a centralized Cloudflare IP scanning and ranking platform. The center owns synchronization, scheduling, filtering, blacklist policy, ranking, authentication, and persistence. Lightweight regional agents only claim work, execute probes, and upload results.

## Read before changing code

1. `docs/README.md` — documentation map and sources of truth;
2. `docs/architecture.md` — data flow, lifecycle, auth, and migration semantics;
3. `docs/development.md` — local setup and required checks;
4. `docs/operations.md` — release, rollback, secrets, and health checks;
5. UI work: `docs/design-system.md` and `docs/ui-component-guidelines.md`;
6. Known debt: `docs/maintenance-audit.md`.

## Architectural boundaries

- Keep the center as a modular Go application, not microservices.
- Keep agents stateless, outbound-only, and free of scheduling policy.
- Use PostgreSQL as the source of truth and task lease store.
- Keep center-agent communication as versioned HTTPS JSON APIs.
- Scope results and blacklists to an Agent because Cloudflare IPs are Anycast.
- Do not add Redis, Kafka, NATS, ClickHouse, or Kubernetes controllers without measured need.
- The runtime schema source is `internal/store/postgres/schema.sql`; the server applies it on startup.
- Do not treat `migrations/` as an active migration runner unless the architecture is intentionally changed and documented.

## UI boundaries

- Use React 19, TypeScript 6, Tailwind CSS 4, shadcn/ui Rhea, and Base UI primitives behind project-owned wrappers.
- Business and shared code must not import `@base-ui/react` or Radix directly.
- Business and shared code must not create raw `button`, `select`, or `textarea` controls.
- Follow `docs/design-system.md`, `docs/ui-component-guidelines.md`, and `docs/ui-component-inventory.md`.
- Preserve documented local adaptations when updating official Registry components.
- Keep controls with the same visual hierarchy aligned in height, surface, text alignment, and focus treatment.

## Security and data rules

- Never commit or print passwords, session cookies, Agent tokens, database DSNs, or production exports.
- Active pairing UUIDs are short-lived secrets: never add them to logs, screenshots, fixtures, or error messages. Long-term Agent tokens must remain local; Center stores only hashes.
- Backend authorization is mandatory; hiding frontend actions is not sufficient.
- Parameterize SQL and bound external I/O with contexts/timeouts.
- Do not mutate or delete production data for UI smoke tests.
- Preserve stopped-job late-result behavior and Agent-scoped blacklist semantics.
- Agent authentication only supports independently enrolled per-Agent credentials; do not reintroduce shared tokens or the removed registration endpoint.

## Commands

```bash
make fmt
make docs-check
make test
make build
make build-web
make check
```

`make check` is the required pre-commit gate. For browser UI changes, also run the permanent smoke suite described in `docs/development.md`.

## Change checklist

Before committing:

- inspect `git status` and exclude unrelated files;
- add or update tests for behavior changes;
- update the matching document when changing config, routes, schema, UI primitives, or operations;
- update `CHANGELOG.md` for user-visible releases;
- run `make check`;
- for UI changes, validate desktop, mobile, light, dark, keyboard, overlays, and runtime errors;
- remove temporary scripts, credentials, accounts, logs, and generated reference projects.

## Deployment boundary

Source changes and GitOps deployment are separate commits. Only update deployment manifests when deployment is requested. In a shared GitOps working tree, path-limit the commit to this application and preserve unrelated changes. Roll only the components whose image or protocol changed.

## Implemented workflow

1. Center synchronizes Cloudflare official IPv4/IPv6, enabled ASN prefixes, and colo metadata.
2. An operator or schedule creates a scan job and selects one or more Agents.
3. Center samples addresses and creates Agent-scoped leased tasks.
4. Agents connect directly to each IP while preserving Host/SNI and upload metrics.
5. Center stores latest/history results, ranks by Agent/region/colo, and updates Agent-scoped temporary blacklists.
6. Due blacklist entries are sampled and scheduled for recheck.
7. Automation runs persist configuration snapshots, trigger reason, status, summary, and errors.

## Public repository rules

- The public module path is `github.com/3011/cfscan/v2` and release images use `ghcr.io/3011/cfscan-*`.
- Do not commit internal registries, private domains, local absolute paths, production Agent names, credentials, cookies, or unsanitized screenshots.
- Public releases are SemVer Git tags created after CI passes on `main`.
- Community, security, responsible-use, and third-party notice files are part of the release contract and must remain accurate.

- Read `docs/agent-enrollment.md` before changing Agent authentication, enrollment, identity persistence, or CLI behavior.
