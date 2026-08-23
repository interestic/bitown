-- events_log already exists in 001_init.sql.
-- This migration adds a lookup index for city event feeds.

CREATE INDEX IF NOT EXISTS events_log_city_created_idx
    ON events_log (city_slug, created_at DESC, id DESC);
