Purpose

This file gives concise, repo-specific instructions for automated coding assistants (Copilot, bots) and for developers who want consistent edits.

What to do

- Work in Go (module-mode). Keep public APIs stable unless asked.
- Follow Go tooling: run `gofmt`/`go vet`/`go test ./...` before pushing.
- Prefer minimal, incremental changes and include tests for behavior changes.
- Keep commits small and focused; explain why in the commit message.

Project layout

- `main.go` — application entry
- `handlers/` — HTTP handlers (auth.go)
- `routes/` — route registration (auth.go)
- `database/`, `models/` — DB access and models
- `utils/` — helpers: crypto, jwt, response

Coding style & checks

- Use `gofmt` formatting and idiomatic Go patterns.
- Add or update unit tests in the same package when changing behavior.
- Run `go test ./...` and ensure all tests pass.
- Don't commit secrets; use environment variables (document expected env vars in README).

When you need clarification

- If a request lacks specifics (inputs, error handling, auth rules), ask the repo owner for the exact behavior.
- Avoid making breaking changes to public functions without approval.

Quick commands (PowerShell)

```pwsh
# build
# Copilot / Automated Assistant Instructions — Enterprise Grade

Purpose

This file tells automated assistants and contributors how to produce enterprise-grade code for this Go service. Be strict: code must be correct, safe, observable, tested, and reviewable.

Goals (what "enterprise-grade" means here)

- Correct: deterministic behavior, validated inputs, explicit error handling.
- Secure: no secrets in source, input validation, safe defaults, least privilege.
- Observable: structured logs, metrics, and (optional) tracing points.
- Testable: unit tests, table-driven tests, and at least one integration/smoke test for changed features.
- Maintainable: clear APIs, small focused commits, and documentation for public behavior.

Hard rules (always follow)

- Run `gofmt` and `go vet` on all changes.
- New or changed behavior must include unit tests covering happy path + at least one edge case.
- Use context.Context for request-scoped operations and timeouts.
- All exported functions must have clear doc comments and stable signatures unless a breaking change is explicitly requested and approved.
- Never commit secrets or credentials. Read secrets from env vars or a secrets manager; document required env vars.

Design & architecture expectations

- Prefer small, well-defined interfaces for external interactions (DB, email, storage). Accept interfaces in constructors for easier testing.
- Keep business logic out of HTTP handlers — handlers should validate input, call a service layer, and translate results to responses.
- Use DTOs (request/response structs) for public API shapes and explicit validation (e.g., validate fields and lengths).

Testing & QA

- Unit tests: table-driven, fast, deterministic. Use table cases for error conditions.
- Integration test: for any DB or external dependency change, add a smoke test that runs against an in-memory DB or a test container if available.
- Coverage: aim for meaningful coverage on critical paths; do not ship breaking behavior without tests.

Security

- Hash passwords with a strong KDF (bcrypt/argon2) from `utils/crypto.go`.
- Use short-lived access tokens and refresh token rotation where applicable. Validate and verify tokens server-side.
- Sanitize and validate all external input. Prefer explicit allow-lists to deny-lists.

Observability & errors

- Use structured logs (JSON or key/value) and include request id / trace id when available.
- Return consistent error responses using `utils/response.go` and standard HTTP statuses.
- Emit a metric or log on authentication failures and critical errors.

CI / Pre-commit expectations

- Before pushing: run `gofmt`, `go vet`, `go test ./...` and a linter (e.g., `staticcheck`) if present.
- Prefer small PRs with a descriptive summary and acceptance criteria; include screenshots / sample requests for API changes.

Commit messages

- Use a short title (<=50 chars), an empty line, and a detailed body describing why the change was made and any migration steps.

Review & acceptance criteria

- PR must include tests for new behavior.
- PR must pass CI linting and tests.
- At least one reviewer must approve; map reviewers to areas (auth, db, infra).

Quick commands (PowerShell)

```pwsh
# format
gofmt -w .
# vet
go vet ./...
# build
go build .
# run
go run .
# tests
go test ./... -v
```

Notes & contact

- Document required environment variables and secrets in `README.md` (e.g., JWT_SECRET, DB_DSN).
- When in doubt about design or backward-compatibility, ask the repo owner with a short design note and options.

This file is a living document — update it when process or tooling changes.
