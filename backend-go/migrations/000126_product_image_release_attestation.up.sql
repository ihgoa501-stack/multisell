CREATE TABLE product_image_rule_snapshots (
  id BIGSERIAL PRIMARY KEY, owner_id BIGINT NOT NULL, channel VARCHAR(64) NOT NULL,
  site VARCHAR(64) NOT NULL, locale VARCHAR(32) NOT NULL, category_id BIGINT NOT NULL,
  version BIGINT NOT NULL, rules JSONB NOT NULL, rules_sha256 VARCHAR(64) NOT NULL,
  effective_at TIMESTAMPTZ NOT NULL, expires_at TIMESTAMPTZ,
  idempotency_key VARCHAR(100) NOT NULL, request_sha256 VARCHAR(64) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(owner_id, channel, site, locale, category_id, version),
  UNIQUE(owner_id, idempotency_key),
  CHECK (rules_sha256 ~ '^[0-9a-f]{64}$'),
  CHECK (request_sha256 ~ '^[0-9a-f]{64}$'),
  CHECK (expires_at IS NULL OR expires_at > effective_at)
);

CREATE TABLE product_image_set_decisions (
  id BIGSERIAL PRIMARY KEY, owner_id BIGINT NOT NULL,
  image_set_id BIGINT NOT NULL REFERENCES product_image_sets(id) ON DELETE RESTRICT,
  image_set_version INTEGER NOT NULL, set_manifest_sha VARCHAR(64) NOT NULL,
  decision VARCHAR(16) NOT NULL CHECK (decision IN ('approved','rejected')),
  reason TEXT NOT NULL, idempotency_key VARCHAR(100) NOT NULL,
  request_sha256 VARCHAR(64) NOT NULL, decided_at TIMESTAMPTZ NOT NULL,
  UNIQUE(owner_id, idempotency_key),
  CHECK (set_manifest_sha ~ '^[0-9a-f]{64}$'),
  CHECK (request_sha256 ~ '^[0-9a-f]{64}$')
);
CREATE INDEX idx_product_image_set_decisions_set ON product_image_set_decisions(owner_id,image_set_id,id DESC);

CREATE TABLE product_image_release_attestations (
  id BIGSERIAL PRIMARY KEY, owner_id BIGINT NOT NULL, listing_id BIGINT NOT NULL REFERENCES product_listing(id),
  product_id BIGINT NOT NULL REFERENCES product(id), platform_id BIGINT NOT NULL REFERENCES platform(id),
  platform_account_id BIGINT NOT NULL REFERENCES platform_integration_account(id),
  channel VARCHAR(64) NOT NULL, site VARCHAR(64) NOT NULL, locale VARCHAR(32) NOT NULL, category_id BIGINT NOT NULL,
  listing_snapshot_sha VARCHAR(64) NOT NULL,
  image_set_id BIGINT NOT NULL REFERENCES product_image_sets(id), image_set_version INTEGER NOT NULL,
  set_manifest_sha VARCHAR(64) NOT NULL, media_manifest JSONB NOT NULL, media_manifest_sha VARCHAR(64) NOT NULL,
  rule_snapshot_id BIGINT NOT NULL REFERENCES product_image_rule_snapshots(id), rule_snapshot_sha VARCHAR(64) NOT NULL,
  set_decision_id BIGINT NOT NULL REFERENCES product_image_set_decisions(id),
  rights_manifest_sha VARCHAR(64) NOT NULL, review_manifest_sha VARCHAR(64) NOT NULL,
  claims JSONB NOT NULL, claims_sha256 VARCHAR(64) NOT NULL, signature VARCHAR(64) NOT NULL,
  key_id VARCHAR(64) NOT NULL, nonce VARCHAR(64) NOT NULL UNIQUE,
  status VARCHAR(16) NOT NULL CHECK(status IN ('issued','consumed','revoked')),
  idempotency_key VARCHAR(100) NOT NULL, request_sha256 VARCHAR(64) NOT NULL,
  issued_at TIMESTAMPTZ NOT NULL, expires_at TIMESTAMPTZ NOT NULL,
  consumed_at TIMESTAMPTZ, consumed_by_type VARCHAR(64), consumed_by_id BIGINT, revoked_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(), UNIQUE(owner_id,idempotency_key),
  CHECK (expires_at > issued_at),
  CHECK ((status='consumed') = (consumed_at IS NOT NULL)),
  CHECK ((status='revoked') = (revoked_at IS NOT NULL)),
  CHECK (listing_snapshot_sha ~ '^[0-9a-f]{64}$' AND set_manifest_sha ~ '^[0-9a-f]{64}$' AND
    media_manifest_sha ~ '^[0-9a-f]{64}$' AND rule_snapshot_sha ~ '^[0-9a-f]{64}$' AND
    rights_manifest_sha ~ '^[0-9a-f]{64}$' AND review_manifest_sha ~ '^[0-9a-f]{64}$' AND
    claims_sha256 ~ '^[0-9a-f]{64}$' AND signature ~ '^[0-9a-f]{64}$' AND
    nonce ~ '^[0-9a-f]{64}$' AND request_sha256 ~ '^[0-9a-f]{64}$')
);
CREATE INDEX idx_product_image_attestation_target ON product_image_release_attestations(owner_id,listing_id,status,expires_at);

CREATE TABLE product_image_release_attestation_items (
  id BIGSERIAL PRIMARY KEY, attestation_id BIGINT NOT NULL REFERENCES product_image_release_attestations(id) ON DELETE RESTRICT,
  ordinal INTEGER NOT NULL CHECK(ordinal > 0), role VARCHAR(32) NOT NULL, task_id BIGINT NOT NULL REFERENCES product_image_tasks(id),
  blob_id VARCHAR(64) NOT NULL, sha256 VARCHAR(64) NOT NULL, mime VARCHAR(64) NOT NULL,
  width INTEGER NOT NULL CHECK(width > 0), height INTEGER NOT NULL CHECK(height > 0),
  UNIQUE(attestation_id,ordinal), CHECK(blob_id ~ '^[0-9a-f]{64}$' AND sha256 ~ '^[0-9a-f]{64}$' AND blob_id=sha256)
);

COMMENT ON TABLE product_image_release_attestations IS 'Signed immutable proof of exact Owner-approved media; issuance is not publication and consumption authorizes one bound external-write attempt only.';
