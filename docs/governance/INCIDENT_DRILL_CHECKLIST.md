# Incident Drill Checklist and Manual Override

Last updated: 2026-07-06

## 1. Incident Severity Levels

| Level | Label | Definition | Response Time | Update Frequency | Escalation |
|-------|-------|-----------|---------------|------------------|------------|
| P0 | Critical | System unavailable, data loss, financial error, security breach | 15 min to acknowledge, 1 h to mitigate | Every 30 min | On-call → Tech Lead → Owner |
| P1 | High | Major feature broken, platform write-back failing, Agent pipeline stalled, incorrect pricing | 30 min to acknowledge, 4 h to mitigate | Every 1 h | On-call → Tech Lead |
| P2 | Medium | Minor feature degraded, single Agent failing, non-critical alert firing | 2 h to acknowledge, next business day to mitigate | Every 4 h | On-call |
| P3 | Low | Cosmetic issue, non-functional bug, informational alert | Next business day | Weekly | None |

## 2. Drill Scenarios

### 2.1 Database Failure

| Phase | Action | Expected Outcome |
|-------|--------|-----------------|
| **Detection** | Alert: `database.connection_pool.exhausted` or `database.query.timeout > 5s` | Incident opened, severity P1 (P0 if full outage) |
| **Containment** | 1. Check PgBouncer / connection pool stats<br>2. Check for long-running queries (`pg_stat_activity`)<br>3. Kill runaway queries (`pg_cancel_backend`)<br>4. If full outage: failover to read replica, write traffic blocked | Read traffic restored, writes queued |
| **Root cause** | 1. Check slow query log<br>2. Check disk space (`df -h`, WAL disk usage)<br>3. Check replication lag<br>4. Check for locks (`pg_locks`) | Root cause identified |
| **Recovery** | 1. If slow query: add index, optimize, or `pg_terminate_backend`<br>2. If disk: archive WAL, extend volume<br>3. If replication: fix network/reconnect<br>4. If corruption: restore from backup (see ROLLBACK_AND_RECOVERY.md) | Database healthy, writes restored |
| **Manual override** | 1. Kill switch: stop all Agent write-back to DB<br>2. Buffer writes to in-memory queue<br>3. If extended outage > 30 min: activate read-only mode for the whole platform | No data loss, manual flush queue when DB restored |

### 2.2 Platform API Outage

| Phase | Action | Expected Outcome |
|-------|--------|-----------------|
| **Detection** | Alert: `platform.{name}.api.error_rate > 5%` or consecutive HTTP 5xx / timeout from platform | Incident opened, severity P1 |
| **Containment** | 1. Stop all write-backs to the affected platform<br>2. Switch agent to dry-run mode for that platform<br>3. Queue pending write-backs (with retry limit) | No further failed writes |
| **Root cause** | 1. Check platform status page (Ozon/Shopee/Lazada status)<br>2. Check if platform API credentials expired<br>3. Check if platform IP allowlist changed<br>4. Check if request format changed (platform API update) | Root cause identified |
| **Recovery** | 1. If platform outage: wait for platform status "operational"<br>2. If credentials: rotate and re-test<br>3. If IP allowlist: update firewall<br>4. If API change: update adapter, deploy | Platform writes restored |
| **Manual override** | 1. Flush pending write-backs to local log instead of platform<br>2. Owner can approve manual platform write via admin panel<br>3. If outage > 4 h: activate alternative fulfillment path (manual processing) | No order loss, manual intervention |

### 2.3 Agent Runaway (Losely Control Loop)

| Phase | Action | Expected Outcome |
|-------|--------|-----------------|
| **Detection** | Alert: `agent.{id}.decision_rate > 5x baseline` or `agent.{id}.budget_exceeded` or `agent.{id}.stuck > 30 min` | Incident opened, severity P1 |
| **Containment** | 1. Global agent kill switch: stop all Agent write-back<br>2. Per-agent kill switch: stop the affected Agent<br>3. Block EventBus events to the Agent's subscription | Agent stopped, no further actions |
| **Root cause** | 1. Check Agent decision logs (what triggered the loop)<br>2. Check LLM response quality (degraded/empty/confabulated)<br>3. Check if a platform response caused unexpected state transition<br>4. Check Agent rule config for ambiguous conditions | Root cause identified |
| **Recovery** | 1. If LLM issue: degrade Agent to rule-based mode or pause<br>2. If rule issue: update agent rule, test in staging<br>3. If state issue: reset Agent state to last known good checkpoint<br>4. Re-enable Agent gradually with increased monitoring | Agent returns to normal behavior |
| **Manual override** | 1. Owner can force-disable the Agent from cockpit dashboard<br>2. Owner can manually approve/reject pending Agent decisions<br>3. If Agent made errant writes: manual correction template for each platform | No irreversible platform damage |

### 2.4 Migration Failure

| Phase | Action | Expected Outcome |
|-------|--------|-----------------|
| **Detection** | Alert: `migration.{name}.failed` or backend fails to start after deployment | Incident opened, severity P1 (P0 if production DB) |
| **Containment** | 1. Immediately disable any automated down-migration (avoid double-failure)<br>2. Isolate the affected server (remove from load balancer)<br>3. Run down migration on staging first | Production not yet affected (if caught early) |
| **Root cause** | 1. Check migration log for exact error<br>2. Check for conflicting migration versions<br>3. Check for syntax error in SQL<br>4. Check for constraint violation on existing data | Root cause identified |
| **Recovery** | 1. Run down migration on production<br>2. Deploy previous release version<br>3. Verify data integrity after rollback (see ROLLBACK_AND_RECOVERY.md)<br>4. Fix migration, test in staging, re-deploy | Previous state restored, migration fixed |
| **Manual override** | 1. If down migration fails: DBA must manually reconstruct previous schema<br>2. If data corrupted during up migration: restore from backup<br>3. Owner decides whether to fix forward or roll back | Minimum data loss |

### 2.5 Authentication Outage

| Phase | Action | Expected Outcome |
|-------|--------|-----------------|
| **Detection** | Alert: `auth.login.error_rate > threshold` or failed JWT validation > 10% of requests | Incident opened, severity P0 |
| **Containment** | 1. Check JWT secret hasn't been rotated unexpectedly<br>2. Check auth service is running and reachable<br>3. If JWT secret mismatch: reload old secret from backup | Logins restored |
| **Root cause** | 1. Check config change history (was JWT_SECRET changed?)<br>2. Check deployment logs (was auth module updated?)<br>3. Check for clock skew affecting token validation | Root cause identified |
| **Recovery** | 1. If JWT secret: restore correct secret<br>2. If deployment: rollback auth module<br>3. If clock skew: fix NTP<br>4. After recovery, force all users to re-login (token revocation) | Authentication working, no data loss |
| **Manual override** | 1. Emergency access: generate one-time admin bypass token<br>2. Owner can authorize emergency access to specific users only<br>3. Rotate all secrets after emergency access is used | Controlled access restored |

## 3. Manual Override Procedures

### 3.1 Agent Stuck

Trigger: Agent has been processing the same task longer than the configured timeout (default: 30 min).

```
1. Confirm state:
   - Open cockpit dashboard, check Agent "{id}" status
   - If status = "running" and last_heartbeat > (now - timeout) → stuck

2. Force-stop:
   DB > UPDATE agent_executions SET status = 'aborted', updated_at = NOW()
        WHERE agent_id = '{id}' AND status = 'running';

3. Check for side effects:
   - Are there in-flight platform API calls from this Agent?
   - Are there partially applied changes that need rollback?

4. Resolve side effects:
   - If platform write in progress: check platform dashboard for partial state
   - If EventBus event partially consumed: check DLQ for orphaned events

5. Restart:
   - Reset agent state: agent.execution_service.Reset(id)
   - If configured to retry: queue the task again
   - If not configured to retry: log the failure, move to next task

6. Document:
   - Log the stuck event with correlation ID, duration, and resolution steps
```

### 3.2 Platform Write-Back Failed

Trigger: Platform returned error for a write operation (price update, inventory sync, listing publish).

```
1. Identify scope:
   - Check platform_error log for failed operation details
   - Check platform response body for error code and message

2. Classify error:
   - Transient (HTTP 429, 503, timeout) → retry with backoff (max 3 times)
   - Authorization (HTTP 401, 403) → rotate credentials, retry once
   - Bad request (HTTP 400, 422) → log payload, do not retry, alert human
   - Not found (HTTP 404) → check if entity was deleted on platform
   - Unknown → alert human

3. Apply resolution:
   - Transient: automated retry with exponential backoff
   - Auth: rotate credentials, replay failed write
   - Bad request: manual payload review and correction
   - Not found: reconcile platform state with local DB

4. Manual bypass:
   DB > UPDATE platform_operation_log SET status = 'skipped', notes = 'Manual override: ...'
        WHERE id = '{operation_id}';
   - Owner can approve manual retry from cockpit dashboard
   - Owner can approve deletion of queued write-back

5. Check for dependency cascade:
   - Did this failure block dependent operations? (e.g., pricing before listing)
   - Re-queue dependent operations after resolution
```

### 3.3 Approval Timeout

Trigger: An Agent action requires Owner approval and the approval window has expired (default: 24 h).

```
1. Confirm timeout:
   - Check action_policy.approval_requests for request_id = '{id}'
   - If status ≠ 'approved' AND created_at < NOW() - INTERVAL '{timeout_hours} hours'

2. Default action (configured per action type):
   - If action is configured to auto-reject: set status = 'rejected', reason = 'timeout'
   - If action is configured to auto-approve: set status = 'approved', reason = 'auto-approval by timeout policy'
   - If action is configured to hold: leave as pending, alert higher escalation path

3. Notify:
   - Alert escalation contact: "{action_type} action timed out without Owner approval"
   - If auto-rejected: notify Owner that action was rejected by timeout
   - If auto-approved: notify Owner that action was auto-approved by timeout

4. Post-incident review:
   - Is the timeout period appropriate for this action type?
   - Should this action type require a secondary approver?
   - Should this action type be reconfigured to default-approve or default-reject?

   Configure via:
   DB > UPDATE action_policy_settings
        SET timeout_action = '{reject|approve|hold}',
            timeout_hours = '{N}'
        WHERE action_type = '{type}';
```

## 4. Post-Incident Review (PIR) Template

Complete within 5 business days of incident resolution.

### 4.1 Incident Summary

| Field | Value |
|-------|-------|
| Incident ID | |
| Severity | |
| Date/Time | |
| Duration | |
| Affected components | |
| Triggered by (deployment / external / user / agent) | |
| Resolved by | |

### 4.2 Timeline

| Time | Event |
|------|-------|
| | First alert fired |
| | Incident acknowledged by |
| | Containment action taken |
| | Root cause identified |
| | Mitigation applied |
| | Service restored |
| | Post-incident review scheduled |

### 4.3 Root Cause

One paragraph describing the root cause. Include:
- What was the trigger
- Why it happened (not just what happened)
- What conditions made it possible
- Why it wasn't caught earlier

### 4.4 Impact

| Metric | Value |
|--------|-------|
| Orders affected | |
| Financial impact (estimated) | |
| User-facing downtime (min) | |
| Platform write-backs lost/corrupted | |
| Agent hours lost | |

### 4.5 Action Items

| # | Action | Owner | Severity | Due Date | Status |
|---|--------|-------|----------|----------|--------|
| | | | | | |
| | | | | | |

### 4.6 Lessons Learned

- What went well:
- What went wrong:
- What would we do differently:
- What monitoring gaps exist:
- What documentation gaps exist:

### 4.7 Owner Decision Points

| Decision | Owner Decision | Date |
|----------|---------------|------|
| Accept risk of recurrence? | | |
| Adjust kill switch configuration? | | |
| Adjust approval timeout policy? | | |
| Invest in automated recovery? | | |
| Change trial boundary? | | |
