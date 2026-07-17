# Security Policy

## Supported versions

Security fixes are provided for the latest minor release in the current major version.

## Reporting a vulnerability

Do not open a public issue for a suspected vulnerability.

Use GitHub's **Report a vulnerability** / private security advisory feature for this repository. Include:

- affected version and deployment method;
- impact and realistic attack scenario;
- reproduction steps or proof of concept;
- relevant configuration with secrets removed;
- suggested mitigation, if known.

You should receive an acknowledgement within 7 days. Fix timing depends on severity and reproducibility.

## Scope

Useful reports include authentication or authorization bypass, credential exposure, unsafe task execution, SSRF or request smuggling, SQL injection, stored XSS, insecure defaults, supply-chain compromise, and cross-tenant data exposure.

Network scan results, latency variation, Cloudflare behavior, and issues requiring access to an already-compromised administrator account are not automatically vulnerabilities.

## Operational responsibility

Operators must use unique Agent tokens, HTTPS, secure session cookies, restricted database credentials, and private backup storage. See [`docs/operations.md`](docs/operations.md).
