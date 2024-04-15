-- +goose Up
-- +goose StatementBegin
CREATE TYPE "user" AS (
    id BIGINT
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TYPE IF EXISTS "user";
-- +goose StatementEnd
