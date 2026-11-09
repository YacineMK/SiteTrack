-- +goose Up
CREATE EXTENSION IF NOT EXISTS pg_trgm;

ALTER TABLE projects ADD COLUMN search tsvector
GENERATED ALWAYS AS (
    to_tsvector('english', coalesce(name, '') || ' ' || coalesce(description, ''))
) STORED;

CREATE INDEX projects_search_idx ON projects USING GIN (search);
CREATE INDEX projects_name_trgm_idx ON projects USING GIN (name gin_trgm_ops);
CREATE INDEX projects_display_id_trgm_idx ON projects USING GIN (display_id gin_trgm_ops);

-- +goose Down
DROP INDEX projects_display_id_trgm_idx;
DROP INDEX projects_name_trgm_idx;
DROP INDEX projects_search_idx;
ALTER TABLE projects DROP COLUMN search;
