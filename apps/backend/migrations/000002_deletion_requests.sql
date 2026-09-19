-- +goose Up
CREATE TABLE deletion_requests (
    id           uuid        PRIMARY KEY,
    user_id      uuid        NOT NULL,
    requested_at timestamptz NOT NULL,
    executed_at  timestamptz
);

CREATE INDEX deletion_requests_pending_idx
    ON deletion_requests (requested_at)
    WHERE executed_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS deletion_requests;
