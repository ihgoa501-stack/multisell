# Changelog

## v0.2.0 (2026-06-25)

### Major
- New Go backend (`backend-go/`) with Gin/GORM/PostgreSQL — full replacement of Python/FastAPI stack
- New Next.js 15 frontend (`frontend-next/`) with Ant Design — replaces Vue 3/Naive UI
- AI AgentOS: 10 agents (A1-A7, G0-G3) with LLM-powered decision analysis, trust score, entropy defense
- Event-driven architecture: in-memory event bus with outbox persistence, scheduler, command dispatcher
- Platform integrations: Shopee, Ozon, 1688 API adapters
- Full CI/CD pipeline: GitHub Actions, Docker Compose (dev/prod/monitoring), deploy/rollback scripts
- Production-readiness: CORS, JWT auth, rate limiter, Sentry, Prometheus metrics, structured logging

### Infrastructure
- Q3 production readiness: middleware stack, graceful shutdown, auth guard on all API routes
- Prometheus + Grafana monitoring stack (`deploy/prometheus/`, `deploy/grafana/`)
- k6 load testing suite (`backend-go/loadtest/`)
- Pre-commit hooks (pre-commit-config.yaml)
- Versioned SQL migrations (000001-000011)

### Under Review
- [P0] T1: WebSocket /ws needs JWT auth
- [P1] T2-T7: EventBus async worker, error info leak, AgentOS error handling, N+1 queries, entropy defense timer, agent pipeline timeout
- [P2] T8-T9: Dead code removal, JWT refresh audit
- [P3] Frontend design review, distributed tracing, horizontal scaling deferred

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
