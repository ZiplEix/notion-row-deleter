# Notion Row Deleter (Web UI)

An internal web tool to archive (soft-delete) all pages of a Notion database with progress tracking. Designed for enterprise IT to review, deploy, and operate safely.

If you just want to use the program, you can download it by clicking on the `<> Code` button and selecting "Download ZIP". Then, you can execute the programme `notion-row-deleter-main/dist/notion-row-deleter-windows-amd64.exe`

## Overview
- Purpose: Archive all pages in a Notion Database via Notion API.
- UI: Simple web form to enter Notion API token and Database ID, plus a live progress page.
- Rate-limiting: ~3 requests/sec globally (shared across workers).
- Concurrency: Worker pool (size = CPU cores, minimum 4) to speed up processing while respecting the global rate limit.
- Real-time status: WebSocket stream (deleted/total/ETA).

## How it works
1. User opens the home page (“/”) and submits a Notion API token + database ID.
2. The app fully paginates the database to count pages (displays the total), then starts archiving.
3. A worker pool archives pages while a global ticker enforces ~3 req/s across all workers.
4. After each archive, the server broadcasts progress via WebSocket to the “/deleting” page.

Notes:
- Archiving is done via Notion PATCH /v1/pages/{id} {"archived": true}.
- The app sends a progress update after every archived page.

## Project structure
- `main.go`: App entrypoint, HTTP routes, template loading.
- `handler.go`: Handlers for `/`, `/delete` (POST), and `/deleting`.
- `ws.go`: WebSocket hub (register, unregister, broadcast Progress).
- `work.go`: Pagination, worker pool, rate limiter, and progress emission.
- `notion.go`: Minimal Notion API client for query + archive operations.
- `models.go`: Request/response types for Notion API.
- `templates/`: HTML templates (embedded via `go:embed`).
- `Makefile`: Build automation (Linux/Windows, amd64/arm64).

## Endpoints
- `GET /` – Home page with form (token and database ID). Starts nothing by itself.
- `POST /delete` – Starts archiving in background and redirects to `/deleting`.
- `GET /deleting` – Progress dashboard (WebSocket-driven).
- `GET /ws` – WebSocket endpoint broadcasting progress JSON:
  - `{ running: bool, deleted: number, total: number, etaSeconds: number }`

## Build and artifacts
Requirements: Go toolchain (project targets Go 1.24; Go ≥1.21 should work).

- Build for host platform:
  - `make build` → `dist/notion-row-deleter`
- Cross-compile:
  - `make build-linux` (amd64/arm64)
  - `make build-windows` (amd64/arm64)
  - `make build-all` (Linux+Windows, amd64/arm64)

## Run
Default listen address: `:8080` (HTTP).

- Development:
  - `make run`, then open `http://localhost:8080`
- Operation:
  - Place behind an HTTPS reverse proxy (TLS termination), e.g., NGINX/Apache/Traefik.
  - Restrict access to the web UI (e.g., VPN, SSO, IP allow-list, or basic auth at proxy).

## Configuration
- No server-side config is required for normal use.
- Credentials are entered in the UI (Notion API token + Database ID).
- The application does not persist tokens or database IDs. They live in process memory for the duration of the run only.
- A local `.env` file is not read by the application; it may be present for development convenience only.

## Security considerations
- Deploy behind HTTPS (via reverse proxy). Do not expose the HTTP endpoint directly to the internet.
- Limit access: ensure only authorized admins can reach the UI.
- Least privilege: use a Notion Integration token scoped only to the target database.
- Tokens are not logged or stored; avoid sharing screenshots/logs that might display sensitive IDs.
- No external storage/database is used; all state is in-memory.

## Networking
- Outbound: connects to `api.notion.com` (HTTPS) to query and archive pages.
- Inbound: listens on TCP 8080 (HTTP). Place behind a proxy for TLS.

## Observability
- Logs: sent to stdout/stderr (e.g., “Archived page: …”, totals). Integrate with your process supervisor’s logging (systemd, container logs, etc.).
- Live progress: the `/deleting` page shows deleted/total/remaining and ETA, updating after each archived page via WebSocket.

## Operational behavior
- Only one deletion run at a time (simple atomic lock). A second start attempt will be rejected.
- Global rate limit: ~3 req/s using a shared ticker (350ms cadence), regardless of worker count.
- Worker pool: parallelizes requests up to CPU core count (minimum 4) but still bounded by the global limiter.
- Failure mode: on first Notion API error, the run stops and a final progress state is emitted. Automatic retries/backoff are not implemented.
- Memory note: the app preloads all page IDs to compute the total and ETA. Extremely large databases may increase memory usage.

## Deployment example (systemd)
Example unit file (adjust paths/user as needed):
```
[Unit]
Description=Notion Row Deleter
After=network.target

[Service]
ExecStart=/opt/notion-row-deleter/notion-row-deleter
WorkingDirectory=/opt/notion-row-deleter
User=notion
Group=notion
Restart=on-failure
Environment=GOTRACEBACK=none

[Install]
WantedBy=multi-user.target
```
Place behind an HTTPS reverse proxy and restrict access (SSO/VPN/ACL).

## Verification checklist (IT)
- Build from source with `make build` or use a signed binary from your internal CI.
- Run in a restricted environment; confirm the UI is reachable only by admins.
- Use a test Notion database and token with minimal permissions.
- Start a run and watch `/deleting` for live updates (deleted/total/ETA).
- Observe logs: ensure steady cadence (≤3 req/s overall) and no unexpected errors.

## Troubleshooting
- No progress updates in UI:
  - Check browser DevTools → Network → WS; ensure the `/ws` connection is established and messages arrive.
  - Confirm reverse proxy allows WebSocket upgrade (Connection/Upgrade headers).
- 429 or 5xx from Notion:
  - The app stops on first error. Re-run later; consider adding a backoff layer if needed.
- Nothing happens after submit:
  - Ensure valid token and database ID; check server logs for errors.
