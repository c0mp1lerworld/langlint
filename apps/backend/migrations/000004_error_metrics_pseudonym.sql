-- +goose Up
-- Analytics are keyed by a pseudonymized identifier, never the raw user id (A8).
-- error_metrics is a derived table: refresh-aggregates rebuilds it from the
-- source of truth, so widening user_id to text is safe.
ALTER TABLE error_metrics ALTER COLUMN user_id TYPE text USING user_id::text;

-- +goose Down
DELETE FROM error_metrics;
ALTER TABLE error_metrics ALTER COLUMN user_id TYPE uuid USING user_id::uuid;
