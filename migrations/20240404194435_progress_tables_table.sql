-- +goose Up
-- +goose StatementBegin
CREATE TABLE progress_tables (
    user_id BIGINT PRIMARY KEY REFERENCES bars_credentials (user_id),
    progress_table JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS progress_tables;
-- +goose StatementEnd
