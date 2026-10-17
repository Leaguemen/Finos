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
- `/dashboard`: your timesheets, with description search, status/date filters, date/hour sorting, and totals for visible rows; `/` routes here.
- **Export to Excel** on the dashboard downloads all timesheets matching the current search and filters, in the selected sort order, as an `.xlsx` file with work date, description, numeric hours, and status. Clear filters to export all your timesheets.
- `/timesheets/new`: create a draft timesheet with a work date, 1–24 whole hours, and an optional description.
- `/reimbursements`: an expense ledger with timesheet-description, category, status, receipt, employee, and date filters; date, amount, and category sorting; filtered totals; receipt viewing; and Excel export in the visible sort order.
- `/reimbursements/new`: create a draft reimbursement for one of the signed-in employee's timesheets, with an exact decimal amount and an optional JPEG/PNG receipt up to 5 MiB.
- `WorkspaceShell`, `CollectionControls`, and `app/lib/collection.ts`: reusable navigation, filter controls, and typed filtering/sorting.
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


Timesheets use the same-origin `GET /api/timesheets` and `POST /api/timesheets` bridge to the Go API. Both forward the current bearer token, bypass caching, and handle backend errors. A 401 clears the browser session. The list expects `{ timesheets: [...] }`; creation sends `{ work_date, hours_worked, description }`. Filtering and sorting happen locally over the current user’s returned records. Saving returns to the list and fetches it again. Edit, delete, and submission are not implemented.

Reimbursements use authenticated same-origin bridges for `GET` and multipart `POST` requests under `/api/timesheets/{id}/reimbursements`. The workspace loads compact IDs and descriptions from `/api/timesheets/options`, loads each accessible timesheet's reimbursement list, and combines the results before filtering, sorting, totaling, and exporting. User-facing lists, selection controls, and exports show the timesheet description instead of its ID. Receipt viewing uses `/api/reimbursements/{id}/receipt`; the Next.js route exchanges the authenticated backend redirect for the receipt without forwarding the Finos bearer token to storage.

Managers (JWT `role: "manager"`) see team timesheets, an employee-name filter, and approval controls. The manager view loads names from `GET /api/v1/users` and joins them to timesheets by `user_id`. Approval forwards `PATCH /api/timesheets/{id}/approve` to `/api/v1/timesheets/{id}/approve` with the bearer token; the returned row updates the list and filtered exports. Already-approved rows offer “Return to draft” through the same endpoint. Buttons are disabled only while a request is in progress. Decoding the JWT only controls UI visibility; the backend enforces manager authorization.
