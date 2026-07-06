# Alert Rules

This document defines all monitoring alert rules for 凌镜 LingMirror.

## Priority Levels

| Level | Meaning | Response |
|-------|---------|----------|
| P0 | Critical — immediate action required | Stop affected agents, notify on-call |
| P1 | High — action required within 1 hour | Notify on-call, investigate |
| P2 | Medium — action required within 24 hours | Log to dashboard, notify during business hours |
| P3 | Warning — informational | Log to dashboard |

## Alert Rules

### R1: Agent Consecutive Failure

- **Rule**: Any agent fails > 3 times consecutively
- **Priority**: P1
- **Check**: Scheduled health check (every 5 min)
- **Action**: Notify admin, pause the agent, log full error context
- **Notified via**: WebSocket push + notification record

### R2: Order Sync Interruption

- **Rule**: Order synchronization (Shopee, Ozon, etc.) stalls > 30 minutes
- **Priority**: P1
- **Check**: Compare last_sync_at timestamp against current time
- **Action**: Trigger manual sync, notify integration owner
- **Notified via**: WebSocket push + notification record

### R3: Abnormal Profit Rate

- **Rule**: Product profit rate < -10%
- **Priority**: P2 — anomalous marker
- **Check**: Profit Watch Agent (A6) on each evaluation cycle
- **Action**: Flag product with anomalous marker, log for manual review
- **Notified via**: Notification record

### R4: LLM Monthly Budget Exceeded

- **Rule**: Monthly LLM API cost > 80% of budget
- **Priority**: P3 (warning)
- **Action**: Notify admin to review usage

- **Rule**: Monthly LLM API cost > 100% of budget
- **Priority**: P0 (critical)
- **Action**: Immediately halt all agent LLM calls, notify on-call
- **Notified via**: WebSocket push + notification record

### R5: Sync Failure Rate

- **Rule**: Synchronization failure rate > 5% over a 1-hour window
- **Priority**: P2
- **Check**: Running failure ratio from integration logs
- **Action**: Inspect platform adapter, check credentials or rate limits
- **Notified via**: Notification record

## Integration

Rules fire via the `NotifyAlert` function in the notification service:

```go
svc.NotifyAlert(ctx, "p1", "Agent Consecutive Failure", "A5 stock_alert failed 4 times")
```

All P0/P1 alerts trigger a WebSocket push in addition to persistent notification records.
