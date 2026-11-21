# OneMore

Self-hostable enterprise AI gateway and admin console. Admins configure model
channels, create members, distribute enterprise quota, and audit every call.

This repository currently implements **M1: the normal happy path** against a
local fake upstream. It does not call real model vendors.

Product scope: [`PHASE1.md`](./PHASE1.md). Technical design: [`SPEC.md`](./SPEC.md).
Agent conventions: [`AGENTS.md`](./AGENTS.md).

## M1 flow

1. Admin creates a model channel (admin-configured upstream URL + allowlisted models).
2. Admin creates a member and assigns an enterprise quota; the member token is returned once.
3. The member calls `POST /v1/chat/completions` with that token.
4. The server authenticates the member, passes both egress gates, reserves quota,
   calls the upstream, settles quota from usage, and writes an audit row.
5. Admin lists the audit and finds the call by request ID.

## Quick start (Docker Compose)

Requires Docker with the Compose plugin.

```bash
export ONEMORE_ADMIN_TOKEN="$(openssl rand -hex 32)"
export ONEMORE_MASTER_KEY="$(openssl rand -hex 32)"
export MYSQL_PASSWORD="$(openssl rand -hex 32)"
export MYSQL_ROOT_PASSWORD="$(openssl rand -hex 32)"
docker compose up --build
```

Keep these generated values in your deployment secret store and reuse them on restart.
Do not regenerate the master key for an existing database: stored channel credentials
would become unreadable. Compose fails if any required value is absent. Its host port
is bound to loopback only; this MVP is not production-ready and has no hardened
browser sessions/CSRF yet. Do not expose it directly to the internet.

Open <http://localhost:8080> and paste the admin token. The console is served by
the Go binary on the same origin and can switch between English and 中文; the
choice is remembered in the browser. MySQL stays on the internal Compose network
and is not published.

## Local development

Requires Go, Node, and a local MySQL 8.4.

```bash
# 1. database (local dev only)
mysql -uroot -e "CREATE DATABASE IF NOT EXISTS onemore CHARACTER SET utf8mb4;"
mysql -uroot -e "CREATE USER IF NOT EXISTS 'onemore'@'127.0.0.1' IDENTIFIED BY 'onemore';"
mysql -uroot -e "GRANT ALL PRIVILEGES ON onemore.* TO 'onemore'@'127.0.0.1';"

# 2. fake upstream (records how many requests it actually received)
go run ./cmd/fakeupstream          # 127.0.0.1:9099

# 3. API server (runs migrations on startup)
ONEMORE_ADMIN_TOKEN=dev-admin-token \
ONEMORE_DSN='onemore:onemore@tcp(127.0.0.1:3306)/onemore?parseTime=true' \
go run ./cmd/server                # 127.0.0.1:8080

# 4. console
cd web && npm install && npm run build   # build once; Go serves web/dist
# or, for hot reload: npm run dev (Vite proxies /api and /v1 to :8080)
```

## Configuration

| Variable | Default | Purpose |
|---|---|---|
| `ONEMORE_ADDR` | `127.0.0.1:8080` | Listen address |
| `ONEMORE_DSN` | local `onemore` DSN | MySQL connection string |
| `ONEMORE_ADMIN_TOKEN` | `dev-admin-token` | MVP admin bearer token |
| `ONEMORE_MASTER_KEY` | empty | AES-GCM key for stored credentials; empty uses an insecure dev key |
| `ONEMORE_WEB_DIR` | `web/dist` | Built console directory |
| `FAKE_UPSTREAM_ADDR` | `127.0.0.1:9099` | Fake upstream listen address |

**Non-loopback listening requires distinct, non-default admin and master secrets, each at least 32 characters.**
The server rejects unsafe shared-deployment settings before connecting to MySQL.
Development defaults are only permitted for an explicit loopback IP bind.

## API (summary)

Admin endpoints use `Authorization: Bearer <ONEMORE_ADMIN_TOKEN>`:

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/api/channels` | List channels (keys masked) |
| `POST` | `/api/channels` | Create channel |
| `PATCH` | `/api/channels/{id}` | Enable / disable |
| `POST` | `/api/channels/{id}/test` | Connectivity check |
| `GET` | `/api/members` | List members and quota |
| `POST` | `/api/members` | Create member; returns token once |
| `PUT` | `/api/members/{id}/quota` | Set enterprise quota |
| `PUT` | `/api/members/{id}/status` | Activate / disable |
| `GET` | `/api/policy` | Read deterministic egress terms |
| `PUT` | `/api/policy` | Replace egress terms |
| `GET` | `/api/audits` | Recent call audit; optional exact `request_id` filter |

Member endpoint:

```bash
curl -s -D - http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer <member-token>" \
  -H 'Content-Type: application/json' \
  -d '{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hello"}]}'
```

The JSON `id` is the model vendor's original response id, unchanged. The platform
audit id is returned in the `X-Request-ID` response header and can be used to look
up exactly one audit record:

```bash
curl -s "http://localhost:8080/api/audits?request_id=<X-Request-ID>" \
  -H "Authorization: Bearer <ONEMORE_ADMIN_TOKEN>"
```

## Locked semantics in this build

- **Two egress gates:** model allowlist (where) and deterministic content terms (what).
  A blocked request never reaches the upstream.
- **Quota:** reserve before the upstream call, settle once from usage. Separate
  enterprise ledger fields keep used and reserved distinct.
- **Error codes:** `POLICY_BLOCKED` 403, `ORG_QUOTA_EXCEEDED` 429,
  `UPSTREAM_RATE_LIMITED` 429, `UPSTREAM_RESULT_UNKNOWN` 504/502.
- **Audit** records metadata only: no request/response bodies, no credentials.
- Credentials are encrypted at rest and only ever shown masked.

## Tests

```bash
go test -race ./...        # unit tests, no external services

# integration: disposable MySQL database, tables are cleared by the test
mysql -uroot -e "CREATE DATABASE IF NOT EXISTS onemore_test CHARACTER SET utf8mb4;"
ONEMORE_TEST_DSN='onemore:onemore@tcp(127.0.0.1:3306)/onemore_test?parseTime=true' \
  go test -run TestAuditRequestIDIsolation -v ./internal/api/
```

## Not in M1

Personal BYO key, server-side browser sessions / CSRF, streaming, retries,
multi-tenancy, real vendor calls, concurrency/persistence acceptance, and the
clean-environment install acceptance still belong to later phases (see
`AGENTS.md`). The admin token is an MVP placeholder, not the final console auth.

## License

OneMore's first-party code is licensed under the [Apache License 2.0](LICENSE).
Third-party components keep their own licenses. See
[Third-party notices and distribution checklist](THIRD_PARTY_NOTICES.md).
