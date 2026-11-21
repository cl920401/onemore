# AGENTS.md

Guidance for agentic coding agents working in this repository.

## Project

**OneMore** — a self-hostable enterprise AI gateway + admin console for small/mid-size
companies. Admins configure model channels, distribute quota, enforce egress policy, and
audit every call. This is **not** a general chat product.

- `PHASE1.md` — authoritative product scope, acceptance scenarios, and out-of-scope list.
- `SPEC.md` — proposed technical design. Treat as a draft unless a decision is recorded here.

Status: **M1 in place.** M1 is the normal happy path: Go API + MySQL 8.4 + Vue 3/TS/Vite
console + a local fake upstream (`cmd/fakeupstream`). Console auth is still an MVP admin
bearer token; hardened server-side sessions, personal keys, and exception branches come later.
The first Go-only experiment (fake upstream, two fake keys) preceded this and still belongs to
`cmd/fakeupstream`.

If docs and this file conflict, `PHASE1.md` wins on scope; this file wins on conventions.

## Repository layout (target)

```
cmd/                  entrypoints (wire deps, start/stop lifecycle)
internal/
  authz/              identity & role checks
  quota/              reservation + settlement ledgers
  egress/             model allowlist gate + deterministic content gate
  router/             channel selection
  audit/              call + config audit
web/                  Vue 3 + TS + Vite console (built same-origin by Go)
migrations/           versioned SQL (never edit existing migrations)
docker-compose.yml    app + db
```

Dependency rule: `handler -> service -> repository`. Never import `handler` from `service`.

## Commands

Go (experiment and later):
```bash
go build ./...
go test ./...                 # full suite
go test -race ./...           # required before finishing changes
go vet ./...
gofmt -l .                    # must print nothing
golangci-lint run ./...       # if installed
```

Run a single test / package:
```bash
go test -run TestEgressBothGatesZeroUpstream ./internal/egress/
go test -run TestEnterprise429NoAutoSwitch -v ./...
```

Frontend (only after the product baseline is scaffolded — not part of the first experiment):
```bash
npm install
npm run dev
npm run build
npm run test
npx vitest run src/path/File.test.ts   # single test
```

Docker (product acceptance phase only):
```bash
docker compose up
```

Run Go commands from the repo root. The fake upstream is `go run ./cmd/fakeupstream` and the
API is `go run ./cmd/server`. Run frontend commands from `web/`. `go build ./...` does not
require MySQL; `go test ./...` uses no external DB. Do not add dependencies unless asked.

## Go conventions

- Pass `ctx context.Context` as the first parameter for I/O, RPC, DB, and upstream calls.
- Always check errors; wrap with context: `fmt.Errorf("settle quota: %w", err)`. No `panic` for
  normal control flow.
- Keep interfaces small and define them where consumed; `-er` naming for single-method interfaces.
- Use the project structured logger; never `fmt.Print*` for logs. Do not read raw env vars inside
  business packages — inject config.
- Use short transactions for quota reserve/settle; never hold a DB lock across a network call.
- Prefer table-driven tests in `*_test.go`; no external network/DB in unit tests.
- Formatting is `gofmt`; imports grouped stdlib / third-party / local.

## Vue / TypeScript conventions

- `<script setup>` with Composition API; strict TypeScript, no `any` unless justified.
- PascalCase components, `useX` composables, camelCase variables.
- Authorization is **server-enforced** in Go. Never rely on hiding a UI button as a permission check.

## Locked domain invariants

- **Egress is two gates, both must pass:** the model allowlist decides *where* a request may go;
  the deterministic rule decides *what* may leave. Only admin-configured upstream URLs are allowed.
- **Audit never stores request/response bodies or secrets by default.** Credentials are masked in
  responses and never logged.
- **Quota:** reserve before the upstream call, settle once per request ID. Enterprise and personal
  ledgers are separate; never double-count.
- **No auto-fallback between payment sources.** An upstream rate limit must never transparently
  retry with the personal key.
- **Personal key** is usable only when the org switch allows it, the request explicitly selects it,
  and the key belongs to the requesting member. It still passes both egress gates.

## Business error codes (frozen four)

| Code | HTTP | Audit | Notes |
|---|---|---|---|
| `POLICY_BLOCKED` | 403 | policy_rejected | failed allowlist or content gate; never sent upstream |
| `ORG_QUOTA_EXCEEDED` | 429 | quota_exceeded | internal enterprise quota; not sent upstream |
| `UPSTREAM_RATE_LIMITED` | 429 | rate_limited | upstream 429; do not switch payer |
| `UPSTREAM_RESULT_UNKNOWN` | 504 timeout / 502 disconnected-or-unverifiable | unknown | keep reservation; no retry, no zeroing, no personal switch |

`ORG_QUOTA_EXCEEDED` and `UPSTREAM_RATE_LIMITED` may both be HTTP 429 but must have distinct
business codes and audit results. A blocked request must produce **zero** upstream calls.

## Security

- Console auth uses **server-side sessions**. Cookie: `HttpOnly`, `Secure` under HTTPS; protect
  write endpoints against CSRF. External client call credentials are separate from browser sessions.
- Encrypt stored credentials; the runtime master key never goes into the database or Git.
- Never commit secrets. Do not read real credentials, push, or deploy without explicit instruction.

## Out of scope (do not add)

No SSE/streaming in v0.1. No CLI, no Redis, no message queue, no multi-tenancy, no vector store,
no Kubernetes, no offline inference. Do not assume One API or New API; whether to reuse it is
undecided until a same-acceptance comparison is run.

## Rules files

No Cursor rules (`.cursor/rules/`, `.cursorrules`) or Copilot instructions
(`.github/copilot-instructions.md`) exist in this repository. If added later, incorporate them here.
