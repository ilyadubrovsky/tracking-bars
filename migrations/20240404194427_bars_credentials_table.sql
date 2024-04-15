-- +goose Up
-- +goose StatementBegin
CREATE TABLE bars_credentials (
    user_id BIGINT PRIMARY KEY REFERENCES users (id),
    username TEXT NOT NULL,
    password BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    deleted_at TIMESTAMPTZ NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE bars_credentials IF EXISTS;
-- +goose StatementEnd
