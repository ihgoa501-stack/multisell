# LingMirror Frontend (Active)

This is the active LingMirror frontend.

- Stack: Next.js / React / TypeScript / Ant Design
- App Router: `src/app/`
- API client: `src/lib/api-client.ts`
- Default API base: `http://localhost:8080/api`

## Development

```bash
npm install
npm run dev -- --hostname 127.0.0.1 --port 3000
```

Open http://localhost:3000.

## Verification

```bash
npm run build
npm run lint
```

## Environment

```bash
NEXT_PUBLIC_API_URL=http://localhost:8080/api
```

Frontend API calls should use `/v1/*` paths through `apiClient`, which produces `/api/v1/*` with the default base URL.

## Legacy Boundary

The old Vue frontend in `../frontend/` is paused. Use it only as a behavior reference for parity, migration, or emergency rollback support.
