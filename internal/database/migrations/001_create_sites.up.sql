
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE sites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    user_id UUID NOT NULL,

    name VARCHAR(100) NOT NULL,

    location VARCHAR(255),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_sites_user
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE,

    CONSTRAINT chk_site_name
        CHECK (LENGTH(BTRIM(name)) BETWEEN 2 AND 100),

    CONSTRAINT chk_site_location
        CHECK (
            location IS NULL
            OR LENGTH(location) <= 255
        )
);

-- Unique site name per user, ignoring case and whitespace
CREATE UNIQUE INDEX idx_sites_user_name_unique
ON sites (user_id, LOWER(BTRIM(name)));

-- Faster lookups for a user's sites
CREATE INDEX idx_sites_user_id
ON sites(user_id);
