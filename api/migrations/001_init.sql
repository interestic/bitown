-- bitown initial schema

CREATE TABLE IF NOT EXISTS cities (
    slug         TEXT        PRIMARY KEY,
    name         TEXT        NOT NULL,
    country_code CHAR(2)     NOT NULL,
    owner_id     UUID        NULL,
    pop          INTEGER     NOT NULL DEFAULT 1,
    ind          INTEGER     NOT NULL DEFAULT 0,
    tra          INTEGER     NOT NULL DEFAULT 0,
    sec          INTEGER     NOT NULL DEFAULT 0,
    env          INTEGER     NOT NULL DEFAULT 0,
    com          INTEGER     NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS visites_log (
    id           BIGSERIAL   PRIMARY KEY,
    city_slug    TEXT        NOT NULL REFERENCES cities(slug) ON DELETE CASCADE,
    sector       TEXT        NOT NULL,
    visitor_hash TEXT        NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS visites_log_city_slug_idx ON visites_log(city_slug);
CREATE INDEX IF NOT EXISTS visites_log_created_at_idx ON visites_log(created_at);

CREATE TABLE IF NOT EXISTS events_log (
    id           BIGSERIAL   PRIMARY KEY,
    city_slug    TEXT        NOT NULL REFERENCES cities(slug) ON DELETE CASCADE,
    event_type   TEXT        NOT NULL,
    delta        JSONB       NOT NULL DEFAULT '{}',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
