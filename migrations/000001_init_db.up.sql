CREATE TABLE "users"
(
    id              BIGSERIAL NOT NULL PRIMARY KEY,
    username        TEXT      NULL DEFAULT '',
    password        BYTEA     NULL DEFAULT '',
    progress_table  JSONB          DEFAULT '{}',
    deleted         BOOL           DEFAULT FALSE
);

CREATE TABLE "changes"
(
    id           BIGSERIAL NOT NULL PRIMARY KEY,
    user_id      BIGINT    NOT NULL REFERENCES "users" (id),
    subject      TEXT      NOT NULL,
    control_event TEXT     NOT NULL,
    old_grade    TEXT      NOT NULL,
    new_grade    TEXT      NOT NULL
)
