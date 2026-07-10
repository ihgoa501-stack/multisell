# Module: `workflow`

Package: `backend-go/internal/domain/workflow/`

**Base mount prefix:** `/api/v1`

## API Routes

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/workflow/defs` | `h.ListDefs` |
| `POST` | `/api/v1/workflow/defs` | `h.CreateDef` |
| `POST` | `/api/v1/workflow/defs/:defId/start` | `h.StartRun` |
| `DELETE` | `/api/v1/workflow/defs/:id` | `h.DeleteDef` |
| `GET` | `/api/v1/workflow/defs/:id` | `h.GetDef` |
| `PUT` | `/api/v1/workflow/defs/:id` | `h.UpdateDef` |
| `GET` | `/api/v1/workflow/monitor` | `h.GetMonitor` |
| `GET` | `/api/v1/workflow/monitor/stats` | `h.GetMonitorStats` |
| `GET` | `/api/v1/workflow/runs` | `h.ListRuns` |
| `GET` | `/api/v1/workflow/runs/:id` | `h.GetRun` |
| `POST` | `/api/v1/workflow/runs/:id/advance` | `h.AdvanceStep` |
| `POST` | `/api/v1/workflow/runs/:id/pause` | `h.PauseRun` |
| `POST` | `/api/v1/workflow/runs/:id/resume` | `h.ResumeRun` |
| `POST` | `/api/v1/workflow/runs/:id/retry` | `h.RetryRun` |
| `GET` | `/api/v1/workflows` | `h.ListWorkflows` |
| `POST` | `/api/v1/workflows` | `h.CreateWorkflow` |
| `GET` | `/api/v1/workflows/:id` | `h.GetWorkflow` |
| `POST` | `/api/v1/workflows/runs/:id/approve` | `h.ApproveStep` |
| `POST` | `/api/v1/workflows/runs/:id/reject` | `h.RejectStep` |

## Models

### `WorkflowDef`
**DB table:** `workflow_def`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `Name` | `string` | `name` | `name` | NOT NULL |
| `Description` | `string` | `description,omitempty` | `description` |  |
| `Steps` | `string` | `steps` | `steps` | NOT NULL, default:'[]' |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |
| `UpdatedAt` | `time.Time` | `updated_at` | `updated_at` |  |

### `WorkflowRun`
**DB table:** `workflow_run`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `WorkflowDefID` | `int64` | `workflow_def_id` | `workflow_def_id` |  |
| `Name` | `string` | `name` | `name` | NOT NULL |
| `Status` | `string` | `status` | `status` | default:pending |
| `Context` | `string` | `context` | `context` | default:'{}' |
| `StartedAt` | `*time.Time` | `started_at,omitempty` | `started_at` |  |
| `CompletedAt` | `*time.Time` | `completed_at,omitempty` | `completed_at` |  |
| `Error` | `string` | `error,omitempty` | `error` |  |
| `CurrentNodeID` | `int64` | `current_node_id` | `current_node_id` | default:0 |
| `RetryCount` | `int` | `retry_count` | `retry_count` | default:0 |
| `MaxRetries` | `int` | `max_retries` | `max_retries` | default:3 |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |
| `UpdatedAt` | `time.Time` | `updated_at` | `updated_at` |  |
| `Steps` | `[]WorkflowStepRun` | `steps,omitempty` | `—` |  |

### `WorkflowStepRun`
**DB table:** `workflow_step_run`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `WorkflowRunID` | `int64` | `workflow_run_id` | `workflow_run_id` | NOT NULL |
| `StepName` | `string` | `step_name` | `step_name` | NOT NULL |
| `StepType` | `string` | `step_type` | `step_type` | NOT NULL |
| `ParentID` | `*int64` | `parent_id,omitempty` | `parent_id` |  |
| `Status` | `string` | `status` | `status` | default:pending |
| `Input` | `string` | `input` | `input` | default:'{}' |
| `Output` | `string` | `output` | `output` | default:'{}' |
| `Error` | `string` | `error,omitempty` | `error` |  |
| `Attempt` | `int` | `attempt` | `attempt` | default:1 |
| `MaxAttempts` | `int` | `max_attempts` | `max_attempts` | default:1 |
| `TimeoutSeconds` | `int` | `timeout_seconds` | `timeout_seconds` | default:300 |
| `StartedAt` | `*time.Time` | `started_at,omitempty` | `started_at` |  |
| `CompletedAt` | `*time.Time` | `completed_at,omitempty` | `completed_at` |  |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |
| `UpdatedAt` | `time.Time` | `updated_at` | `updated_at` |  |

### `StepDef`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Name` | `string` | `name` | `—` |  |
| `Type` | `string` | `type` | `—` |  |
| `AgentID` | `string` | `agent_id,omitempty` | `—` |  |
| `DecisionPoint` | `string` | `decision_point,omitempty` | `—` |  |
| `Command` | `string` | `command,omitempty` | `—` |  |
| `Condition` | `string` | `condition,omitempty` | `—` |  |
| `Inputs` | `map[string]interface{}` | `inputs,omitempty` | `—` |  |
| `Forks` | `[]StepDef` | `forks,omitempty` | `—` |  |
| `JoinSteps` | `[]string` | `join_steps,omitempty` | `—` |  |
| `WaitForEvent` | `string` | `wait_for_event,omitempty` | `—` |  |
| `DelaySeconds` | `int` | `delay_seconds,omitempty` | `—` |  |
| `TimeoutSeconds` | `int` | `timeout_seconds,omitempty` | `—` |  |
| `RetryCount` | `int` | `retry_count,omitempty` | `—` |  |
| `RetryBackoffMs` | `int` | `retry_backoff_ms,omitempty` | `—` |  |

### `WorkflowNode`
**DB table:** `workflow_node`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `uint` | `id` | `id` | PK |
| `WorkflowID` | `uint` | `workflow_def_id` | `workflow_def_id` | NOT NULL |
| `Type` | `string` | `type` | `type` | NOT NULL |
| `Config` | `json.RawMessage` | `config` | `config` | default:'{}' |
| `OrderIndex` | `int` | `order_index` | `order_index` | default:0 |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |
| `UpdatedAt` | `time.Time` | `updated_at` | `updated_at` |  |

### `NodeConfig`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Condition` | `string` | `condition,omitempty` | `—` |  |
| `ApprovalRoles` | `string` | `approval_roles,omitempty` | `—` |  |
| `Command` | `string` | `command,omitempty` | `—` |  |
| `AgentID` | `string` | `agent_id,omitempty` | `—` |  |
| `DecisionPoint` | `string` | `decision_point,omitempty` | `—` |  |
| `EventTopic` | `string` | `event_topic,omitempty` | `—` |  |
| `TimeoutSeconds` | `int` | `timeout_seconds,omitempty` | `—` |  |

### `ApprovalResult`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Approved` | `bool` | `approved` | `—` |  |
| `Comment` | `string` | `comment,omitempty` | `—` |  |
| `Reviewer` | `string` | `reviewer,omitempty` | `—` |  |

---
_Auto-generated by `docgen`. Do not edit manually._
