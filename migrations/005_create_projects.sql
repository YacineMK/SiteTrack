-- migrate:up
CREATE TABLE projects (
    id                 UUID PRIMARY KEY,
    display_id         TEXT NOT NULL,
    name               TEXT NOT NULL,
    description        TEXT NOT NULL DEFAULT '',
    planned_start_date DATE NOT NULL,
    planned_end_date   DATE NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX projects_display_id_idx ON projects (display_id);
CREATE INDEX projects_name_idx ON projects (name);
CREATE INDEX projects_planned_start_idx ON projects (planned_start_date);
CREATE INDEX projects_planned_end_idx ON projects (planned_end_date);
CREATE INDEX projects_created_at_idx ON projects (created_at DESC);

-- migrate:down
DROP TABLE projects;
