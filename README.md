# OpenFi Server (Go + Echo + SQLite)

## Directory Layout
- `src/` server source code
- `config/` server config files (`config.json`, `config.dev.json`)
- `dist/` compiled binary output
- `local_run/` one-click local run directory created by `dev_run.sh`

## Build
```bash
cd server
./build.sh
```
Output binary: `server/dist/openfi-server`

## Run
```bash
cd server
go run ./src -config ./config/config.json
```

## Dev One-Click Run
```bash
cd server
./dev_run.sh
```
This script will:
1. Compile server to `dist/`
2. Copy binary + config to `local_run/`
3. Start from `local_run/config/config.json` (dev mode)

## Config
No environment variable is required.  
All runtime parameters are read from `config/*.json`.

Key fields:
- `appBaseUrl` (full URL for www homepage, e.g. `https://app.example.com`)
- `server.port`
- `server.tokenSecret`
- `storage.dbPath`
- `agentPool.fixedKey` (32-byte UTF-8 string or 64-char hex)
- `hyperliquid.baseURL`
- `xOAuth.clientId`, `xOAuth.clientSecret`, `xOAuth.scopes`, `xOAuth.authorizeUrl`, `xOAuth.tokenUrl`, `xOAuth.userInfoUrl`
- `frontend.mode` (`release` / `dev`)
- `frontend.release.wwwDistDir`, `frontend.release.adminDistDir`
- `frontend.dev.wwwDevServer`, `frontend.dev.adminDevServer`

## Unified Path Strategy
- `GET /admin/api/*` admin backend API
- `GET /admin/*` admin frontend static assets
- `GET /api/*` www backend API
- `GET /*` www frontend static assets

## Static Resource Hosting Modes
- `frontend.mode = "release"`  
  Serve static files from configured dist directories.
- `frontend.mode = "dev"`  
  Reverse proxy static requests to Vite dev servers:
  - `/admin/*` -> admin dev server
  - `/*` -> www dev server

## X OAuth
Backend endpoints:
- `GET /api/auth/x/start`
- `GET /api/auth/x/callback`

OAuth callback URL in X app should point to:
- `<appBaseUrl>/api/auth/x/callback`

`xOAuth.redirectUrl` and frontend callback URLs are generated from `appBaseUrl` automatically.

## Admin Auth & Management
- On startup, server checks `admins` table; if empty, it auto-creates a default admin and logs a temporary password.
- Admin auth:
  - `POST /admin/api/login`
  - `GET /admin/api/me`
- Admin management:
  - `GET /admin/api/admins`
  - `POST /admin/api/admins`
  - `PATCH /admin/api/admins/:id/password`
  - `DELETE /admin/api/admins/:id`

## Agent Pool Import Payload
`POST /admin/api/agent-accounts/import` expects:
```json
{
  "encryptedPayload": "{\"status\":\"ok\",\"format\":\"AES-GCM-256\",\"encrypted_data\":\"<hex>\",\"count\":12}"
}
```

Plain JSON private-key arrays are **not accepted**.

2026/03/02 20:46:40 bootstrap admin created: email=admin@openfi.local temporary_password=VXDRuL3v4KM3RAWm (please change immediately)
