# Python Legacy Stack — Archive Notice

**Date:** 2026-06-24
**Status:** All AI agent logic migrated to Go. Python backend frozen.

## What Has Been Migrated

### Go Implementation (`backend-go/`)

| Python File | Go File | Status |
|---|---|---|
| agent/agents/product_scout.py | internal/agent/impl/product_scout.go | Ported |
| agent/agents/listing_optimizer.py | internal/agent/impl/listing_optimizer.go | Ported |
| agent/agents/ad_advice.py | internal/agent/impl/ad_advice.go | Ported |
| agent/agents/customer_service.py | internal/agent/impl/customer_service.go | Ported |
| agent/agents/inventory_alert.py | internal/agent/impl/inventory_alert.go | Ported |
| agent/agents/profit_watch.py | internal/agent/impl/profit_watch.go | Ported |
| agent/agents/compliance.py | internal/agent/impl/compliance_guard.go | Ported |
| agent/agents/dashboard.py | internal/agent/impl/dashboard.go | Ported |
| agent/agents/warehouse_customs.py | internal/agent/impl/warehouse_customs.go | Ported |
| agent/agents/discount_risk.py | internal/agent/impl/discount_risk.go | Ported |

### New Go-Only Agents

| Agent | File | Description |
|---|---|---|
| G0 Coordinator | internal/agent/impl/coordinator.go | Supervisor, escalation, coordination |
| A8 Settlement Recon | internal/agent/impl/settlement_recon.go | Settlement import, reconciliation |
| A9 Batch Ops | internal/agent/impl/batch_ops.go | Batch price/inventory/listing ops |
| A10 Logistics Ops | internal/agent/impl/logistics_ops.go | Carrier compare, shipping audit |
| A11 Aftersales Mgmt | internal/agent/impl/aftersales_mgmt.go | Returns, refunds, disputes |

### Infrastructure (Go-Only)

| Module | Description |
|---|---|
| internal/domain/actionpolicy/ | Auto-approval policy rules engine |
| internal/domain/trustscore/ | Agent trust scores + autonomy upgrade |
| Frontend /actions | Action Center (approve/reject/execute) |
| Frontend /agents/trust | Trust scores + autonomy levels |
| Frontend /settings/policy | Approval policy rules viewer |

## What Remains in Python

Python backend (`backend/`) is frozen — reference only. No new development.
