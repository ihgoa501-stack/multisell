CREATE TABLE IF NOT EXISTS auth_refresh_session (
    token_id VARCHAR(64) PRIMARY KEY,
    family_id VARCHAR(64) NOT NULL,
    user_id BIGINT NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    revoked_at TIMESTAMP WITH TIME ZONE,
    replaced_by VARCHAR(64) NOT NULL DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_auth_refresh_session_family
    ON auth_refresh_session(family_id);
CREATE INDEX IF NOT EXISTS idx_auth_refresh_session_user
    ON auth_refresh_session(user_id);
CREATE INDEX IF NOT EXISTS idx_auth_refresh_session_expiry
    ON auth_refresh_session(expires_at);
CREATE INDEX IF NOT EXISTS idx_auth_refresh_session_active
    ON auth_refresh_session(user_id, revoked_at, expires_at);
