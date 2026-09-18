-- +goose Up
CREATE TABLE practices (
    id           uuid        PRIMARY KEY,
    user_id      uuid        NOT NULL,
    source_text  text        NOT NULL,
    draft_text   text        NOT NULL,
    target_rules jsonb       NOT NULL DEFAULT '[]'::jsonb,
    status       text        NOT NULL,
    created_at   timestamptz NOT NULL,
    updated_at   timestamptz NOT NULL,
    deleted_at   timestamptz
);

CREATE INDEX practices_user_created_idx
    ON practices (user_id, created_at DESC)
    WHERE deleted_at IS NULL;

CREATE TABLE analyses (
    id            uuid        PRIMARY KEY,
    practice_id   uuid        NOT NULL UNIQUE REFERENCES practices (id),
    fragments     jsonb       NOT NULL DEFAULT '[]'::jsonb,
    model         text        NOT NULL,
    model_version text        NOT NULL,
    status        text        NOT NULL,
    created_at    timestamptz NOT NULL
);

CREATE INDEX analyses_practice_idx ON analyses (practice_id);

CREATE TABLE error_metrics (
    user_id      uuid        NOT NULL,
    code         text        NOT NULL,
    "window"     text        NOT NULL,
    count        integer     NOT NULL DEFAULT 0,
    last_seen_at timestamptz NOT NULL,
    PRIMARY KEY (user_id, code, "window")
);

CREATE TABLE outbox_events (
    id           uuid        PRIMARY KEY,
    event_type   text        NOT NULL,
    payload      jsonb       NOT NULL,
    created_at   timestamptz NOT NULL,
    published_at timestamptz,
    attempts     integer     NOT NULL DEFAULT 0
);

CREATE INDEX outbox_events_unpublished_idx
    ON outbox_events (created_at)
    WHERE published_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS outbox_events;
DROP TABLE IF EXISTS error_metrics;
DROP TABLE IF EXISTS analyses;
DROP TABLE IF EXISTS practices;
