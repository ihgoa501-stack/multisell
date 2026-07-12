DROP INDEX IF EXISTS idx_approval_request_reviewer_user;
DROP INDEX IF EXISTS idx_approval_request_requester_user;

ALTER TABLE approval_request
    DROP COLUMN IF EXISTS reviewer_user_id,
    DROP COLUMN IF EXISTS requester_user_id;
