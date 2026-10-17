# AGENTS.md

## Scope

These instructions apply to the Go backend in this directory and all of its descendants. The Git repository root is the parent `Finos` directory and also contains a Next.js frontend. Do not modify frontend files unless the task explicitly includes the frontend.

Preserve unrelated working-tree changes. Inspect `git status --short` before editing and never discard user changes to make the tree clean.

## Project Purpose

Finos is a small internal workforce/finance application intended for fewer than 50 users. Keep solutions straightforward, inexpensive to operate, and easy to understand. Prefer a clear implementation over infrastructure or abstractions that are unnecessary at this scale.

## Technology

- Go 1.26
- Standard-library `net/http` routing
- PostgreSQL 17
- `github.com/jackc/pgx/v5`
- `github.com/golang-jwt/jwt/v5`
- `golang.org/x/crypto/bcrypt`
- Docker and Docker Compose
- OpenAPI 3.0.3 with an embedded Swagger UI

Avoid adding dependencies when the standard library or an existing dependency is sufficient.

## Important Paths

```text
cmd/api/main.go                 Application composition and HTTP server
internal/auth/                  JWTs, authenticated principal, middleware
internal/config/                Environment loading and validation
internal/database/              PostgreSQL pool creation
internal/helper/                Shared HTTP helpers
internal/server/router.go       Route registration
internal/server/middleware.go   Request logging
internal/server/openapi.yaml    API contract
internal/server/docs.go         Embedded OpenAPI and Swagger UI
internal/user/                  User repository, service, handlers, models
internal/timesheet/             Timesheet repository, service, handlers, models
internal/reimbursement/         Reimbursement feature area
migrations/                     Forward and rollback SQL migrations
```

## Architecture

Use the existing dependency flow:

```text
HTTP handler → service → repository interface → PostgreSQL repository
```

- Handlers own HTTP concerns: headers, decoding, status codes, and response JSON.
- Services own validation, authorization rules, and business behavior.
- Repositories own SQL and database-error translation.
- Construct dependencies explicitly in `cmd/api/main.go` and pass handlers to `server.New`.
- Define small interfaces at the consumer boundary so services and handlers can be tested without PostgreSQL.
- Use context from the incoming request for all service and repository calls.

Keep domain/database models separate from request and response models when exposing the domain model could leak private fields.

## HTTP Conventions

- Use Go method-pattern routes such as `"POST /api/v1/timesheets"`.
- API routes use the `/api/v1` prefix; `/health`, `/docs/`, and `/openapi.yaml` are exceptions.
- Accept JSON only when a body is required. Validate `Content-Type` with `mime.ParseMediaType`.
- Limit request bodies, reject unknown JSON fields, and reject multiple JSON values.
- Return JSON errors in this shape:

```json
{
  "error": {
    "code": "validation_error",
    "message": "Human-readable message"
  }
}
```

- Do not expose database errors, password hashes, secrets, or stack traces.
- Use `Cache-Control: no-store` for responses containing credentials or sensitive account data.
- Return empty arrays as `[]`, not `null`.
- Update `internal/server/openapi.yaml` whenever an endpoint, request, response, status code, or authorization rule changes.

## Authentication and Authorization

Clients send access tokens as:

```http
Authorization: Bearer <JWT>
```

JWTs use HS256 and contain the user ID in `sub` and the role in the custom `role` claim. They are signed, not encrypted; never place sensitive data in claims.

Protected routes must use `auth.Authenticate`. Retrieve identity with:

```go
principal, ok := auth.PrincipalFromContext(r.Context())
```

Security rules:

- Never accept `user_id`, role, ownership, or approval authority from client JSON when it can be derived from the authenticated principal.
- Public registration always creates an employee; clients cannot register as managers.
- Enforce manager-only operations on the backend even if the frontend hides the controls.
- Do not give clients `JWT_SECRET` or `DATABASE_URL`.
- Hash passwords with bcrypt and verify with the stored hash first:

```go
bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(plainPassword))
```

- Return one generic invalid-credentials response for unknown email and wrong password when modifying login behavior.

## Database and Migrations

PostgreSQL owns primary keys using `BIGINT GENERATED ALWAYS AS IDENTITY`. Corresponding Go IDs are `int64`. Do not switch IDs to strings in application code without a migration covering every primary key, foreign key, model, repository method, and JWT subject conversion.

Type conventions:

- PostgreSQL `BIGINT` → Go `int64`
- PostgreSQL `INTEGER` → Go `int32`
- PostgreSQL `DATE`/`TIMESTAMPTZ` → Go `time.Time`
- Nullable `TEXT` → Go `*string` or an explicit nullable type
- Restricted database values → named Go string types with constants

Migration rules:

- Add a new numbered migration instead of rewriting a migration that may already have run.
- Supply both `.up.sql` and `.down.sql` files.
- Consider existing rows before adding `NOT NULL` columns.
- Preserve foreign-key behavior and add indexes for frequently filtered foreign keys/status/date columns.
- Do not delete Docker volumes or reset the database unless the user explicitly authorizes data loss.

Receipt images belong in object storage. Store only their URL or object key in PostgreSQL; do not store image blobs.

## Go Style

- Run `gofmt` on every edited Go file.
- Keep names idiomatic: exported identifiers use comments when their purpose is not obvious; local variables begin lowercase.
- Wrap errors with context and `%w` when callers need `errors.Is`/`errors.As` behavior.
- Use sentinel errors for business outcomes that handlers map to HTTP responses.
- Log unexpected internal errors server-side and return a generic client response.
- Avoid `log.Fatal` below `main`.
- Keep SQL parameter counts and argument order visibly aligned.
- Normalize user email addresses at repository/service boundaries consistently.

## Testing and Verification

Add focused tests whenever behavior changes. Prefer interface-backed stubs and `httptest` for unit/handler tests. Security-sensitive changes must test both allowed and denied cases.

Before handing off a change, run:

```bash
gofmt -w <edited-go-files>
GOCACHE=/tmp/finos-go-build go test ./...
GOCACHE=/tmp/finos-go-build go vet ./...
GOCACHE=/tmp/finos-go-build go build ./cmd/api
git diff --check
```

The sandbox may emit a non-fatal Go module stat-cache warning for the read-only home-directory cache. Redirect `GOCACHE` as shown; do not treat the warning as a test failure if the command exits successfully.

For OpenAPI edits, also parse or validate `internal/server/openapi.yaml`. Existing tests verify that the embedded document and Swagger HTML are served.

## Running Locally

Required environment values:

```bash
APP_ENV=development
HTTP_PORT=8080
DATABASE_URL=postgres://finos:finos_password@localhost:5433/finos?sslmode=disable
JWT_SECRET=<at-least-32-bytes>
JWT_TTL=8h
```

Generate a development secret with:

```bash
openssl rand -hex 32
```

When Go runs on the host, PostgreSQL from Compose is available on port `5433`. Inside Compose, the database hostname is `db` and its port is `5432`.

After backend source changes, rebuild the image:

```bash
docker compose up -d --build api
```

`docker compose restart api` only restarts the old compiled binary.

Documentation is available at:

```text
http://localhost:8080/docs/
http://localhost:8080/openapi.yaml
```

The interactive Swagger page currently loads its UI assets from jsDelivr; the OpenAPI YAML itself is embedded in the Go binary.

## Current Behavioral Notes

- Registration logs the new user in and returns a bearer token.
- Timesheet creation derives `user_id` from the JWT and forces `draft` status.
- Timesheet listing returns an employee's own records and all records for managers.
- The approval route currently toggles `approved` records back to `draft` and approves all other statuses. Preserve or deliberately redesign this behavior; do not change it accidentally.
- The manager user-list endpoint currently returns summaries rather than password-bearing database models.
- Reimbursement handlers/services/repositories may be incomplete; inspect them before assuming an API exists.

## Change Discipline

- Inspect relevant files before editing; the user may have changed code since the prior task.
- Keep edits scoped to the requested feature.
- Do not reformat or rewrite unrelated files.
- Never use destructive Git commands to discard work.
- Report the exact files changed and the verification performed.
