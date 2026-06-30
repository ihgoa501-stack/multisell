# Production Closed Loop Acceptance

## Goal

Verify that a candidate product can move from evaluation to a blocked listing task, then to Owner approval, then to listing task execution without bypassing approval.

## Preconditions

- Backend running on `http://localhost:8080`.
- Frontend running on `http://localhost:3000`.
- User is logged in with permissions for candidate, approval, listing task, and owner cockpit pages.
- Database migrations are applied through `000030_approval_target_fields`.

## Scenario

1. Open `/candidates`.
2. Select a complete, profitable candidate product.
3. Run evaluation.
4. Confirm the page shows a generated listing task and says approval is required.
5. Open `/owner`.
6. Confirm pending approval count increased by 1.
7. Open `/listing-tasks/:id`.
8. Confirm the blocked task shows "等待 Owner 审批" and offers "去审批", not "重试" or "启动执行".
9. Open `/approval`.
10. Confirm a pending `publish` approval exists for `listing_task`.
11. Try to execute the listing task before approval through `POST /api/v1/listing-task/:task_id/execute`.
12. Confirm the API rejects execution with `approval required`.
13. Approve the request in `/approval`.
14. Return to `/listing-tasks/:id`.
15. Confirm the task shows "审批已通过，可以执行".
16. Execute the listing task.
17. Confirm the task changes from `blocked` to `completed` or Prism-blocked if Prism strict compliance fails.
18. Open `/operation-logs`.
19. Confirm review and execution-related operations are traceable.

## Success Criteria

- No listing task execution happens before approval.
- Owner sees pending work from `/owner`.
- `/owner` does not directly approve or reject listing task status.
- `/candidates` explains why the candidate is recommended, cautious, or skipped.
- `/approval` explains what approve/reject will do in business language.
- `/listing-tasks/:id` blocks execution UI until approval.
- Approval record includes target type, target id, reason, risk level, requester, reviewer, and status.
- The final state is visible from listing task detail and operation logs.

## Must Not Happen

- A blocked listing task executes without an approved approval request.
- The UI hides the high-risk nature of publish approval.
- `/owner` directly changes listing task status instead of routing through approval.
- Agent recommendation directly publishes to an external platform.
