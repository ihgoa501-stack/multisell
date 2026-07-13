CREATE TABLE sourcing_material_asset (
    id BIGSERIAL PRIMARY KEY,
    owner_id BIGINT NOT NULL REFERENCES "user"(id),
    sourcing_product_id BIGINT NOT NULL REFERENCES sourcing_1688_product(id),
    task_link_id BIGINT NOT NULL REFERENCES sourcing_1688_task_link(id),
    snapshot_id BIGINT NOT NULL REFERENCES sourcing_1688_snapshot(id),
    canonical_sku_mapping_id BIGINT REFERENCES sourcing_sku_mapping(id),
    role VARCHAR(16) NOT NULL CHECK (role IN ('main','gallery','sku','detail','video')),
    ordinal INTEGER NOT NULL CHECK (ordinal >= 0),
    source_url TEXT NOT NULL,
    source_sha256 VARCHAR(64) NOT NULL CHECK (source_sha256 ~ '^[0-9a-f]{64}$'),
    media_type VARCHAR(16) NOT NULL CHECK (media_type IN ('image','video')),
    mime_type VARCHAR(120) NOT NULL,
    byte_size BIGINT NOT NULL CHECK (byte_size >= 0),
    width INTEGER CHECK (width IS NULL OR width > 0),
    height INTEGER CHECK (height IS NULL OR height > 0),
    duration_ms BIGINT CHECK (duration_ms IS NULL OR duration_ms >= 0),
    status VARCHAR(16) NOT NULL DEFAULT 'active' CHECK (status IN ('active','archived')),
    used_at TIMESTAMPTZ,
    created_by BIGINT NOT NULL REFERENCES "user"(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX ux_sourcing_material_main ON sourcing_material_asset(owner_id, task_link_id) WHERE role='main' AND status='active';
CREATE UNIQUE INDEX ux_sourcing_material_order ON sourcing_material_asset(owner_id, task_link_id, role, ordinal) WHERE status='active';
CREATE INDEX idx_sourcing_material_task ON sourcing_material_asset(owner_id, sourcing_product_id, task_link_id, role, ordinal);

CREATE TABLE sourcing_material_rights_evidence (
    id BIGSERIAL PRIMARY KEY,
    asset_id BIGINT NOT NULL REFERENCES sourcing_material_asset(id),
    owner_id BIGINT NOT NULL REFERENCES "user"(id),
    version INTEGER NOT NULL CHECK (version > 0),
    status VARCHAR(16) NOT NULL CHECK (status IN ('pending','approved','rejected','revoked','expired')),
    license_scope TEXT NOT NULL,
    countries JSONB NOT NULL,
    channels JSONB NOT NULL,
    licensor VARCHAR(240) NOT NULL,
    source_uri TEXT NOT NULL,
    observed_at TIMESTAMPTZ NOT NULL,
    valid_until TIMESTAMPTZ,
    submitted_by BIGINT NOT NULL REFERENCES "user"(id),
    reviewed_by BIGINT REFERENCES "user"(id),
    reviewed_at TIMESTAMPTZ,
    review_note TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ux_sourcing_material_rights_version UNIQUE(asset_id, version)
);

CREATE TABLE sourcing_material_rendition (
    id BIGSERIAL PRIMARY KEY,
    asset_id BIGINT NOT NULL REFERENCES sourcing_material_asset(id),
    owner_id BIGINT NOT NULL REFERENCES "user"(id),
    image_processing_record_id BIGINT NOT NULL REFERENCES sourcing_1688_image_processing(id),
    created_by BIGINT NOT NULL REFERENCES "user"(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ux_sourcing_material_rendition UNIQUE(asset_id, image_processing_record_id)
);

CREATE FUNCTION protect_sourcing_material_source() RETURNS trigger AS $$
BEGIN
  IF NEW.owner_id <> OLD.owner_id OR NEW.sourcing_product_id <> OLD.sourcing_product_id OR NEW.task_link_id <> OLD.task_link_id OR
     NEW.snapshot_id <> OLD.snapshot_id OR NEW.source_url <> OLD.source_url OR NEW.source_sha256 <> OLD.source_sha256 OR
     NEW.media_type <> OLD.media_type OR NEW.mime_type <> OLD.mime_type OR NEW.byte_size <> OLD.byte_size OR
     NEW.width IS DISTINCT FROM OLD.width OR NEW.height IS DISTINCT FROM OLD.height OR NEW.duration_ms IS DISTINCT FROM OLD.duration_ms OR
     NEW.canonical_sku_mapping_id IS DISTINCT FROM OLD.canonical_sku_mapping_id OR NEW.role <> OLD.role THEN
    RAISE EXCEPTION 'sourcing material source identity and hash are immutable';
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE TRIGGER sourcing_material_source_immutable BEFORE UPDATE ON sourcing_material_asset FOR EACH ROW EXECUTE FUNCTION protect_sourcing_material_source();

CREATE FUNCTION protect_sourcing_material_rights_content() RETURNS trigger AS $$
BEGIN
  IF NEW.asset_id <> OLD.asset_id OR NEW.owner_id <> OLD.owner_id OR NEW.version <> OLD.version OR
     NEW.license_scope <> OLD.license_scope OR NEW.countries <> OLD.countries OR NEW.channels <> OLD.channels OR
     NEW.licensor <> OLD.licensor OR NEW.source_uri <> OLD.source_uri OR NEW.observed_at <> OLD.observed_at OR
     NEW.valid_until IS DISTINCT FROM OLD.valid_until OR NEW.submitted_by <> OLD.submitted_by THEN
    RAISE EXCEPTION 'material rights evidence content is immutable';
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE TRIGGER sourcing_material_rights_content_immutable BEFORE UPDATE ON sourcing_material_rights_evidence FOR EACH ROW EXECUTE FUNCTION protect_sourcing_material_rights_content();
