-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS users (
  id        BIGSERIAL PRIMARY KEY,
  first_name  VARCHAR(200) NOT NULL,
  last_name  VARCHAR(200) NOT NULL,
  email     VARCHAR(100) NOT NULL UNIQUE,
  pass_hash BYTEA NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_email ON users (email);

CREATE TABLE IF NOT EXISTS apps (
  id     INTEGER PRIMARY KEY,
  name   VARCHAR(200) NOT NULL UNIQUE,
  secret TEXT NOT NULL UNIQUE
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_email;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS apps;
-- +goose StatementEnd