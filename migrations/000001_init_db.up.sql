CREATE TABLE "users"
(
    created_at      TIMESTAMPTZ NOT NULL,
    updated_at      TIMESTAMPTZ NOT NULL,
    id              BIGINT NOT NULL PRIMARY KEY,
    username        TEXT      NULL DEFAULT '',
    password        BYTEA     NULL DEFAULT '',
    progress_table  JSONB     NULL DEFAULT '{"tables": null}'::jsonb,
    deleted         BOOL      NULL DEFAULT FALSE
);

CREATE TABLE "changes"
(
    created_at      TIMESTAMPTZ NOT NULL,
    id           BIGSERIAL NOT NULL PRIMARY KEY,
    user_id      BIGINT    NOT NULL REFERENCES "users" (id) ON DELETE CASCADE,
    subject      TEXT      NOT NULL,
    control_event TEXT     NOT NULL,
    old_grade    TEXT      NOT NULL,
    new_grade    TEXT      NOT NULL
);