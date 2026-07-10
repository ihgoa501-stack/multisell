# AI-Native Developer Handbooks Overview

> **Target Audience**: AI coding agents spawned in this repository to implement features, fix bugs, or maintain systems.
> **Scope**: Explains how to develop safely under the Multi-Domain AgentOS framework.

Welcome! This repository is designed to be developed and maintained **100% by AI coding agents** pair-programming with a non-technical Owner. To ensure all development is safe, robust, and free of regression errors, you **MUST** read and follow these guidebooks before making edits:

---

## 📖 The Guidebooks

1. **[AI-Native Context Sync (ARCS) Manual](file:///Users/lc/multisell/docs/guides/ai-native-arcs-guide.md)**
   * **What it covers**: Manifest formats (`.ai-manifest.json`), handoff ledgers (`.loop/dev-state.md`), single source of truth catalogs, and onboarding checklists.
   * **Why it matters**: Resolves session amnesia so you can align on goals in under 5 seconds.
2. **[Automated Sandbox Staging Guide](file:///Users/lc/multisell/docs/guides/ai-native-sandbox-guide.md)**
   * **What it covers**: Docker Compose sandbox configurations, internal network routing, shared compiler volume caches, and E2E Playwright tests.
   * **Why it matters**: Ensures all changes are verified in isolated stage networks before merging.
3. **[Stateful Mocking & Outbound safety Guide](file:///Users/lc/multisell/docs/guides/ai-native-mocking-guide.md)**
   * **What it covers**: Stateful mock databases, financial cent precision, transactional advisory locks, and HTTP-level outbound network blocks (`FailSafeRoundTripper`).
   * **Why it matters**: Prevents live API request leaks and storefront account suspensions.
4. **[AI Edit Loop Prevention Guide](file:///Users/lc/multisell/docs/guides/ai-native-loop-prevention-guide.md)**
   * **What it covers**: State signatures, normalized error hashing, ping-pong oscillation loops, error stagnation, and fallback patch saves.
   * **Why it matters**: Halts wasteful token-burning cycles and loops.

---

## 🛠️ Unified System Layout (For AI Developers)
When developing new verticals (e.g., E-Commerce, Finance, Trade), the codebase enforces a strict **separation of mechanism and policy**:

```
                       ┌────────────────────────────┐
                       │    Go Platform Kernel      │
                       │ (Auth, RBAC, EventBus,     │
                       │  Scheduler, ToolBridge)    │
                       └─────────────┬──────────────┘
                                     │ Implicit Go Interfaces (Contracts)
                                     ▼
              ┌──────────────────────┼──────────────────────┐
              ▼                      ▼                      ▼
     [E-Commerce Suite]       [Finance Suite]        [Foreign Trade Suite]
    - Product, Order, SKU    - Bank accounts,       - Inquiries, RFQ,
    - PlatformAdapter        - BankAdapter          - RFQAdapter
```

* **Go Platform Kernel**: Defines interfaces (contracts) and handles authorization, auditing, and routing. **AI agents must not modify this layer without Owner approval.**
* **Domain Modules**: Implement the business logic for specific industries inside independent packages. **AI agents have full permission to create and refactor this layer.**

Before starting your slice, check the ledger at `.loop/dev-state.md` to see your current goal!
