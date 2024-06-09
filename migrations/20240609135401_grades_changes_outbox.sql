-- +goose Up
-- +goose StatementBegin
CREATE TABLE grades_changes_outbox (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    grades_change JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS grades_changes_outbox;
-- +goose StatementEnd
