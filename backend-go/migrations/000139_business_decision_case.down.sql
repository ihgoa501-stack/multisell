DROP TRIGGER IF EXISTS trg_business_owner_immutable ON business_owner_decision;
DROP TRIGGER IF EXISTS trg_business_ai_immutable ON business_ai_recommendation;
DROP TRIGGER IF EXISTS trg_business_snapshot_immutable ON business_decision_fact_snapshot;
DROP TRIGGER IF EXISTS trg_business_case_immutable ON business_decision_case;
DROP TABLE IF EXISTS business_owner_decision;
DROP TABLE IF EXISTS business_ai_recommendation;
DROP TABLE IF EXISTS business_decision_fact_snapshot;
DROP TABLE IF EXISTS business_decision_case;
DROP FUNCTION IF EXISTS guard_business_decision_immutable();
