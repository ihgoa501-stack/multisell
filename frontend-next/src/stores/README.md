# State Management Boundaries

| Tool | Purpose | Example |
|------|---------|---------|
| **Zustand** | Synchronous UI state — sidebar, panel, modal, command palette | `app-store.ts` (sidebar collapsed) |
| **React Query** | Server state — all API data (lists, details, mutations) | `useQuery`, `useMutation` |
| **useState** | Component-local ephemeral state — form inputs | local-only |

## Rules

1. **Zustand for UI state only.** If you find API calls in a Zustand store, migrate it to React Query.
2. **React Query for all server data.** Use the `query-client.ts` instance. Cache invalidation is handled by query keys.
3. **No manual fetch in components.** Always use `apiClient` from `lib/api-client.ts` — not raw `fetch`.

## Exceptions

- `auth-store.ts` — needs synchronous token access for route guards
- `permission-store.ts` — needs synchronous code list for menu rendering

These are read-only caches of auth state, not server data stores. They get updated by the auth/refresh flow automatically.
