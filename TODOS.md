# TODOs

## Metabolism M1 — Phase 1 Migration

- **What:** Migration for `metabolism_log` table + `event_outbox` indexed columns
- **Why:** M1 scores records and needs a table to store score results. `event_outbox` needs `excreted_at` (tagged for deletion) and `excretion_reason` (why it was scored/excreted) columns for scheduled cleanup in Phase 2.
- **Context:** Added during /plan-eng-review on 2026-06-26. Phase 1 is dry-run (no actual deletion), but the schema should ship from day 1 so Phase 2 doesn't need a second migration.
- **Action:** `backend-go/migrations/XXX_add_metabolism.sql` — CREATE TABLE metabolism_log + ALTER TABLE event_outbox ADD COLUMNS.
- **Depends on:** Design approval of MetabolismModel fields (see design doc).
- **Blocked by:** Nothing.
