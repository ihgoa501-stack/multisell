CREATE TABLE IF NOT EXISTS approval_request (
    id BIGSERIAL PRIMARY KEY,
    product_id BIGINT NOT NULL,
    request_type VARCHAR(32) NOT NULL,
    requester VARCHAR(64) NOT NULL,
    reviewer VARCHAR(64),
    status VARCHAR(16) NOT NULL DEFAULT 'pending',
    old_value TEXT,
    new_value TEXT,
    reason TEXT,
    review_note TEXT,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_approval_status ON approval_request(status);
CREATE INDEX idx_approval_request_type ON approval_request(request_type);
CREATE INDEX idx_approval_product_id ON approval_request(product_id);
CREATE INDEX idx_approval_requester ON approval_request(requester);
CREATE INDEX idx_approval_created_at ON approval_request(created_at);
