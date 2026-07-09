# Module: `entropy`

Package: `backend-go/internal/domain/entropy/`

**Base mount prefix:** `/api/v1`

## API Routes

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/entropy` | `h.GetSummary` |
| `GET` | `/api/v1/entropy/changelog` | `h.GetChangeLog` |
| `POST` | `/api/v1/entropy/defense` | `h.RunDefenses` |
| `GET` | `/api/v1/entropy/health` | `h.GetHealthScores` |
| `GET` | `/api/v1/entropy/spc` | `h.GetSpcStatus` |

## Models

### `SpcControlLimit`
**DB table:** `spc_control_limit`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `UserID` | `int64` | `user_id` | `user_id` | NOT NULL |
| `AgentID` | `string` | `agent_id` | `agent_id` | NOT NULL |
| `DecisionPoint` | `string` | `decision_point` | `decision_point` | NOT NULL |
| `MetricName` | `string` | `metric_name` | `metric_name` | NOT NULL |
| `BaselineMean` | `float64` | `baseline_mean` | `baseline_mean` | NOT NULL |
| `BaselineStddev` | `float64` | `baseline_stddev` | `baseline_stddev` | NOT NULL |
| `BaselineSamples` | `int` | `baseline_samples` | `baseline_samples` | NOT NULL |
| `UCL` | `float64` | `ucl` | `ucl` | NOT NULL |
| `LCL` | `float64` | `lcl` | `lcl` | NOT NULL |
| `UWL` | `float64` | `uwl` | `uwl` | NOT NULL |
| `LWL` | `float64` | `lwl` | `lwl` | NOT NULL |
| `ConsecutiveSameSide` | `int` | `consecutive_same_side` | `consecutive_same_side` | default:0 |
| `LastBreachAt` | `*time.Time` | `last_breach_at,omitempty` | `last_breach_at` |  |
| `BaselineRecalcAt` | `time.Time` | `baseline_recalc_at` | `baseline_recalc_at` | NOT NULL |
| `NextRecalcAt` | `time.Time` | `next_recalc_at` | `next_recalc_at` | NOT NULL |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |
| `UpdatedAt` | `time.Time` | `updated_at` | `updated_at` |  |

### `PersonalRule`
**DB table:** `personal_rule`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `UserID` | `int64` | `user_id` | `user_id` | NOT NULL |
| `AgentID` | `string` | `agent_id` | `agent_id` | NOT NULL |
| `DecisionPoint` | `string` | `decision_point` | `decision_point` | NOT NULL |
| `RuleType` | `string` | `rule_type` | `rule_type` | NOT NULL |
| `RuleName` | `string` | `rule_name` | `rule_name` | NOT NULL |
| `RuleCondition` | `string` | `rule_condition` | `rule_condition` | NOT NULL |
| `RuleAction` | `string` | `rule_action` | `rule_action` | NOT NULL |
| `Priority` | `int` | `priority` | `priority` | default:100 |
| `Status` | `string` | `status` | `status` | default:active |
| `Confidence` | `float64` | `confidence` | `confidence` | default:0 |
| `TimesApplied` | `int` | `times_applied` | `times_applied` | default:0 |
| `TimesOverridden` | `int` | `times_overridden` | `times_overridden` | default:0 |
| `LastAppliedAt` | `*time.Time` | `last_applied_at,omitempty` | `last_applied_at` |  |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |
| `UpdatedAt` | `time.Time` | `updated_at` | `updated_at` |  |

### `AgentDecision`
**DB table:** `agent_decision`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `UserID` | `int64` | `user_id` | `user_id` | NOT NULL |
| `AgentID` | `string` | `agent_id` | `agent_id` | NOT NULL |
| `DecisionPoint` | `string` | `decision_point` | `decision_point` | NOT NULL |
| `ContextJSON` | `string` | `context_json` | `context_json` | NOT NULL |
| `AgentOutput` | `string` | `agent_output` | `agent_output` | NOT NULL |
| `FinalDecision` | `string` | `final_decision` | `final_decision` | NOT NULL |
| `UserAction` | `string` | `user_action` | `user_action` | NOT NULL |
| `UserOverrides` | `*string` | `user_overrides,omitempty` | `user_overrides` |  |
| `UserFeedback` | `*string` | `user_feedback,omitempty` | `user_feedback` |  |
| `RulesApplied` | `*string` | `rules_applied,omitempty` | `rules_applied` |  |
| `RuleOverrides` | `int` | `rule_overrides` | `rule_overrides` | default:0 |
| `EvolutionStage` | `string` | `evolution_stage` | `evolution_stage` | NOT NULL |
| `Confidence` | `*float64` | `confidence,omitempty` | `confidence` |  |
| `ResponseTimeMs` | `*int` | `response_time_ms,omitempty` | `response_time_ms` |  |
| `TokenCount` | `*int` | `token_count,omitempty` | `token_count` |  |
| `SessionID` | `string` | `session_id` | `session_id` | NOT NULL |
| `EpisodeID` | `*int64` | `episode_id,omitempty` | `episode_id` |  |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |

### `RuleMarkChange`
**DB table:** `rule_mark_change`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `TargetType` | `string` | `target_type` | `target_type` | NOT NULL |
| `TargetID` | `int64` | `target_id` | `target_id` | NOT NULL |
| `FieldPath` | `string` | `field_path` | `field_path` | NOT NULL |
| `OldValue` | `*string` | `old_value,omitempty` | `old_value` |  |
| `NewValue` | `string` | `new_value` | `new_value` | NOT NULL |
| `SourceType` | `string` | `source_type` | `source_type` | NOT NULL |
| `SourceID` | `*string` | `source_id,omitempty` | `source_id` |  |
| `ChangeSummary` | `string` | `change_summary` | `change_summary` | NOT NULL |
| `ParentChangeID` | `*int64` | `parent_change_id,omitempty` | `parent_change_id` |  |
| `RelatedDecisionIDs` | `*string` | `related_decision_ids,omitempty` | `related_decision_ids` |  |
| `ContextJSON` | `*string` | `context_json,omitempty` | `context_json` |  |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |

### `DefenseResult`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Name` | `string` | `name` | `—` |  |
| `Status` | `string` | `status` | `—` |  |
| `Action` | `string` | `action` | `—` |  |
| `Count` | `int` | `count` | `—` |  |
| `Message` | `string` | `message,omitempty` | `—` |  |

### `EntropySummary`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `TotalRules` | `int` | `total_rules` | `—` |  |
| `ActiveRules` | `int` | `active_rules` | `—` |  |
| `ShadowRules` | `int` | `shadow_rules` | `—` |  |
| `AvgHealthScore` | `float64` | `avg_health_score` | `—` |  |
| `UnhealthyRuleCount` | `int` | `unhealthy_rule_count` | `—` |  |
| `WarningRuleCount` | `int` | `warning_rule_count` | `—` |  |
| `PendingMergeCount` | `int` | `pending_merge_count` | `—` |  |
| `RecentChangesCount` | `int` | `recent_changes_count` | `—` |  |
| `ConflictsCount` | `int` | `conflicts_count` | `—` |  |
| `SystemEntropyIndex` | `float64` | `system_entropy_index` | `—` |  |

### `RuleHealthScore`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `RuleID` | `int64` | `rule_id` | `—` |  |
| `RuleName` | `string` | `rule_name` | `—` |  |
| `RuleType` | `string` | `rule_type` | `—` |  |
| `AgentID` | `string` | `agent_id` | `—` |  |
| `DecisionPoint` | `string` | `decision_point` | `—` |  |
| `Status` | `string` | `status` | `—` |  |
| `Score` | `float64` | `score` | `—` |  |
| `Dimensions` | `HealthDimensions` | `dimensions` | `—` |  |
| `TimesApplied` | `int` | `times_applied` | `—` |  |
| `TimesOverridden` | `int` | `times_overridden` | `—` |  |
| `OverrideRate` | `float64` | `override_rate` | `—` |  |
| `DaysSinceLastApplied` | `*int` | `days_since_last_applied,omitempty` | `—` |  |
| `Confidence` | `float64` | `confidence` | `—` |  |
| `RiskLevel` | `string` | `risk_level` | `—` |  |

### `HealthDimensions`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Acceptance` | `float64` | `acceptance` | `—` |  |
| `Confidence` | `float64` | `confidence` | `—` |  |
| `Freshness` | `float64` | `freshness` | `—` |  |
| `Frequency` | `float64` | `frequency` | `—` |  |
| `TypeWeight` | `float64` | `type_weight` | `—` |  |

### `HealthSummary`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `TotalRules` | `int` | `total_rules` | `—` |  |
| `ActiveRules` | `int` | `active_rules` | `—` |  |
| `ShadowRules` | `int` | `shadow_rules` | `—` |  |
| `AvgHealthScore` | `float64` | `avg_health_score` | `—` |  |
| `UnhealthyCount` | `int` | `unhealthy_count` | `—` |  |
| `HealthyCount` | `int` | `healthy_count` | `—` |  |
| `WarningCount` | `int` | `warning_count` | `—` |  |

### `SpcStatus`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `AgentID` | `string` | `agent_id` | `—` |  |
| `DecisionPoint` | `string` | `decision_point` | `—` |  |
| `MetricName` | `string` | `metric_name` | `—` |  |
| `CurrentValue` | `float64` | `current_value,omitempty` | `—` |  |
| `BaselineMean` | `float64` | `baseline_mean` | `—` |  |
| `UCL` | `float64` | `ucl` | `—` |  |
| `LCL` | `float64` | `lcl` | `—` |  |
| `UWL` | `float64` | `uwl` | `—` |  |
| `LWL` | `float64` | `lwl` | `—` |  |
| `BaselineSamples` | `int` | `baseline_samples` | `—` |  |
| `ConsecutiveSameSide` | `int` | `consecutive_same_side` | `—` |  |
| `IsOutOfControl` | `bool` | `is_out_of_control` | `—` |  |
| `IsWarning` | `bool` | `is_warning` | `—` |  |
| `Alerts` | `[]SpcAlert` | `alerts,omitempty` | `—` |  |
| `LastBreachAt` | `*time.Time` | `last_breach_at,omitempty` | `—` |  |
| `NextRecalcAt` | `*time.Time` | `next_recalc_at,omitempty` | `—` |  |

### `SpcAlert`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Level` | `string` | `level` | `—` |  |
| `Message` | `string` | `message` | `—` |  |

### `DefenseSummary`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Actions` | `*ast.StructType` | `actions` | `—` |  |
| `TotalAffected` | `int` | `total_affected` | `—` |  |
| `MarkChanges` | `[]ChangeLogEntry` | `mark_changes,omitempty` | `—` |  |
| `DuplicatesFound` | `int` | `duplicates_found` | `—` |  |
| `MergeCandidates` | `[]MergeCandidate` | `merge_candidates,omitempty` | `—` |  |

### `ChangeLogEntry`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `—` |  |
| `TargetType` | `string` | `target_type` | `—` |  |
| `TargetID` | `int64` | `target_id` | `—` |  |
| `FieldPath` | `string` | `field_path` | `—` |  |
| `OldValue` | `*string` | `old_value,omitempty` | `—` |  |
| `NewValue` | `string` | `new_value` | `—` |  |
| `SourceType` | `string` | `source_type` | `—` |  |
| `SourceID` | `*string` | `source_id,omitempty` | `—` |  |
| `ChangeSummary` | `string` | `change_summary` | `—` |  |
| `CreatedAt` | `string` | `created_at,omitempty` | `—` |  |

### `MergeCandidate`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `KeepID` | `int64` | `keep_id` | `—` |  |
| `KeepName` | `string` | `keep_name` | `—` |  |
| `RemoveID` | `int64` | `remove_id` | `—` |  |
| `RemoveName` | `string` | `remove_name` | `—` |  |
| `Similarity` | `float64` | `similarity` | `—` |  |

### `DuplicatePair`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Keep` | `*PersonalRule` | `` | `—` |  |
| `Remove` | `*PersonalRule` | `` | `—` |  |
| `Similarity` | `float64` | `` | `—` |  |

---
_Auto-generated by `docgen`. Do not edit manually._
