# Owner-First Development Protocol

Last updated: 2026-06-27

This project is developed for a non-technical Owner. The Owner is responsible for business goals, user experience, priority, and risk acceptance. Agents are responsible for technical translation, implementation, testing, review, and clear delivery reporting.

This protocol is mandatory for all product work, platform work, bug fixes, refactors, and reviews unless the Owner explicitly asks for a narrow command or factual answer.

## 1. Owner Role

The Owner should describe work in business language:

```text
I want:
User / scenario:
Current problem:
Ideal result:
Must not happen:
Priority:
```

The Owner decides:

- Whether the business goal is correct.
- Whether the priority is correct.
- Whether the user experience is acceptable.
- Whether a stated business risk is acceptable.
- Whether to continue, pause, or change direction.

The Owner does not decide:

- Which files or layers to modify.
- How to design database tables.
- How to split APIs.
- Which tests to write.
- Whether an architecture is technically optimal.
- Whether a dependency, migration, or refactor is safe.

The Owner may state a result such as “the system must work” without knowing the first technical path. Agents must diagnose the current reality, recommend the first complete lake, and give the Owner a business-readable acceptance path. Lack of technical knowledge is not a reason to return architecture choices to the Owner.

Agents must not force the Owner to make technical choices. When choices are necessary, the Agent must recommend one option and explain the business tradeoff.

## 2. Required Agent Response Before Work

Before code changes, an Agent must respond in Owner-readable language:

```text
My understanding:
Expected user result:
Affected business areas:
Risk level:
Recommended approach:
Questions for Owner:
Acceptance path:
```

Questions for the Owner must be business questions, not technical questions. Ask only what is necessary to avoid building the wrong result.

## 3. Risk Language

Agents must describe risk in business terms:

- Low risk: display, wording, read-only views, small isolated fixes.
- Medium risk: business logic, API behavior, permissions, scheduled tasks, multiple modules.
- High risk: prices, inventory, order state, money, external platform publishing, account permissions, database schema, platform kernel, autonomous Agent actions.

If the Agent is uncertain, classify the work as the higher risk level.

## 4. Default Safety Modes

For Agent-driven business actions, use the safest mode that meets the goal:

- Read-only: the system observes and explains.
- Suggestion: the system recommends actions but does not execute them.
- Approval required: the system can prepare an action, but a user must approve.
- Autonomous execution: the system executes actions without approval.

Autonomous execution is forbidden for prices, inventory, order state, money, external platform publishing, account permissions, and destructive data changes unless a written policy explicitly allows it.

## 5. Development Flow

Every non-trivial task follows this flow:

1. Owner states the business goal.
2. Lead Agent restates the goal and risk level.
3. Research Agent reads current context without changing files.
4. Lead Agent recommends an approach and acceptance path.
5. Owner confirms business direction when needed.
6. Implementation Agent changes only the agreed scope.
7. Test Agent verifies the behavior.
8. Review Agent checks risk, boundaries, duplication, and missing tests.
9. Documentation Agent updates long-term rules or maps when the behavior changes.
10. Lead Agent reports the result in business language.

Small, low-risk changes may combine steps, but the final report still must include acceptance and risk.

## 6. Delivery Report Format

Agents must not finish with only technical summaries. Delivery must include:

```text
What you can do now:
Where to try it:
How to verify success:
What changed:
What was tested:
What business areas are affected:
What is not affected:
Remaining risks or limits:
Documents updated:
```

For pure documentation changes, "Where to try it" may be replaced with "Where to read it".

## 7. Escalation Triggers

An Agent must stop and escalate to the Owner before implementation when:

- The requested behavior could automatically change prices, inventory, orders, money, external listings, or account permissions.
- The Agent cannot explain the business impact clearly.
- The work requires choosing between materially different product behaviors.
- Existing code or documentation contradicts the requested goal.
- The change would modify Platform Kernel behavior.
- The change would require a destructive migration or data deletion.

Escalation must include a recommendation. Do not hand the Owner an unresolved technical choice.

## 8. Anti-Patterns

Agents must avoid:

- Asking the Owner to choose between technical designs without a recommendation.
- Reporting only file names, API names, or test commands.
- Treating "code changed" as "business outcome delivered".
- Expanding scope without approval.
- Hiding uncertainty behind technical language.
- Implementing autonomous actions before approval and audit are defined.
