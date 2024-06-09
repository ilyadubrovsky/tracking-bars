-- +goose Up
-- +goose StatementBegin
ALTER TABLE progress_tables
DROP CONSTRAINT progress_tables_user_id_fkey;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE bars_credentials
DROP CONSTRAINT bars_credentials_user_id_fkey;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE progress_tables
ADD CONSTRAINT progress_tables_user_id_fkey
FOREIGN KEY (user_id) REFERENCES bars_credentials (user_id);
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE bars_credentials
ADD CONSTRAINT bars_credentials_user_id_fkey
FOREIGN KEY (user_id) REFERENCES users (id);
-- +goose StatementEnd