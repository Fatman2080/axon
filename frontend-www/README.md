# OpenFi WWW (Vite + React)

## Run
```bash
cd frontend-www
npm install
npm run dev
```

Default URL: `http://localhost:9334`

## Build
```bash
npm run build
```

## API Strategy
- Default API base is same-origin `/api/*`
- X OAuth start URL is `/api/auth/x/start`

Recommended dev access:
- Open backend URL `http://localhost:9333/`
- Let backend proxy static requests to Vite dev servers (`frontend.mode = "dev"`)
