ALTER TABLE business_owner_decision ADD COLUMN input_payload JSONB NOT NULL DEFAULT '{}'::jsonb;
-- Historical selected decisions predate payload retention and are backfilled as {}.
-- New writes are canonicalized and SHA-verified by the service before immutable insert.
ALTER TABLE business_owner_decision ADD CONSTRAINT ck_business_owner_decision_payload_scope CHECK (
  (decision='selected' AND input_sha256 ~ '^[0-9a-f]{64}$')
  OR (decision<>'selected' AND input_payload = '{}'::jsonb AND input_sha256='')
);
