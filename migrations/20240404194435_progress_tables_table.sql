-- +goose Up
-- +goose StatementBegin
CREATE TABLE progress_tables (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES bars_credentials (user_id),
    progress_table JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE progress_tables IF EXISTS;
-- +goose StatementEnd
