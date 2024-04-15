-- +goose Up
-- +goose StatementBegin
CREATE TABLE bars_credentials (
    user_id BIGINT PRIMARY KEY REFERENCES users (id),
    username TEXT NOT NULL,
    password BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    deleted_at TIMESTAMPTZ NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS bars_credentials;
-- +goose StatementEnd
