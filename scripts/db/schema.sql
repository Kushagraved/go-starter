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

-- ─────────────────────────────────────────────────────────────────────────────
-- Car booking domain
-- ─────────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS branches (
    id         BIGSERIAL PRIMARY KEY,
    name       VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT uq_branches_name UNIQUE (name)
);

CREATE TABLE IF NOT EXISTS cars (
    id         BIGSERIAL PRIMARY KEY,
    branch_id  BIGINT       NOT NULL REFERENCES branches (id),
    -- Category is one of SEDAN, SUV, HATCHBACK. Enforced in the app; a CHECK keeps
    -- the DB honest too.
    category   VARCHAR(32)  NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT chk_cars_category CHECK (category IN ('SEDAN', 'SUV', 'HATCHBACK'))
);

-- Candidate selection filters cars by category then joins pricing per branch.
CREATE INDEX IF NOT EXISTS idx_cars_category_branch ON cars (category, branch_id);

CREATE TABLE IF NOT EXISTS price_configurations (
    id          BIGSERIAL PRIMARY KEY,
    branch_id   BIGINT       NOT NULL REFERENCES branches (id),
    category    VARCHAR(32)  NOT NULL,
    -- Amount is stored in the smallest currency unit (e.g. paise/cents) as an
    -- integer to avoid floating-point money.
    amount      BIGINT       NOT NULL,
    unit        VARCHAR(32)  NOT NULL DEFAULT 'PER_HOUR',
    -- audit_trail is the price-change history: [{updated_at, from_amount, to_amount}].
    audit_trail JSONB        NOT NULL DEFAULT '[]'::jsonb,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT chk_price_configurations_category CHECK (category IN ('SEDAN', 'SUV', 'HATCHBACK')),
    -- One price per (branch, category); the repository upserts on this key.
    CONSTRAINT uq_price_configurations_branch_category UNIQUE (branch_id, category)
);

CREATE TABLE IF NOT EXISTS users (
    id         BIGSERIAL PRIMARY KEY,
    name       VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS bookings (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT       NOT NULL REFERENCES users (id),
    car_id          BIGINT       NOT NULL REFERENCES cars (id),
    -- Half-open window [start_time, end_time). Overlap check: existing.start_time
    -- < new.end_time AND new.start_time < existing.end_time.
    start_time      TIMESTAMPTZ  NOT NULL,
    end_time        TIMESTAMPTZ  NOT NULL,
    amount          BIGINT       NOT NULL,
    unit            VARCHAR(32)  NOT NULL,
    status          VARCHAR(32)  NOT NULL,
    idempotency_key VARCHAR(255) NOT NULL,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT chk_bookings_window CHECK (end_time > start_time),
    -- Same client idempotency key must never create two bookings.
    CONSTRAINT uq_bookings_idempotency_key UNIQUE (idempotency_key)
);

-- Overlap detection and listing both hit (car_id, time window) and (user_id, created_at).
CREATE INDEX IF NOT EXISTS idx_bookings_car_window ON bookings (car_id, start_time, end_time);
CREATE INDEX IF NOT EXISTS idx_bookings_user_created ON bookings (user_id, created_at DESC);

-- Belt-and-braces against overlapping confirmed bookings for the same car at the
-- database level, complementing the app-level FOR UPDATE SKIP LOCKED selection.
-- Requires the btree_gist extension for the equality part on car_id.
CREATE EXTENSION IF NOT EXISTS btree_gist;
ALTER TABLE bookings DROP CONSTRAINT IF EXISTS ex_bookings_no_overlap;
ALTER TABLE bookings
    ADD CONSTRAINT ex_bookings_no_overlap
    EXCLUDE USING gist (
        car_id WITH =,
        tstzrange(start_time, end_time) WITH &&
    )
    WHERE (status = 'CONFIRMED');
