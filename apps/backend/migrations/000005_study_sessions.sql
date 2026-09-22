-- +goose Up
-- Study sessions of the adaptive tutor (PRODUCT_DOMAIN §12.1). The profile,
-- traps and exercises are the LLM-generated content stored as JSONB, mirroring
-- how practices.target_rules and analyses.fragments are stored. The user_id is
-- the portable identity (raw, not pseudonymized): sessions are owned data (A9),
-- unlike the derived analytics metrics.
CREATE TABLE study_sessions (
    id          uuid        PRIMARY KEY,
    user_id     uuid        NOT NULL,
    profile     jsonb       NOT NULL DEFAULT '[]'::jsonb,
    theory      text        NOT NULL,
    traps       jsonb       NOT NULL DEFAULT '[]'::jsonb,
    exercises   jsonb       NOT NULL DEFAULT '[]'::jsonb,
    status      text        NOT NULL,
    created_at  timestamptz NOT NULL
);

CREATE INDEX study_sessions_user_created_idx
    ON study_sessions (user_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS study_sessions;
