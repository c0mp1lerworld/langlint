-- +goose Up
CREATE TABLE access_events (
    id            uuid        PRIMARY KEY,
    user_id       uuid        NOT NULL,
    action        text        NOT NULL,
    resource_type text,
    resource_id   text,
    occurred_at   timestamptz NOT NULL
);

CREATE INDEX access_events_user_occurred_idx
    ON access_events (user_id, occurred_at DESC);

-- +goose Down
DROP TABLE IF EXISTS access_events;
