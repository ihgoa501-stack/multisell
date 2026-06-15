# Runtime And Migrations Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make MultiSell reproducible from a fresh checkout with aligned local/Docker database settings, a real Alembic baseline, and a backend test database path.

**Architecture:** Keep the current FastAPI application and Docker Compose layout. Use PostgreSQL as the only runtime database, create a separate test database in the Docker Postgres container, and run Alembic migrations before backend startup.

**Tech Stack:** Python 3.11, FastAPI, SQLAlchemy async, Alembic, PostgreSQL 15, pytest, Docker Compose, Vue 3/Vite.

---

### Task 1: Align Runtime Configuration

**Files:**
- Modify: `backend/app/config.py`
- Modify: `.env.example`

- [ ] **Step 1: Set local defaults to Docker-compatible PostgreSQL**

Use these defaults in `backend/app/config.py`:

```python
DATABASE_URL: str = "postgresql+asyncpg://postgres:postgres@localhost:5432/product_management"
DATABASE_URL_SYNC: str = "postgresql+psycopg2://postgres:postgres@localhost:5432/product_management"
```

- [ ] **Step 2: Add `.env.example`**

Create `.env.example` with:

```env
APP_NAME=MultiSell - AI跨境电商商品中台
APP_VERSION=2.0.0
DEBUG=True
APP_PORT=8001

DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management
DATABASE_URL_SYNC=postgresql+psycopg2://postgres:postgres@localhost:5432/product_management
TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test

AUTH_ENABLED=True
ENCRYPTION_KEY=change-me-to-a-random-32-byte-secret

UPLOAD_DIR=./uploads
STATIC_URL=/static

LLM_API_URL=https://api.openai.com/v1/chat/completions
LLM_API_KEY=
LLM_MODEL=gpt-4o-mini
```

- [ ] **Step 3: Verify config imports**

Run:

```bash
cd backend && python3 -c "from app.config import settings; print(settings.DATABASE_URL)"
```

Expected: prints a PostgreSQL URL using `postgres:postgres@localhost:5432/product_management`.

### Task 2: Make Docker Compose Create Runtime And Test Databases

**Files:**
- Modify: `docker-compose.yml`
- Create: `backend/scripts/init-db.sql`

- [ ] **Step 1: Add init SQL**

Create `backend/scripts/init-db.sql`:

```sql
SELECT 'CREATE DATABASE product_management_test'
WHERE NOT EXISTS (
    SELECT FROM pg_database WHERE datname = 'product_management_test'
)\gexec
```

- [ ] **Step 2: Mount init SQL into Postgres**

In `docker-compose.yml`, add this db volume:

```yaml
- ./backend/scripts/init-db.sql:/docker-entrypoint-initdb.d/init-db.sql:ro
```

- [ ] **Step 3: Run migrations before backend startup**

In the backend service, add:

```yaml
command: >
  sh -c "alembic upgrade head &&
         uvicorn app.main:app --host 0.0.0.0 --port 8000"
```

- [ ] **Step 4: Add backend test database environment**

In the backend service environment, add:

```yaml
TEST_DATABASE_URL: postgresql+asyncpg://postgres:postgres@db:5432/product_management_test
AUTH_ENABLED: "true"
```

### Task 3: Replace Empty Alembic Baseline

**Files:**
- Modify: `backend/alembic/versions/c065b94903eb_baseline.py`

- [ ] **Step 1: Make baseline create current metadata**

Replace empty upgrade/downgrade with:

```python
def upgrade() -> None:
    from app.database import Base
    import app.models  # noqa: F401

    bind = op.get_bind()
    Base.metadata.create_all(bind=bind)


def downgrade() -> None:
    from app.database import Base
    import app.models  # noqa: F401

    bind = op.get_bind()
    Base.metadata.drop_all(bind=bind)
```

- [ ] **Step 2: Verify migration script imports**

Run:

```bash
cd backend && python3 -m py_compile alembic/versions/c065b94903eb_baseline.py
```

Expected: command exits successfully.

### Task 4: Make Backend Tests Use The Test Database Explicitly

**Files:**
- Modify: `backend/tests/conftest.py`

- [ ] **Step 1: Use `TEST_DATABASE_URL` fallback**

Set the default test URL to:

```python
os.environ.setdefault(
    "DATABASE_URL",
    os.environ.get(
        "TEST_DATABASE_URL",
        "postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test",
    ),
)
```

- [ ] **Step 2: Reset test schema at session start**

In `prepare_db`, before `Base.metadata.create_all`, run `Base.metadata.drop_all` to avoid unique-key failures from previous interrupted runs:

```python
async with engine.begin() as conn:
    await conn.run_sync(Base.metadata.drop_all)
    await conn.run_sync(Base.metadata.create_all)
```

- [ ] **Step 3: Drop test schema at session end**

Replace row-by-row delete cleanup with:

```python
async with engine.begin() as conn:
    await conn.run_sync(Base.metadata.drop_all)
await engine.dispose()
```

### Task 5: Update Developer Documentation

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Correct Docker URLs**

Document:

```markdown
访问前端：http://localhost:3000
访问后端 API：http://localhost:8000/docs
```

- [ ] **Step 2: Document local setup**

Document:

```bash
docker compose up -d db
cd backend
python3 -m venv .venv
.venv/bin/pip install -r requirements.txt
.venv/bin/alembic upgrade head
.venv/bin/python seed.py
.venv/bin/uvicorn app.main:app --reload --port 8001
```

- [ ] **Step 3: Document test command**

Document:

```bash
cd backend
TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test \
  python3 -m pytest -q
```

### Task 6: Verify Phase 0

**Files:**
- No new code edits unless verification exposes a root cause.

- [ ] **Step 1: Run backend compile checks**

Run:

```bash
cd backend && python3 -m py_compile app/config.py tests/conftest.py alembic/env.py alembic/versions/c065b94903eb_baseline.py
```

Expected: exits successfully.

- [ ] **Step 2: Run frontend build**

Run:

```bash
cd frontend && npm run build
```

Expected: build exits successfully.

- [ ] **Step 3: Run backend tests when PostgreSQL is available**

Run:

```bash
cd backend && python3 -m pytest -q
```

Expected: tests either pass or reach business assertions. A connection refused error means local PostgreSQL is not running, not that Phase 0 code failed.
