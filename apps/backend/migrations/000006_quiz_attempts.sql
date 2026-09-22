-- +goose Up
-- Quiz attempts of the active-practice loop (checklist 9.7). It is an
-- append-only ledger of the learner's quiz answers, keyed by the raw portable
-- user id (owned data, A9): it is a separate source of truth from error_metrics,
-- so it never participates in the refresh-aggregates reconciliation. There is no
-- foreign key to practices on purpose: attempts outlive the soft-deleted
-- practice they belonged to (like access_events), and are only removed by the
-- right-to-be-forgotten job.
CREATE TABLE quiz_attempts (
    id          uuid        PRIMARY KEY,
    user_id     uuid        NOT NULL,
    practice_id uuid        NOT NULL,
    correct     boolean     NOT NULL,
    created_at  timestamptz NOT NULL
);

CREATE INDEX quiz_attempts_user_created_idx
    ON quiz_attempts (user_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS quiz_attempts;
