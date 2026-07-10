# AI-Native Repository Context Synchronization (ARCS) Manual

> **Purpose**: Guide AI coding agents on how to synchronize project context, read system manifests, and maintain the handoff ledger between development sessions.

---

## 1. The Core Problem: AI Session Amnesia
AI coding agents are stateless. Each time a new session starts, the agent has zero memory of past runs. Reading the entire repository to reconstruct the development state wastes context tokens and is prone to errors.

**ARCS** resolves this by providing:
1. **`.ai-manifest.json`**: A tiny, dense configuration index that allows incoming agents to parse the tech stack and index directories in under 200 tokens.
2. **`.loop/dev-state.md`**: A session handoff ledger showing the current goal, active slice, system delta, and next steps.
3. **DSL (Documentation Sync Linter)**: A lint checker enforcing that all API endpoints and database schemas match the module catalog.

---

## 2. Root Manifest Schema: `.ai-manifest.json`
Every AI agent entering the repository **MUST** read `.ai-manifest.json` first. The file is structured as follows:

```json
{
  "manifest_version": "1.0.0",
  "project_name": "LingMirror (MultiSell)",
  "active_stack": {
    "backend": "Go 1.21 / Gin",
    "frontend": "Next.js 14 / App Router / AntD",
    "database": "PostgreSQL 15 / GORM v2",
    "event_bus": "In-memory Publisher-Subscriber (internal/platform/eventbus)"
  },
  "entrypoints": {
    "backend_main": "backend-go/cmd/server/main.go",
    "frontend_root": "frontend-next/src/app/"
  },
  "core_indexes": {
    "module_catalog": "docs/reference-module-catalog.md",
    "documentation_index": "docs/INDEX.md",
    "governance_directory": "docs/governance/",
    "session_state": ".loop/dev-state.md",
    "codegraph_db": ".codegraph/codegraph.db",
    "agent_instructions": "AGENTS.md",
    "claude_instructions": "CLAUDE.md"
  },
  "safe_boundaries": {
    "risk_manifest": "docs/governance/PLATFORM_CONSTITUTION.md",
    "high_risk_layers": ["price", "inventory", "order", "finance", "audit", "auth"]
  }
}
```

*Note: Do not expand this manifest with dynamic data. It must remain static and small to conserve token consumption.*

---

## 3. Session Handoff Ledger: `.loop/dev-state.md`

### 3.1 Strict Context Compaction Rules
To prevent context inflation, the handoff ledger has a **Strict Compaction Policy**:
* **Keep only the 1-2 most recent slices**: Delete older completed slices from this ledger (archive them to `.loop/history.md`).
* **No Raw Console Output**: Do not paste raw multi-line compiler outputs. Use status abbreviations (e.g., `go test: PASS`, `npm run lint: PASS`).
* **Short Bullet Points**: Keep the system delta and debt descriptions to short single-sentence descriptions.

### 3.2 Handoff Template

```markdown
# AI Developer Handoff Ledger

- **Current Goal**: "[Overall feature goal]"
- **Active Task Slice**: "[Task name under execution]"
- **Completed in Branch**:
  - [x] [Completed item 1]
  - [x] [Completed item 2]
- **Verification Results**:
  - `go test ./...`: PASS (12/12)
  - `npm run lint`: PASS
  - `npm run build`: PASS
- **New System Delta**:
  - [File]: Description of change
  - [API]: Method + Route
  - [Model]: GORM model modification
- **Open Debt/Unresolved Warnings**:
  - [Warning/Debt detail]
- **Next Step**: "[Exact instruction for the next agent session]"
```

---

## 4. Documentation Sync Linter (DSL)

To prevent the documentation from getting out of sync with code (a common cause of AI developer hallucinations), a pre-commit or CI script checks Gin route definitions and registers them against the catalog.

### Linter Shell Script (`scripts/check_module_catalog.sh`)
```bash
#!/usr/bin/env bash
set -euo pipefail

CATALOG="docs/reference-module-catalog.md"
if [ ! -f "$CATALOG" ]; then
  echo "Error: Catalog file $CATALOG not found"
  exit 1
fi

echo "Verifying API endpoints in docs/reference-module-catalog.md..."
# Extract endpoints defined in backend-go/internal/domain (finding patterns like r.GET("/path") or r.POST("/path"))
grep -r -h -o -E '"/api/v1/[a-zA-Z0-9_\-/:]+"' backend-go/internal/ | tr -d '"' | sort -u | while read -r endpoint; do
  if ! grep -q "$endpoint" "$CATALOG"; then
    echo "Error: API endpoint $endpoint is not documented in $CATALOG!"
    exit 1
  fi
done

echo "Verification complete. All Gin endpoints are documented."
```

---

## 5. Agent Onboarding Checklist
When you spawn in this repository, follow these 5 steps:
1. **Intake**: Read `.ai-manifest.json` and `.loop/dev-state.md` to verify active technologies and current task goals.
2. **Exclusion Check**: Check `.ai-manifest.json` -> `safe_boundaries.high_risk_layers` to see if your code is editing a restricted layer (requires approval check).
3. **Catalog Verification**: Read `docs/reference-module-catalog.md` to see what models and APIs already exist in the target package.
4. **Implement & Test**: Write tests first (TDD), implement the feature, and run package tests.
5. **Update Ledger**: Update `.loop/dev-state.md`, run the DSL linter (`scripts/check_module_catalog.sh`), and commit.
