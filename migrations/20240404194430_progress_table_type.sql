-- +goose Up
-- +goose StatementBegin
CREATE TYPE progress_table AS (
    id UUID,
    user_id BIGINT,
    progress_table JSONB,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TYPE IF EXISTS progress_table;
-- +goose StatementEnd
