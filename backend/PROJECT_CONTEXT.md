# Finos Backend — Conversation Handoff

## Project overview

The user is building a lightweight backend for an internal application used by fewer than 50 people.

Technology stack:

- Go
- PostgreSQL
- Docker and Docker Compose
- Standard Go `net/http` router
- `pgx/v5` PostgreSQL driver
- JWT authentication using `github.com/golang-jwt/jwt/v5`
- bcrypt password hashing using `golang.org/x/crypto/bcrypt`

Backend directory:

```text
/home/sean-nugroho/Documents/Semester9/Finos/backend
```

The Git repository is in the parent Finos directory.

## Current project structure

```text
backend/
├── cmd/
│   └── api/
│       └── main.go
├── internal/
│   ├── auth/
│   │   ├── context.go
│   │   ├── middleware.go
│   │   ├── middleware_test.go
│   │   └── token.go
│   ├── config/
│   │   └── config.go
│   ├── database/
│   │   └── postgres.go
│   ├── helper/
│   │   └── helper.go
│   ├── reimbursement/
│   ├── server/
│   │   ├── middleware.go
│   │   └── router.go
│   ├── timesheet/
│   │   ├── handler.go
│   │   ├── handler_test.go
│   │   ├── model.go
│   │   ├── repository.go
│   │   ├── service.go
│   │   └── service_test.go
│   └── user/
│       ├── handler.go
│       ├── model.go
│       ├── repository.go
│       └── service.go
├── migrations/
│   ├── 000001_init.down.sql
│   └── 000001_init.up.sql
├── compose.yaml
├── Dockerfile
├── go.mod
└── go.sum
```

## Database schema

The original migration uses automatically generated `BIGINT` primary keys:

```sql
CREATE TABLE users (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'employee'
        CHECK (role IN ('employee', 'manager')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

Timesheets:

```sql
CREATE TABLE timesheets (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    work_date DATE NOT NULL,
    hours_worked INTEGER NOT NULL
        CHECK (hours_worked > 0 AND hours_worked <= 24),
    description TEXT,
    status TEXT NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'submitted', 'approved', 'rejected')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

Reimbursements exist in the migration and contain a `receipt_url` rather than storing image blobs.

### ID decision

The user experimented with random 16-character string IDs but encountered:

```text
cannot insert a non-DEFAULT value into column "id"
Column "id" is an identity column defined as GENERATED ALWAYS
```

The application was returned to using PostgreSQL-generated `BIGINT` IDs. Current Go ID types are `int64`.

If string IDs are reconsidered, the database schema, foreign keys, repository methods, models, and JWT subject parsing must all change together. UUID is preferable to a custom 16-character scheme.

## Architecture

The code follows:

```text
HTTP handler
    ↓
Service
    ↓
Repository interface
    ↓
PostgreSQL repository
    ↓
pgx pool
```

Dependency construction happens in `cmd/api/main.go`:

```go
repository := user.NewRepository(databasePool)
service := user.NewService(repository)
handler := user.NewHandler(service, tokenManager)
```

Handlers are passed into `server.New(...)` and registered in `router.go`.

A handler method such as `userHandler.Register` is passed as a function value. Go's HTTP server later supplies `w http.ResponseWriter` and `r *http.Request` when a request arrives.

## Configuration

Environment variables include:

```bash
APP_ENV=development
HTTP_PORT=8080
DATABASE_URL=postgres://finos:finos_password@localhost:5432/finos?sslmode=disable
JWT_SECRET=<at-least-32-bytes>
JWT_TTL=8h
```

Inside Docker Compose, the database host is `db`, not `localhost`.

`JWT_SECRET` must contain at least 32 bytes. Example secret generation:

```bash
openssl rand -hex 32
```

The configuration loader validates `DATABASE_URL`, `HTTP_PORT`, `JWT_SECRET`, and `JWT_TTL`.

## Docker

After source changes, rebuild the API:

```bash
docker compose up -d --build api
```

This is insufficient:

```bash
docker compose restart api
```

Restarting does not rebuild the Go binary. A previous `404` on registration was caused by an old Docker image.

PostgreSQL is exposed on host port `5433`, while containers connect to port `5432`.

## Authentication

### Password hashing

Registration hashes passwords using bcrypt:

```go
bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
```

Login verifies passwords using:

```go
bcrypt.CompareHashAndPassword(
    []byte(foundUser.PasswordHash),
    []byte(input.Password),
)
```

The stored hash must be the first argument and the submitted plaintext password second. Do not hash the submitted password again and compare strings because bcrypt generates a new salt for every hash.

### JWT implementation

JWTs use HS256. The custom claims type embeds standard claims:

```go
type Claims struct {
    Role string `json:"role"`
    jwt.RegisteredClaims
}
```

Example token payload:

```json
{
  "role": "employee",
  "iss": "finos-api",
  "sub": "9",
  "iat": 1791016844,
  "exp": 1791045644
}
```

- `sub`: user ID as a string
- `role`: application-specific role
- `iss`: issuer
- `iat`: issued-at timestamp
- `exp`: expiry timestamp

JWTs are signed, not encrypted. Never include passwords, hashes, secrets, or sensitive data.

### Authentication middleware

The middleware:

1. Reads `Authorization`.
2. Requires `Bearer <token>`.
3. Verifies the JWT signature, issuer, and expiry.
4. Parses `sub` into an `int64` user ID.
5. Stores an authenticated `Principal` in the request context.

```go
type Principal struct {
    UserID int64
    Role   string
}
```

Handlers retrieve it with:

```go
principal, ok := auth.PrincipalFromContext(r.Context())
```

Protected handlers use `principal.UserID`, not a client-supplied `user_id`.

Currently, any valid authenticated user can create and list their own timesheets. The role is stored for future role-specific authorization.

### Postman authentication

Use Postman's `Bearer Token` authorization type, not `JWT Bearer`.

Postman sends:

```http
Authorization: Bearer <access-token>
```

Clients must never receive the backend's `JWT_SECRET`.

Login post-response script:

```javascript
const response = pm.response.json();
pm.environment.set("access_token", response.access_token);
```

Protected requests can use `{{access_token}}`.

## Implemented API endpoints

### Health check

```http
GET /health
```

Checks API and PostgreSQL connectivity.

### Register

```http
POST /api/v1/auth/register
Content-Type: application/json
```

Request:

```json
{
  "name": "Sean Nugroho",
  "email": "sean@example.com",
  "password": "strong-password-123"
}
```

Public registrations receive the `employee` role. Registration also logs the user in by returning a JWT:

```json
{
  "access_token": "<jwt>",
  "token_type": "Bearer",
  "expires_in": 28799,
  "user": {
    "id": 9,
    "name": "Sean Nugroho",
    "email": "sean@example.com",
    "role": "employee"
  }
}
```

Behavior:

- Returns `201 Created`.
- Rejects unknown JSON fields.
- Limits the request body to 64 KiB.
- Returns `409 Conflict` for duplicate email.
- Never returns the password or hash.
- Adds `Cache-Control: no-store`.

### Login

```http
POST /api/v1/auth/login
Content-Type: application/json
```

Request:

```json
{
  "email": "sean@example.com",
  "password": "strong-password-123"
}
```

The service normalizes the email and verifies the bcrypt hash. A missing `return` after login errors was fixed so invalid login does not continue into token generation.

For security, nonexistent emails and incorrect passwords should return the same public error:

```json
{
  "error": {
    "code": "invalid_credentials",
    "message": "Invalid email or password"
  }
}
```

### Create timesheet

```http
POST /api/v1/timesheets
Authorization: Bearer <jwt>
Content-Type: application/json
```

Request:

```json
{
  "work_date": "2026-10-03",
  "hours_worked": 8,
  "description": "Worked on the authentication API"
}
```

The client must not supply `user_id` or `status`.

Behavior:

- `user_id` comes from JWT `sub`.
- Status is forced to `draft`.
- Work date must use `YYYY-MM-DD`.
- Hours must be between 1 and 24.
- Description is optional.
- Blank descriptions become SQL `NULL`.
- Client-supplied `user_id` is rejected.

Example response:

```json
{
  "id": 1,
  "user_id": 9,
  "work_date": "2026-10-03",
  "hours_worked": 8,
  "description": "Worked on the authentication API",
  "status": "draft",
  "created_at": "2026-10-03T04:00:00Z",
  "updated_at": "2026-10-03T04:00:00Z"
}
```

### List authenticated user's timesheets

```http
GET /api/v1/timesheets
Authorization: Bearer <jwt>
```

No body is required. The repository filters using the JWT user ID:

```sql
SELECT id, user_id, work_date, hours_worked, description, status, created_at, updated_at
FROM timesheets
WHERE user_id = $1
ORDER BY work_date DESC, id DESC
```

Example response:

```json
{
  "timesheets": [
    {
      "id": 2,
      "user_id": 9,
      "work_date": "2026-10-03",
      "hours_worked": 8,
      "description": "Worked on the authentication API",
      "status": "draft",
      "created_at": "2026-10-03T04:00:00Z",
      "updated_at": "2026-10-03T04:00:00Z"
    }
  ]
}
```

When no records exist:

```json
{
  "timesheets": []
}
```

## Timesheet model

```go
type Status string

const (
    StatusDraft     Status = "draft"
    StatusSubmitted Status = "submitted"
    StatusApproved  Status = "approved"
    StatusRejected  Status = "rejected"
)

type Timesheet struct {
    ID          int64
    UserID      int64
    WorkDate    time.Time
    HoursWorked int32
    Description *string
    Status      Status
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

Type mappings:

- PostgreSQL `BIGINT` → Go `int64`
- PostgreSQL `INTEGER` → Go `int32`
- PostgreSQL `DATE` → Go `time.Time`
- Nullable `TEXT` → Go `*string`
- PostgreSQL `TIMESTAMPTZ` → Go `time.Time`
- Restricted status values → custom `Status` type

## Logout decision

No backend logout endpoint currently exists. With stateless bearer JWTs, the first version can log out by deleting the frontend token.

A backend logout endpoint becomes useful with:

- `HttpOnly` cookies
- Refresh tokens
- Server-side sessions
- Token revocation or denylisting

The token remains valid until expiry even after the frontend deletes it.

## Error behavior

- `400 Bad Request`: invalid JSON or validation failure
- `401 Unauthorized`: absent, malformed, invalid, or expired JWT
- `409 Conflict`: duplicate registration email
- `415 Unsupported Media Type`: request is not JSON
- `500 Internal Server Error`: unexpected internal/database error

Unexpected internal errors are logged server-side while clients receive a generic message.

## Verification performed

At the last implementation point, these checks passed:

```bash
GOCACHE=/tmp/finos-go-build go test ./...
GOCACHE=/tmp/finos-go-build go vet ./...
git diff --check
```

Tests cover:

- Valid JWT adding the correct principal to context.
- Missing and invalid JWT rejection.
- Timesheet creation using the JWT user ID.
- Rejection of a client-supplied `user_id`.
- Timesheet validation.
- Listing only the authenticated user's timesheets.
- Returning an empty non-nil timesheet list.

## Important implementation notes

- Preserve existing user changes; the worktree may be dirty.
- Use `apply_patch` for file edits.
- Rebuild the API Docker image after Go source changes.
- Do not place `JWT_SECRET` in frontend code or Postman JWT-generation settings.
- Do not trust `user_id`, `role`, or approval status from client JSON.
- Manager-specific endpoints should use the authenticated principal's role.
- The list endpoint has no pagination because the application is small.
- Receipt images should eventually be stored in object storage, with only a `receipt_url` stored in PostgreSQL.

## Likely next milestones

1. Get one timesheet by ID while enforcing ownership.
2. Update draft timesheets owned by the authenticated user.
3. Delete draft timesheets owned by the authenticated user.
4. Submit a draft timesheet.
5. Add manager-only list, approve, and reject endpoints.
6. Implement reimbursement creation and receipt upload.
7. Add refresh tokens or secure `HttpOnly` cookie authentication.
8. Add CORS configuration for the frontend.
9. Add pagination and filters when needed.
10. Add integration tests against PostgreSQL.
