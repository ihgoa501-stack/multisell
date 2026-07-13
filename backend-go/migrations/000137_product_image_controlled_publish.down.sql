DROP TABLE IF EXISTS product_image_publish_attempts;

ALTER TABLE product_image_release_attestations
  DROP CONSTRAINT IF EXISTS product_image_release_attestations_claim_check,
  DROP CONSTRAINT IF EXISTS product_image_release_attestations_revoked_check,
  DROP CONSTRAINT IF EXISTS product_image_release_attestations_consumed_check,
  DROP CONSTRAINT IF EXISTS product_image_release_attestations_status_check;

ALTER TABLE product_image_release_attestations
  ADD CONSTRAINT product_image_release_attestations_status_check
    CHECK (status IN ('issued','consumed','revoked')),
  ADD CONSTRAINT product_image_release_attestations_check1
    CHECK ((status='consumed') = (consumed_at IS NOT NULL)),
  ADD CONSTRAINT product_image_release_attestations_check2
    CHECK ((status='revoked') = (revoked_at IS NOT NULL));
