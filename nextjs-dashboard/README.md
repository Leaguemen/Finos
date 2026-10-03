# Finos

Mobile-first Next.js frontend for the Finos Go API.

## Run

```sh
pnpm install
pnpm dev
```

Open http://localhost:3000. Start the Go API on port 8080. To use another API origin, set `FINOS_API_URL` in `.env.local` and restart Next.js. For testing from your phone, open the frontend using your computer's LAN address; API requests still originate from the Next.js server.

## Pages and reusable pieces

- `/login` and `/register`: shared `AuthShell`, `AuthForm`, `Field`, `Button`, and `Brand` components in `app/components`.
- `/dashboard`: a basic signed-in workspace with logout; `/` routes here.
- `app/ui/global.css`: shared gold, charcoal, gray, and white design tokens inspired by the supplied company-profile PDF.
- `app/lib/auth.ts`: typed API contract, session storage, and `bearerHeaders()` for future authenticated requests.
- `AuthProvider` / `AuthGate`: browser-session initialization, guest/protected page redirects, expiration, and tab synchronization.

The same-origin `POST /api/auth/login` and `POST /api/auth/register` route forwards to `/api/v1/auth/*` on the Go server, avoiding browser CORS requirements. Payloads exactly match `LoginInput` and `RegisterInput`. Both endpoints are expected to return `{ access_token, token_type: "Bearer", expires_in, user: { id, name, email, role } }`. Errors use `{ error: { code, message } }`.

The browser stores the bearer token, expiry, and user in localStorage under `finos.session`. No password is stored. This is a UI session check, not server-side authorization or token verification. Future protected backend endpoints must validate the Authorization bearer header. Logout removes the browser session; no token revocation endpoint exists yet.

## Validation

```sh
pnpm exec tsc --noEmit
pnpm build
```

Backend observation: `internal/user/handler.go` currently lacks `return` after `handler.writeLoginError(w, err)` in `Login`. Add it in the backend so failed logins stop before token generation and return one JSON error. The frontend treats malformed error responses as failed logins.
