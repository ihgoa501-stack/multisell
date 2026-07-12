CREATE TABLE extension_pairing (
    id BIGSERIAL PRIMARY KEY,
    nonce_hash VARCHAR(64) NOT NULL UNIQUE,
    user_id BIGINT NOT NULL REFERENCES "user"(id),
    environment VARCHAR(40) NOT NULL CHECK (environment IN ('development', 'acceptance', 'production')),
    extension_id VARCHAR(128) NOT NULL DEFAULT '',
    device_id VARCHAR(128) NOT NULL DEFAULT '',
    browser_label VARCHAR(120) NOT NULL DEFAULT '',
    claim_secret_hash VARCHAR(64) NOT NULL DEFAULT '',
    status VARCHAR(24) NOT NULL CHECK (status IN ('waiting_for_browser', 'waiting_for_owner', 'confirmed', 'exchanged')),
    expires_at TIMESTAMPTZ NOT NULL,
    confirmed_at TIMESTAMPTZ,
    exchanged_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_extension_pairing_user_created ON extension_pairing(user_id, created_at DESC);

CREATE TABLE extension_device (
    device_id VARCHAR(128) PRIMARY KEY,
	installation_id VARCHAR(128) NOT NULL,
    user_id BIGINT NOT NULL REFERENCES "user"(id),
    extension_id VARCHAR(128) NOT NULL,
    environment VARCHAR(40) NOT NULL CHECK (environment IN ('development', 'acceptance', 'production')),
    browser_label VARCHAR(120) NOT NULL,
    secret_hash VARCHAR(64) NOT NULL,
    scope VARCHAR(120) NOT NULL CHECK (scope = 'sourcing1688.collect'),
    revoked_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_extension_device_user ON extension_device(user_id, created_at DESC);
CREATE UNIQUE INDEX ux_extension_device_install ON extension_device(user_id, environment, installation_id);

COMMENT ON TABLE extension_device IS 'Owner-confirmed browser extension identities. Secrets are stored only as SHA-256 digests and each device can be revoked independently.';
