ALTER TABLE approval_request
    ADD COLUMN IF NOT EXISTS requester_user_id BIGINT,
    ADD COLUMN IF NOT EXISTS reviewer_user_id BIGINT;

CREATE INDEX IF NOT EXISTS idx_approval_request_requester_user
    ON approval_request(requester_user_id);

CREATE INDEX IF NOT EXISTS idx_approval_request_reviewer_user
    ON approval_request(reviewer_user_id);
