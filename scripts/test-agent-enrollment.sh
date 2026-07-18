#!/bin/sh
set -eu

network="cfscan-enrollment-test-$$"
postgres="cfscan-enrollment-postgres-$$"
server="cfscan-enrollment-server-$$"
agent="cfscan-enrollment-agent-$$"
agent_volume="cfscan-enrollment-agent-data-$$"
server_image="${CFSCAN_ENROLLMENT_SERVER_IMAGE:-cfscan-server:enrollment-test}"
agent_image="${CFSCAN_ENROLLMENT_AGENT_IMAGE:-cfscan-agent:enrollment-test}"
python_image="${CFSCAN_ENROLLMENT_PYTHON_IMAGE:-python:3-alpine}"
postgres_image="${CFSCAN_ENROLLMENT_POSTGRES_IMAGE:-postgres:17-alpine}"
database_password="$(python3 -c 'import secrets; print(secrets.token_urlsafe(24))')"
admin_password="$(python3 -c 'import secrets; print(secrets.token_urlsafe(24))')"

cleanup() {
  docker rm -f "$agent" "$server" "$postgres" >/dev/null 2>&1 || true
  docker volume rm -f "$agent_volume" >/dev/null 2>&1 || true
  docker network rm "$network" >/dev/null 2>&1 || true
}
trap cleanup EXIT INT TERM

if [ "${CFSCAN_ENROLLMENT_SKIP_BUILD:-0}" != "1" ]; then
  docker build \
    --build-arg "HTTP_PROXY=${HTTP_PROXY:-}" \
    --build-arg "HTTPS_PROXY=${HTTPS_PROXY:-}" \
    --build-arg "NO_PROXY=${NO_PROXY:-}" \
    -f Dockerfile.server -t "$server_image" . >/dev/null
  docker build \
    --build-arg "HTTP_PROXY=${HTTP_PROXY:-}" \
    --build-arg "HTTPS_PROXY=${HTTPS_PROXY:-}" \
    --build-arg "NO_PROXY=${NO_PROXY:-}" \
    --build-arg VERSION=integration-test \
    -f Dockerfile.agent -t "$agent_image" . >/dev/null
fi

docker network create "$network" >/dev/null
docker run -d --name "$postgres" --network "$network" \
  -e POSTGRES_DB=cfscan \
  -e POSTGRES_USER=cfscan \
  -e "POSTGRES_PASSWORD=$database_password" \
  "$postgres_image" >/dev/null

for _ in $(seq 1 40); do
  if docker exec "$postgres" pg_isready -h 127.0.0.1 -U cfscan -d cfscan >/dev/null 2>&1; then
    break
  fi
  sleep 1
done
docker exec "$postgres" pg_isready -h 127.0.0.1 -U cfscan -d cfscan >/dev/null

docker run -d --name "$server" --network "$network" \
  -e CFSCAN_HTTP_ADDR=:8080 \
  -e "CFSCAN_DATABASE_URL=postgres://cfscan:${database_password}@${postgres}:5432/cfscan?sslmode=disable" \
  -e CFSCAN_BOOTSTRAP_ADMIN_USERNAME=admin \
  -e "CFSCAN_BOOTSTRAP_ADMIN_PASSWORD=$admin_password" \
  -e CFSCAN_COOKIE_SECURE=false \
  -e CFSCAN_PUBLIC_URL=http://public.example.test \
  "$server_image" >/dev/null

for _ in $(seq 1 60); do
  if [ "$(docker inspect -f '{{.State.Running}}' "$server" 2>/dev/null || true)" != "true" ]; then
    echo "Center exited before becoming healthy" >&2
    docker logs "$server" --tail 120 >&2 || true
    exit 1
  fi
  if docker exec "$server" wget -qO- http://127.0.0.1:8080/healthz >/dev/null 2>&1; then
    break
  fi
  sleep 1
done
if ! docker exec "$server" wget -qO- http://127.0.0.1:8080/healthz >/dev/null 2>&1; then
  echo "Center did not become healthy" >&2
  docker logs "$server" --tail 120 >&2 || true
  exit 1
fi

client_log="$(mktemp)"
set +e
docker run --rm -i --network "$network" \
  -e "CFSCAN_TEST_SERVER=http://${server}:8080" \
  -e "CFSCAN_TEST_ADMIN_PASSWORD=$admin_password" \
  "$python_image" python - >"$client_log" 2>&1 <<'PY'
import http.cookiejar
import json
import os
import secrets
import time
import urllib.error
import urllib.request
import uuid

BASE = os.environ["CFSCAN_TEST_SERVER"]
ADMIN_PASSWORD = os.environ["CFSCAN_TEST_ADMIN_PASSWORD"]


def request(path, data=None, headers=None, opener=None, expected=None):
    payload = None if data is None else json.dumps(data).encode()
    req = urllib.request.Request(
        BASE + path,
        data=payload,
        headers={"Accept": "application/json", "Content-Type": "application/json", **(headers or {})},
    )
    client = opener or urllib.request.build_opener()
    try:
        with client.open(req, timeout=10) as response:
            body = response.read()
            result = None if not body else json.loads(body)
            if expected is not None and response.status != expected:
                raise AssertionError(f"{path}: expected {expected}, got {response.status}")
            return response.status, result
    except urllib.error.HTTPError as error:
        body = error.read()
        result = None if not body else json.loads(body)
        if expected is not None and error.code == expected:
            return error.code, result
        raise AssertionError(f"{path}: unexpected HTTP {error.code}") from error


for _ in range(60):
    try:
        request("/healthz", expected=200)
        break
    except Exception:
        time.sleep(1)
else:
    raise SystemExit("Center did not become healthy")

cookies = http.cookiejar.CookieJar()
admin = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(cookies))
request("/api/v1/auth/login", {"username": "admin", "password": ADMIN_PASSWORD}, opener=admin, expected=200)
_, config = request("/api/v1/agent-enrollments/config", opener=admin, expected=200)
assert config["public_url"] == "http://public.example.test"
assert "public_web_url" not in config and "public_agent_url" not in config

_, device = request(
    "/api/v1/agent/enrollments",
    {"name": "worker-test-01", "os": "linux", "architecture": "amd64", "version": "test", "concurrency": 32},
    expected=201,
)
pairing_token = device["pairing_token"]
assert len(pairing_token) == 36
assert device["verification_uri_complete"] == f"http://public.example.test/agents/pair/{pairing_token}"

_, pending = request("/api/v1/agent-enrollments", opener=admin, expected=200)
assert len(pending["items"]) == 1
assert pending["items"][0]["status"] == "pending"
enrollment_id = pending["items"][0]["id"]
request(
    f"/api/v1/agent-enrollments/id/{enrollment_id}/approve",
    {"name": "worker-test-01", "region": "Test Region", "continent": "Asia", "concurrency": 24},
    opener=admin,
    expected=200,
)

credential_id = str(uuid.uuid4())
credential_secret = "secret_with_underscores_" + secrets.token_urlsafe(24)
claim_input = {
    "pairing_token": pairing_token,
    "credential_id": credential_id,
    "credential_secret": credential_secret,
    "os": "linux",
    "architecture": "amd64",
    "version": "test",
}
_, claim = request("/api/v1/agent/enrollments/claim", claim_input, expected=200)
assert claim["status"] == "claimed"
assert claim["concurrency"] == 24
agent_id = claim["agent_id"]

_, retry = request("/api/v1/agent/enrollments/claim", claim_input, expected=200)
assert retry["agent_id"] == agent_id

agent_token = f"cfa_{credential_id}_{credential_secret}"
request(
    "/api/v1/agent/heartbeat",
    {},
    {"Authorization": f"Bearer {agent_token}"},
    expected=204,
)
request(
    "/api/v1/agent/heartbeat",
    {"agent_id": agent_id},
    {"Authorization": f"Bearer {agent_token}"},
    expected=400,
)
_, agents = request("/api/v1/agents", opener=admin, expected=200)
agent = next(item for item in agents["items"] if item["id"] == agent_id)
assert "auth_mode" not in agent
assert agent["name"] == "worker-test-01"
assert agent["os"] == "linux"

# Shared-token authentication and the old registration endpoint were removed in v2.
request(
    "/api/v1/agent/heartbeat",
    {},
    {"Authorization": "Bearer removed-shared-token"},
    expected=401,
)

def raw_status(path, data, headers):
    req = urllib.request.Request(
        BASE + path,
        data=json.dumps(data).encode(),
        headers={"Accept": "application/json", "Content-Type": "application/json", **headers},
    )
    try:
        with urllib.request.urlopen(req, timeout=10) as response:
            return response.status
    except urllib.error.HTTPError as error:
        return error.code

assert raw_status(
    "/api/v1/agent/register",
    {"name": "removed-registration"},
    {"Authorization": f"Bearer {agent_token}"},
) == 404

_, preauthorized = request(
    "/api/v1/agent-enrollments/preauthorized",
    {"name": "auto-test-01", "region": "Automation", "continent": "Europe", "concurrency": 12, "ttl_minutes": 30},
    opener=admin,
    expected=201,
)
preauthorized_input = {
    "pairing_token": preauthorized["pairing_token"],
    "credential_id": str(uuid.uuid4()),
    "credential_secret": secrets.token_urlsafe(32),
    "os": "linux",
    "architecture": "arm64",
    "version": "test",
}
_, preauthorized_claim = request("/api/v1/agent/enrollments/claim", preauthorized_input, expected=200)
assert preauthorized_claim["status"] == "claimed"
request(
    "/api/v1/agent/enrollments/claim",
    {
        "pairing_token": preauthorized["pairing_token"],
        "credential_id": str(uuid.uuid4()),
        "credential_secret": secrets.token_urlsafe(32),
    },
    expected=409,
)

_, rejected = request(
    "/api/v1/agent/enrollments",
    {"name": "rejected-test", "os": "linux", "architecture": "amd64", "version": "test", "concurrency": 4},
    expected=201,
)
_, active = request("/api/v1/agent-enrollments", opener=admin, expected=200)
rejected_item = next(item for item in active["items"] if item["requested_name"] == "rejected-test")
request(f"/api/v1/agent-enrollments/id/{rejected_item['id']}/reject", {}, opener=admin, expected=200)
request(
    "/api/v1/agent/enrollments/claim",
    {
        "pairing_token": rejected["pairing_token"],
        "credential_id": str(uuid.uuid4()),
        "credential_secret": secrets.token_urlsafe(32),
    },
    expected=403,
)

print("Agent enrollment integration passed")
PY
status=$?
set -e

if [ "$status" -ne 0 ]; then
  cat "$client_log" >&2
  docker logs "$server" --tail 120 >&2 || true
  rm -f "$client_log"
  exit "$status"
fi
cat "$client_log"
rm -f "$client_log"

# Exercise the real Agent image, writable named volume, approval polling, and identity reuse.
docker volume create "$agent_volume" >/dev/null
docker run -d --name "$agent" --network "$network" \
  -v "$agent_volume:/var/lib/cfscan-agent" \
  "$agent_image" connect \
    --server "http://${server}:8080" \
    --allow-insecure-http \
    --name cli-test-agent \
    --concurrency 7 \
    --pair-only >/dev/null

approval_log="$(mktemp)"
set +e
docker run --rm -i --network "$network" \
  -e "CFSCAN_TEST_SERVER=http://${server}:8080" \
  -e "CFSCAN_TEST_ADMIN_PASSWORD=$admin_password" \
  "$python_image" python - >"$approval_log" 2>&1 <<'PYCLIENT'
import http.cookiejar
import json
import os
import time
import urllib.request

base = os.environ["CFSCAN_TEST_SERVER"]
jar = http.cookiejar.CookieJar()
client = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(jar))


def request(path, data=None):
    payload = None if data is None else json.dumps(data).encode()
    req = urllib.request.Request(
        base + path,
        data=payload,
        headers={"Content-Type": "application/json", "Accept": "application/json"},
    )
    with client.open(req, timeout=10) as response:
        body = response.read()
        return None if not body else json.loads(body)


request("/api/v1/auth/login", {"username": "admin", "password": os.environ["CFSCAN_TEST_ADMIN_PASSWORD"]})
for _ in range(60):
    items = request("/api/v1/agent-enrollments")["items"]
    match = next((item for item in items if item["requested_name"] == "cli-test-agent"), None)
    if match is not None:
        request(
            f"/api/v1/agent-enrollments/id/{match['id']}/approve",
            {"name": "cli-test-agent", "region": "CLI Test", "continent": "Asia", "concurrency": 7},
        )
        break
    time.sleep(1)
else:
    raise SystemExit("real Agent did not create an enrollment")
PYCLIENT
approval_status=$?
set -e
if [ "$approval_status" -ne 0 ]; then
  cat "$approval_log" >&2
  docker logs "$agent" >&2 || true
  rm -f "$approval_log"
  exit "$approval_status"
fi
rm -f "$approval_log"

agent_status="$(docker wait "$agent")"
if [ "$agent_status" != "0" ]; then
  docker logs "$agent" >&2 || true
  exit "$agent_status"
fi
identity_mode="$(docker run --rm -v "$agent_volume:/data" alpine:3.24 stat -c '%a' /data/identity.json)"
[ "$identity_mode" = "600" ]

docker run --rm --network "$network" \
  -v "$agent_volume:/var/lib/cfscan-agent" \
  "$agent_image" connect \
    --server "http://${server}:8080" \
    --allow-insecure-http \
    --pair-only >/dev/null

echo "Agent CLI enrollment integration passed"
