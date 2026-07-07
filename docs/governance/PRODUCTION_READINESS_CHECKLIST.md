# Production Readiness Checklist

Last updated: 2026-07-06

This checklist must be completed for every feature, integration, or Agent workflow before it ships to production. Each item is PASS/FAIL/NA. Items marked "Owner must decide" require explicit Owner sign-off.

## 1. Configuration Checklist

| # | Item | PASS/FAIL/NA | Notes |
|---|------|-------------|-------|
| 1.1 | All environment-specific values externalized to `config.yaml` or env vars | | |
| 1.2 | Config loaded at startup, fails fast on missing required keys | | |
| 1.3 | No secrets in config.yaml (env vars or secret store only) | | |
| 1.4 | Config values have documented defaults and valid ranges | | |
| 1.5 | Feature flags exist for all gated capabilities | | |
| 1.6 | Config reload on SIGHUP or requires restart (documented) | | |

## 2. Secrets Management

| # | Item | PASS/FAIL/NA | Notes |
|---|------|-------------|-------|
| 2.1 | JWT secret rotated at least once per quarter | | |
| 2.2 | Platform API keys (Ozon, Shopee, Lazada) stored in env / secret store, not in code | | |
| 2.3 | Database credentials not hardcoded | | |
| 2.4 | LLM API keys (OpenAI, Anthropic) from env only | | |
| 2.5 | Webhook HMAC secrets unique per platform | | |
| 2.6 | No secrets in logs, error messages, or audit output | | |

## 3. Migrations Reviewed

| # | Item | PASS/FAIL/NA | Notes |
|---|------|-------------|-------|
| 3.1 | Migration file named with correct version (no duplicates -- see `backend-go/migrations/`) | | |
| 3.2 | Both `up.sql` and `down.sql` present | | |
| 3.3 | Down migration reverses up migration exactly | | |
| 3.4 | Migration does not lock large tables for extended periods (CREATE INDEX CONCURRENTLY for tables > 100k rows) | | |
| 3.5 | Migration tested against a copy of production data | | |
| 3.6 | No irreversible operations (DROP COLUMN, destructive UPDATE) without explicit Owner approval | | |

## 4. Backup / Restore Verified

| # | Item | PASS/FAIL/NA | Notes |
|---|------|-------------|-------|
| 4.1 | Full database backup taken before deployment | | |
| 4.2 | Backup verified restorable (restore drill < 30 min) | | |
| 4.3 | Backup stored in separate physical location / cloud bucket | | |
| 4.4 | Backup retention policy documented and enforced | | |
| 4.5 | Point-in-time recovery (WAL archiving) confirmed working | | |

## 5. Observability

### 5.1 EventBus

| # | Item | PASS/FAIL/NA | Notes |
|---|------|-------------|-------|
| 5.1.1 | Every event topic documented in `docs/governance/KERNEL_CONTRACTS.md` | | |
| 5.1.2 | Dead-letter queue (DLQ) configured for each subscription | | |
| 5.1.3 | Event processing latency metrics exported | | |
| 5.1.4 | Event processing error rate alert threshold set | | |
| 5.1.5 | Idempotency key mechanism verified for business-state mutations | | |

### 5.2 Scheduler

| # | Item | PASS/FAIL/NA | Notes |
|---|------|-------------|-------|
| 5.2.1 | Every scheduled agent listed in `router.go` and `docs/governance/KERNEL_CONTRACTS.md` | | |
| 5.2.2 | Agent heartbeat metric exported (last tick timestamp) | | |
| 5.2.3 | Missed-tick alert configured (no heartbeat for > 2x interval) | | |
| 5.2.4 | Concurrent execution guard confirmed (no overlapping ticks) | | |

### 5.3 Agent Observability

| # | Item | PASS/FAIL/NA | Notes |
|---|------|-------------|-------|
| 5.3.1 | Agent execution traced (OpenTelemetry span per agent run) | | |
| 5.3.2 | Agent decision output logged with correlation ID | | |
| 5.3.3 | Agent lifecycle state visible in cockpit dashboard | | |
| 5.3.4 | Agent failure metrics by agent ID | | |
| 5.3.5 | Agent stuck / hung detection (timeout per agent > threshold) | | |

### 5.4 LLM Cost & Failure Retry

| # | Item | PASS/FAIL/NA | Notes |
|---|------|-------------|-------|
| 5.4.1 | LLM token usage per agent logged and aggregated | | |
| 5.4.2 | LLM error rate (timeout, rate-limit, bad response) tracked | | |
| 5.4.3 | Retry mechanism with exponential backoff for transient LLM failures | | |
| 5.4.4 | Retry count and total backoff duration documented per agent | | |
| 5.4.5 | Circuit breaker on sustained LLM failure (e.g., 10 consecutive errors) | | |

### 5.5 Queue Depth

| # | Item | PASS/FAIL/NA | Notes |
|---|------|-------------|-------|
| 5.5.1 | Command Dispatcher queue depth exported as metric | | |
| 5.5.2 | Alert on queue depth > threshold for > 5 minutes | | |
| 5.5.3 | Scheduler backlog metric (pending ticks) | | |
| 5.5.4 | Platform write-back queue depth visible per platform | | |

## 6. Alerting

| # | Item | PASS/FAIL/NA | Notes |
|---|------|-------------|-------|
| 6.1 | All alerts defined in `docs/ops/ALERT_RULES.md` | | |
| 6.2 | Alert severity classified (P0/P1/P2/P3) | | |
| 6.3 | Alert destinations configured (PagerDuty, Slack, email) | | |
| 6.4 | On-call roster documented and reachable | | |
| 6.5 | Alert response SLOs defined per severity | | |
| 6.6 | Integration-free paths tested end-to-end (webhook, email) | | |

## 7. Rate Limiting

| # | Item | PASS/FAIL/NA | Notes |
|---|------|-------------|-------|
| 7.1 | Global rate limit configured per user/IP (`ratelimit.go`) | | |
| 7.2 | Per-endpoint rate limits documented (sensitive endpoints) | | |
| 7.3 | Platform API rate limit enforced (per-platform QPS) | | |
| 7.4 | Rate limit violations logged and alerted | | |
| 7.5 | Rate limit returned as standard HTTP 429 with Retry-After header | | |

## 8. Cost Control

| # | Item | PASS/FAIL/NA | Notes |
|---|------|-------------|-------|
| 8.1 | Single agent daily budget set and enforced | | |
| 8.2 | Single task token cap configured per agent | | |
| 8.3 | Retry cap per task (max retries before abort) | | |
| 8.4 | Auto-degrade on over-budget: agent drops to read-only / monitoring mode | | |
| 8.5 | Budget exceeded event published on EventBus | | |
| 8.6 | Dashboard shows daily/weekly/monthly cost per agent | | |

**Owner must decide:**

| Item | Decision | Date |
|------|----------|------|
| Daily budget per agent | | |
| Token cap per agent task | | |
| Max retries per task | | |
| Auto-degrade triggers (which thresholds) | | |

## 9. Kill Switch

| # | Item | PASS/FAIL/NA | Notes |
|---|------|-------------|-------|
| 9.1 | Global Agent kill switch (disable all automated write-back) | | |
| 9.2 | Per-Agent kill switch (disable specific Agent write operations) | | |
| 9.3 | Per-platform kill switch (disable write-back to Ozon, Shopee, etc.) | | |
| 9.4 | Kill switch accessible via cockpit dashboard (not only SSH) | | |
| 9.5 | Kill switch activation triggers notification to all on-call | | |
| 9.6 | Kill switch state visible to Owner (dashboard indicator) | | |

## 10. External Platform Safety

| # | Item | PASS/FAIL/NA | Notes |
|---|------|-------------|-------|
| 10.1 | Platform adapter tested against sandbox/developer environment | | |
| 10.2 | Platform API error handling: all expected error codes mapped to typed responses | | |
| 10.3 | Platform write-back rate limit respects platform's API rate caps | | |
| 10.4 | Platform credential expiry detection (alert when API key near expiry) | | |
| 10.5 | Platform webhook signature verification enforced | | |
| 10.6 | Platform outage detection (consecutive API failures trigger alert, not infinite retry) | | |

## 11. Production Trial Boundary

**Owner must decide:**

| Item | Value | Owner Decision | Date |
|------|-------|---------------|------|
| Which stores are trial (store IDs) | | | |
| Which platforms are active (Ozon/Shopee/Lazada) | | | |
| Which SKUs are writable | | | |
| Max daily actions (write-back operations) | | | |
| Which actions are dry-run only | | | |
| Trial duration | | | |
| Escalation path when boundary is hit | | | |

| # | Item | PASS/FAIL/NA | Notes |
|---|------|-------------|-------|
| 11.1 | Trial boundary enforced in code (not just convention) | | |
| 11.2 | Boundary metrics exported (actions taken, remaining daily budget) | | |
| 11.3 | Alert when approaching boundary (e.g., 80% of daily limit) | | |
| 11.4 | Hard stop when boundary exceeded (no more write-backs for the day) | | |
| 11.5 | Dry-run actions logged as if real but never sent to platform | | |
| 11.6 | Owner can adjust boundary without deployment (config/flag change) | | |
