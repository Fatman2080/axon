# OpenFi Admin (Vite + Vue3)

## Run
```bash
cd frontend-admin
npm install
npm run dev
```

Default URL: `http://localhost:9335`

When served by backend static hosting, entry URL is:
- `http://localhost:9333/admin/`

## Build
```bash
npm run build
```

## Environment
Optional:
- `VITE_API_URL` (default: empty string, API uses same-origin `/admin/api/*`)

## Default Admin Login
- email: `admin@clwafi.io`
- password: `admin123`

## Admin Features
- Strategy review (approve/reject)
- User tier management
- Agent status management
- Vault metrics editing
- Invite code management (single/batch creation, export unused codes)
- Agent account pool import (AES-256-GCM encrypted payload decryption on server)
