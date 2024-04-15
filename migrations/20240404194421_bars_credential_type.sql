-- +goose Up
-- +goose StatementBegin
CREATE TYPE bars_credential AS (
    user_id BIGINT,
    username TEXT,
    password BYTEA,
    created_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TYPE IF EXISTS bars_credential;
-- +goose StatementEnd
