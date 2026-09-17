-- Schema of record for this service.
--
-- This project does NOT use GORM auto-migration. Every table is defined here as
-- explicit DDL and applied out of band (see `make db-schema`, or your migration
-- process in real environments).
--
-- Conventions:
--   * Primary keys are BIGSERIAL (Postgres auto-increment), mapped to int64 in Go.
--   * Name unique constraints explicitly (e.g. uq_<table>_<col>) so the
--     repository can map a violation to a specific domain error.
--   * Timestamps default to now() and are set by the database.
--
-- Add a CREATE TABLE per model below as you build each domain. Example:
--
-- CREATE TABLE IF NOT EXISTS users (
--     id         BIGSERIAL PRIMARY KEY,
--     email      VARCHAR(255) NOT NULL,
--     name       VARCHAR(255) NOT NULL,
--     created_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
--     updated_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
--     CONSTRAINT uq_users_email UNIQUE (email)
-- );

-- ── Expense sharing ─────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS users (
    id         BIGSERIAL PRIMARY KEY,
    name       VARCHAR(255) NOT NULL,
    email      VARCHAR(255) NOT NULL,
    phone      VARCHAR(32)  NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT uq_users_email UNIQUE (email),
    CONSTRAINT uq_users_phone UNIQUE (phone)
);

CREATE TABLE IF NOT EXISTS expenses (
    id                 BIGSERIAL PRIMARY KEY,
    created_by_user_id BIGINT       NOT NULL REFERENCES users(id),
    name               VARCHAR(255) NOT NULL,
    amount_paisa       BIGINT       NOT NULL CHECK (amount_paisa >= 0),
    split_type         VARCHAR(16)  NOT NULL,
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS expense_splits (
    id          BIGSERIAL PRIMARY KEY,
    expense_id  BIGINT NOT NULL REFERENCES expenses(id),
    user_id     BIGINT NOT NULL REFERENCES users(id),
    owed_paisa  BIGINT NOT NULL CHECK (owed_paisa >= 0),
    paid_paisa  BIGINT NOT NULL CHECK (paid_paisa >= 0),
    CONSTRAINT uq_expense_splits_expense_user UNIQUE (expense_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_expense_splits_user ON expense_splits(user_id);
