-- +goose Up
CREATE TABLE users
(
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    username      VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    settings      TEXT         NOT NULL DEFAULT '{}',
    created_at    TIMESTAMP    NOT NULL,
    updated_at    TIMESTAMP    NOT NULL,
    UNIQUE (username)
);

CREATE TABLE sessions
(
    token      CHAR(64)  NOT NULL PRIMARY KEY,
    user_id    INTEGER   NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL
);
CREATE INDEX idx_sessions_user_id ON sessions (user_id);
CREATE INDEX idx_sessions_expires_at ON sessions (expires_at);

CREATE TABLE notes
(
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    title      VARCHAR(255) NOT NULL,
    content    TEXT         NOT NULL,
    checksum   CHAR(16)     NOT NULL,
    created_at TIMESTAMP    NOT NULL,
    updated_at TIMESTAMP    NOT NULL
);

CREATE TABLE tags
(
    name       VARCHAR(255) NOT NULL PRIMARY KEY,
    created_at TIMESTAMP    NOT NULL
);

CREATE TABLE note_tags
(
    note_id  INTEGER      NOT NULL REFERENCES notes (id) ON DELETE CASCADE,
    tag_name VARCHAR(255) NOT NULL REFERENCES tags (name) ON DELETE CASCADE,
    PRIMARY KEY (note_id, tag_name)
);
CREATE INDEX idx_note_tags_tag_name ON note_tags (tag_name);

-- +goose Down
DROP TABLE note_tags;
DROP TABLE tags;
DROP TABLE notes;
DROP TABLE sessions;
DROP TABLE users;
