#!/usr/bin/env python3
from __future__ import annotations

import re
import sys
from pathlib import Path
from urllib.parse import unquote

ROOT = Path(__file__).resolve().parents[1]
REQUIRED = [
    ROOT / "AGENTS.md",
    ROOT / "README.md",
    ROOT / "CHANGELOG.md",
    ROOT / "docs" / "README.md",
    ROOT / "docs" / "architecture.md",
    ROOT / "docs" / "development.md",
    ROOT / "docs" / "operations.md",
    ROOT / "docs" / "design-system.md",
    ROOT / "docs" / "ui-component-guidelines.md",
    ROOT / "docs" / "ui-component-inventory.md",
    ROOT / "docs" / "maintenance-audit.md",
]

PUBLIC_REQUIRED = [
    ROOT / "LICENSE",
    ROOT / "NOTICE",
    ROOT / "CONTRIBUTING.md",
    ROOT / "CODE_OF_CONDUCT.md",
    ROOT / "SECURITY.md",
    ROOT / "SUPPORT.md",
    ROOT / "RESPONSIBLE_USE.md",
    ROOT / "THIRD_PARTY_NOTICES.md",
    ROOT / ".github" / "CODEOWNERS",
    ROOT / ".github" / "dependabot.yml",
    ROOT / ".github" / "workflows" / "ci.yml",
    ROOT / ".github" / "workflows" / "codeql.yml",
    ROOT / ".github" / "workflows" / "secret-scan.yml",
    ROOT / ".github" / "workflows" / "release.yml",
    ROOT / "docs" / "images" / "dashboard.png",
    ROOT / "docs" / "images" / "results.png",
    ROOT / "docs" / "images" / "settings.png",
]

errors: list[str] = []
for path in REQUIRED:
    if not path.is_file():
        errors.append(f"missing required document: {path.relative_to(ROOT)}")
for path in PUBLIC_REQUIRED:
    if not path.is_file():
        errors.append(f"missing public repository file: {path.relative_to(ROOT)}")

if (ROOT / "deploy" / "ca-certificates.crt").exists():
    errors.append("generated CA bundle must not be committed; install ca-certificates in the runtime image")

if (ROOT / "go.mod").read_text(encoding="utf-8").splitlines()[0] != "module github.com/3011/cfscan":
    errors.append("go.mod must use public module path github.com/3011/cfscan")

if "REGISTRY ?= ghcr.io/3011" not in (ROOT / "Makefile").read_text(encoding="utf-8"):
    errors.append("Makefile must default to the public ghcr.io/3011 registry")

forbidden_patterns = {
    re.compile(r"\b10\.(?:\d{1,3}\.){2}\d{1,3}(?::\d+)?\b"): "private RFC1918 address",
    re.compile(r"/root/[A-Za-z0-9._/-]+"): "local root absolute path",
    re.compile(r"(?:home|cluster)-k3s"): "environment-specific GitOps path",
    re.compile("config" + "-git"): "environment-specific repository path",
    re.compile(r"\.nip\.io"): "environment-specific deployment hostname",
    re.compile(r"\b[a-z]{2}-live-\d+\b"): "production-style Agent name",
    re.compile(r"github\.com/3011/" + "cloudflare-ip-scanner"): "old module path",
}
text_suffixes = {".go", ".ts", ".tsx", ".js", ".mjs", ".json", ".yaml", ".yml", ".md", ".txt", ".toml", ".sql", ".sh", ".env", ""}
for path in ROOT.rglob("*"):
    if not path.is_file() or ".git" in path.parts or "node_modules" in path.parts or path.suffix not in text_suffixes:
        continue
    try:
        text = path.read_text(encoding="utf-8")
    except UnicodeDecodeError:
        continue
    for pattern, description in forbidden_patterns.items():
        if pattern.search(text):
            errors.append(f"{path.relative_to(ROOT)}: contains {description}")

markdown_files = sorted({*ROOT.glob("*.md"), *ROOT.glob("docs/**/*.md")})
link_pattern = re.compile(r"\[[^\]]*\]\(([^)]+)\)")
for path in markdown_files:
    text = path.read_text(encoding="utf-8")
    for line_number, line in enumerate(text.splitlines(), 1):
        if line.rstrip() != line:
            errors.append(f"{path.relative_to(ROOT)}:{line_number}: trailing whitespace")
    for match in link_pattern.finditer(text):
        target = match.group(1).strip().split()[0].strip("<>")
        if not target or target.startswith(("http://", "https://", "mailto:", "#")):
            continue
        target = unquote(target.split("#", 1)[0])
        resolved = (path.parent / target).resolve()
        try:
            resolved.relative_to(ROOT.resolve())
        except ValueError:
            errors.append(f"{path.relative_to(ROOT)}: link escapes repository: {target}")
            continue
        if not resolved.exists():
            errors.append(f"{path.relative_to(ROOT)}: broken link: {target}")

config_text = (ROOT / "internal/config/config.go").read_text(encoding="utf-8")
env_keys = sorted(set(re.findall(r'"(CFSCAN_[A-Z0-9_]+)"', config_text)))
for document in [ROOT / ".env.example", ROOT / "docs/operations.md"]:
    if not document.exists():
        continue
    text = document.read_text(encoding="utf-8")
    missing = [key for key in env_keys if key not in text]
    if missing:
        errors.append(f"{document.relative_to(ROOT)}: missing environment variables: {', '.join(missing)}")

agents = (ROOT / "AGENTS.md").read_text(encoding="utf-8") if (ROOT / "AGENTS.md").exists() else ""
if "docs/README.md" not in agents:
    errors.append("AGENTS.md must link to docs/README.md")

if errors:
    print("\n".join(errors), file=sys.stderr)
    sys.exit(1)

print(f"Documentation checks passed: {len(markdown_files)} Markdown files, {len(env_keys)} environment variables")
