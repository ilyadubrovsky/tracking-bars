-- +goose Up
-- +goose StatementBegin
CREATE TYPE "user" AS (
    id BIGINT,
    created_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TYPE IF EXISTS "user";
-- +goose StatementEnd
