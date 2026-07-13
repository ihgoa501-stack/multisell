ALTER TABLE business_owner_decision DROP CONSTRAINT IF EXISTS ck_business_owner_decision_payload_scope;
ALTER TABLE business_owner_decision DROP COLUMN IF EXISTS input_payload;
