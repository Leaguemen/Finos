# Finos Handoff — 2026-10-17

## 1. GOAL

Build and ship a small internal finance/workforce application for fewer than 50 users. The Go/PostgreSQL backend should provide JWT authentication, employee timesheets, manager workflows, reimbursements with externally stored receipt images, Docker deployment, and Swagger documentation. Keep the architecture simple enough to learn and extend quickly.

## 2. REPO OVERVIEW

Git root:

```text
/home/sean-nugroho/Documents/Semester9/Finos
```

Backend:

```text
backend/
├── cmd/api/main.go              # composition root and HTTP server
├── internal/
│   ├── auth/                    # JWT generation, parsing, middleware, context principal
│   ├── config/                  # environment validation
│   ├── database/                # pgxpool connection
│   ├── helper/                  # JSON response helper
│   ├── server/                  # router, logging, Swagger/OpenAPI
│   ├── user/                    # models, repository, service, handlers
│   ├── timesheet/               # models, repository, service, handlers
│   └── reimbursement/           # currently stubs
├── migrations/
├── Dockerfile
├── compose.yaml
└── go.mod
```

Frontend:

```text
nextjs-dashboard/
```

Stack:

- Go 1.26
- `net/http` method-pattern routing
- PostgreSQL 17
- `pgx/v5`
- bcrypt via `golang.org/x/crypto`
- HS256 JWT via `github.com/golang-jwt/jwt/v5`
- Docker Compose
- Next.js frontend

Pattern:

```text
handler → service → repository interface → PostgreSQL repository
```

Dependencies are assembled in `cmd/api/main.go`. Handlers receive authenticated identity through `auth.PrincipalFromContext`.

Commands:

```bash
cd /home/sean-nugroho/Documents/Semester9/Finos/backend

GOCACHE=/tmp/finos-go-build go test ./...
GOCACHE=/tmp/finos-go-build go vet ./...
GOCACHE=/tmp/finos-go-build go build ./cmd/api

docker compose up -d --build api
docker compose logs -f api
```

After Go changes, `docker compose restart` is insufficient; rebuild with `--build`.

## 3. CURRENT STATE

Last commit:

```text
c56c754 feat: managerial
```

Implemented routes:

```text
GET   /health
POST  /api/v1/auth/register
POST  /api/v1/auth/login
GET   /api/v1/users                         # manager-only; currently ID/name only
PATCH /api/v1/users/{userID}/promote        # manager-only
POST  /api/v1/timesheets                    # authenticated user ID from JWT
GET   /api/v1/timesheets                    # employee: own; manager: all
PATCH /api/v1/timesheets/{timesheetID}/approve # manager-only toggle
GET   /docs
GET   /docs/
GET   /openapi.yaml
```

Verified on 2026-10-17:

```text
go test ./...       PASS
go vet ./...        PASS
go build ./cmd/api  PASS
OpenAPI YAML parse  PASS
git diff --check    PASS
```

Non-fatal sandbox warning during build:

```text
go: writing stat cache: mkdir /home/sean-nugroho/go/pkg/mod/cache/download/finos.com: read-only file system
```

Exact dirty files from `git status --short --untracked-files=all`:

- `backend/internal/server/router.go` — registers `/docs`, `/docs/`, and `/openapi.yaml`.
- `backend/internal/server/docs.go` — embeds OpenAPI YAML and serves Swagger UI.
- `backend/internal/server/docs_test.go` — verifies raw spec and Swagger HTML handlers.
- `backend/internal/server/openapi.yaml` — OpenAPI 3.0.3 definition for all current routes.
- `nextjs-dashboard/README.md` — documents manager employee-name lookup.
- `nextjs-dashboard/app/components/workspace.tsx` — loads manager user summaries and displays employee names.
- `nextjs-dashboard/app/lib/timesheets.ts` — adds `Employee` type.
- `nextjs-dashboard/app/ui/global.css` — centers approval column/actions.
- `nextjs-dashboard/app/api/users/route.ts` — untracked Next.js proxy for backend user listing.

Tracked diff stat excludes untracked files:

```text
5 files changed, 45 insertions(+), 12 deletions(-)
```

Swagger is code-complete but uncommitted and not live-tested in Docker during the latest turn. Swagger UI uses jsDelivr assets; raw OpenAPI is embedded in the Go binary.

## 4. KEY DECISIONS

- PostgreSQL owns IDs with `BIGINT GENERATED ALWAYS AS IDENTITY`. A custom random 16-character ID attempt was rejected because code and schema diverged; changing IDs would require converting all foreign keys and JWT parsing. UUID would be preferable if opaque IDs become necessary.
- JWT contains `sub` (user ID), `role`, `iss`, `iat`, and `exp`. JWT is signed, not encrypted.
- Clients send `Authorization: Bearer <token>`. In Postman use **Bearer Token**, not **JWT Bearer**; clients must never know `JWT_SECRET`.
- Protected handlers never accept `user_id` from JSON. Middleware verifies JWT and stores `{UserID, Role}` in context.
- Registration always creates `employee` users and immediately returns a JWT.
- Passwords use bcrypt; verification is `CompareHashAndPassword(storedHash, plaintext)`.
- Timesheets are created as `draft`. Employees list their records; managers list all records.
- The approval endpoint currently toggles `approved ↔ draft`; it is not a one-way approval action.
- User list and promotion authorization is checked against the JWT role.
- Receipt images should live in object storage; PostgreSQL stores only `receipt_url`.
- OpenAPI is hand-maintained and embedded with `go:embed`, avoiding a Swagger generation dependency.
- Nginx/reverse proxy was deferred; a hosting platform can terminate HTTPS initially.

## 5. OPEN PROBLEMS

- The request “get all user data if you’re a manager” was interrupted. `GET /api/v1/users` exists but returns only `{id, name}` through `user.Summary`, not email, role, or timestamps. Password hashes must never be returned.
- Reimbursement Go files are empty package stubs; no reimbursement endpoint exists.
- Migration `000002_add_timesheet_id_to_reimbursements.up.sql` adds a `NOT NULL` foreign key. Whether it has been applied and whether existing reimbursement rows make it fail is **UNVERIFIED**.
- Swagger UI runtime behavior inside the rebuilt container is **UNVERIFIED**.
- Swagger UI depends on browser access to jsDelivr.
- `compose.yaml` contains `sslmode=disable\}` in the default `DATABASE_URL`; whether the backslash is intentional/correct is **UNVERIFIED**.
- `.env.example` contains `CORS_ALLOWED_ORIGIN`, but Go has no CORS middleware and does not read it.
- Login returns `201 Created`, not conventional `200 OK`.
- Login exposes different messages for nonexistent user versus wrong password, enabling account enumeration.
- JWT roles remain stale until token expiry after role changes.
- No pagination; acceptable for current scale but not future-proof.
- No failing tests currently.

## 6. NEXT STEPS

1. Preserve all dirty frontend work; rebuild API and smoke-test:
   - `/docs/`
   - `/openapi.yaml`
   - Swagger “Authorize” and a protected request.
2. Commit the Swagger files and router change if runtime verification succeeds.
3. Complete manager user listing with safe fields: `id`, `name`, `email`, `role`, `created_at`, `updated_at`; exclude `password_hash`. Update repository, service, handler response, tests, frontend type, and OpenAPI together.
4. Normalize login failures to one `401 invalid_credentials` response and consider changing successful login to `200`.
5. Verify/fix the Compose `DATABASE_URL` escape.
6. Decide migration handling for existing reimbursements before applying migration 000002.
7. Implement reimbursement creation, object-storage receipt upload, listing, and manager approval.
8. Add integration tests against PostgreSQL and deployment smoke tests.

## 7. CONSTRAINTS & PREFERENCES

- Ship quickly; keep infrastructure inexpensive and suitable for fewer than 50 users.
- Preserve unrelated dirty work, especially frontend changes.
- PostgreSQL stores receipt URLs, not image blobs.
- Use JWT authentication.
- Prefer clear repository/service/handler separation.
- Public registration must not allow choosing manager role.
- Never trust client-provided user IDs, roles, or approval status.
- Do not expose password hashes, JWT secret, or database credentials.
- Git repository is one directory above `backend`.
- Use `apply_patch` for source edits.
- Avoid destructive database-volume deletion unless explicitly approved.
- Rebuild Docker after backend source changes.

## 8. IMPORTANT SNIPPETS

Required environment:

```bash
APP_ENV=development
HTTP_PORT=8080
DATABASE_URL=postgres://finos:finos_password@db:5432/finos?sslmode=disable
JWT_SECRET=<at-least-32-bytes>
JWT_TTL=8h
```

Generate a development secret:

```bash
openssl rand -hex 32
```

Authenticated request:

```http
Authorization: Bearer <access_token>
```

Create timesheet body:

```json
{
  "work_date": "2026-10-17",
  "hours_worked": 8,
  "description": "Worked on the Finos API"
}
```

Swagger URLs:

```text
http://localhost:8080/docs/
http://localhost:8080/openapi.yaml
```

Current manager user-list database query:

```sql
SELECT id, name
FROM users
ORDER BY name ASC, id ASC;
```

This query/model is the first place to expand when completing the pending “all safe user data” requirement.