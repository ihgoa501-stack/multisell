ALTER TABLE product_image_release_attestations
  DROP CONSTRAINT product_image_release_attestations_status_check,
  DROP CONSTRAINT product_image_release_attestations_check1,
  DROP CONSTRAINT product_image_release_attestations_check2;

ALTER TABLE product_image_release_attestations
  ADD CONSTRAINT product_image_release_attestations_status_check
    CHECK (status IN ('issued','claimed','reconcile_required','consumed','revoked')),
  ADD CONSTRAINT product_image_release_attestations_consumed_check
    CHECK ((status='consumed') = (consumed_at IS NOT NULL)),
  ADD CONSTRAINT product_image_release_attestations_revoked_check
    CHECK ((status='revoked') = (revoked_at IS NOT NULL)),
  ADD CONSTRAINT product_image_release_attestations_claim_check
    CHECK ((status IN ('claimed','reconcile_required','consumed')) =
           (consumed_by_type IS NOT NULL AND consumed_by_type<>'' AND consumed_by_id IS NOT NULL));

CREATE TABLE product_image_publish_attempts (
  id BIGSERIAL PRIMARY KEY,
  owner_id BIGINT NOT NULL,
  attestation_id BIGINT NOT NULL REFERENCES product_image_release_attestations(id) ON DELETE RESTRICT,
  listing_id BIGINT NOT NULL REFERENCES product_listing(id) ON DELETE RESTRICT,
  platform_id BIGINT NOT NULL REFERENCES platform(id) ON DELETE RESTRICT,
  platform_account_id BIGINT NOT NULL REFERENCES platform_integration_account(id) ON DELETE RESTRICT,
  channel VARCHAR(64) NOT NULL,
  media_manifest_sha VARCHAR(64) NOT NULL,
  status VARCHAR(32) NOT NULL CHECK (status IN ('pending','calling','reconcile_required','succeeded','failed_terminal','unsupported')),
  idempotency_key VARCHAR(100) NOT NULL,
  request_sha256 VARCHAR(64) NOT NULL,
  remote_reference TEXT,
  receipt_evidence JSONB,
  receipt_sha256 VARCHAR(64),
  reconcile_evidence JSONB,
  failure_code VARCHAR(64),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  claimed_at TIMESTAMPTZ,
  external_called_at TIMESTAMPTZ,
  reconcile_required_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ,
  UNIQUE(owner_id,idempotency_key),
  CHECK (media_manifest_sha ~ '^[0-9a-f]{64}$' AND request_sha256 ~ '^[0-9a-f]{64}$'),
  CHECK (receipt_sha256 IS NULL OR receipt_sha256 ~ '^[0-9a-f]{64}$'),
  CHECK ((status IN ('calling','reconcile_required','succeeded','failed_terminal')) = (claimed_at IS NOT NULL)),
  CHECK ((status IN ('reconcile_required','succeeded','failed_terminal')) = (external_called_at IS NOT NULL)),
  CHECK ((status='reconcile_required') = (reconcile_required_at IS NOT NULL)),
  CHECK ((status IN ('succeeded','failed_terminal')) = (completed_at IS NOT NULL)),
  CHECK ((status='succeeded') = (remote_reference IS NOT NULL AND remote_reference<>'' AND receipt_sha256 IS NOT NULL))
);

CREATE INDEX idx_product_image_publish_attempts_owner_status
  ON product_image_publish_attempts(owner_id,status,id DESC);
CREATE UNIQUE INDEX ux_product_image_publish_active_attestation
  ON product_image_publish_attempts(attestation_id)
  WHERE status <> 'unsupported';

COMMENT ON TABLE product_image_publish_attempts IS
  'One-shot controlled-byte media delivery attempts. URL-based legacy adapters are not eligible; uncertain calls require explicit reconciliation and are never retried automatically.';
